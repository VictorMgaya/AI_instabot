package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	youtubelib "github.com/kkdai/youtube/v2"

	"github.com/Davincible/goinsta/v3"
	"github.com/chromedp/chromedp"
)

// ytVideoMeta holds scraped metadata for the active YouTube Short.
type ytVideoMeta struct {
	URL         string
	VideoID     string
	Title       string
	Description string
	Author      string
}

// ytSourceLoop is the main loop that crawls YouTube Shorts organically by loading
// the main explore feed (youtube.com/shorts), inspecting the playing Shorts,
// and downloading/reposting qualifying videos.
func (myInstabot MyInstabot) ytSourceLoop() {
	rand.Seed(time.Now().UnixNano())
	log.Println("YTSource: Starting organic YouTube Shorts explore crawler...")

	seen := make(map[string]bool)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-blink-features", "AutomationControlled"),
		)...,
	)
	defer cancel()

	ctx, ctxCancel := chromedp.NewContext(allocCtx)
	defer ctxCancel()

	// Initial load of the main Shorts feed
	log.Println("YTSource: Loading Shorts explore feed...")
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.youtube.com/shorts"),
		chromedp.Sleep(8*time.Second),
	)
	if err != nil {
		log.Fatalf("YTSource: Failed to load Shorts explore feed: %v", err)
	}

	for {
		var currentURL string
		err := chromedp.Run(ctx, chromedp.Location(&currentURL))
		if err != nil {
			log.Printf("YTSource: Error getting current location: %v — reloading feed", err)
			_ = chromedp.Run(ctx, chromedp.Navigate("https://www.youtube.com/shorts"), chromedp.Sleep(8*time.Second))
			continue
		}

		// URL format is https://www.youtube.com/shorts/VideoID
		parts := strings.Split(currentURL, "/shorts/")
		if len(parts) < 2 {
			log.Printf("YTSource: Not on a Shorts page (%s), skipping to next...", currentURL)
			nextShort(ctx)
			time.Sleep(3*time.Second)
			continue
		}

		videoID := parts[1]
		if idx := strings.Index(videoID, "?"); idx != -1 {
			videoID = videoID[:idx]
		}

		if seen[videoID] {
			// Already inspected this Short, scroll to the next one
			nextShort(ctx)
			time.Sleep(4*time.Second)
			continue
		}
		seen[videoID] = true

		// Scrape details from the active video player
		var meta *ytVideoMeta
		var detailsRaw map[string]string
		err = chromedp.Run(ctx,
			chromedp.Evaluate(`
				(function() {
					var activeReel = document.querySelector('ytd-reel-video-renderer[is-active]') || 
					                 document.querySelector('ytd-reel-video-renderer[active]') ||
					                 document.querySelector('ytd-reel-video-renderer');
					if (!activeReel) return null;
					
					var titleEl = activeReel.querySelector('h2.title') || 
					              activeReel.querySelector('.title') || 
					              activeReel.querySelector('h2');
					var title = titleEl ? titleEl.innerText : "";
					
					var channelEl = activeReel.querySelector('ytd-channel-name a') || 
					                activeReel.querySelector('#channel-name a') || 
					                activeReel.querySelector('.channel-name') ||
					                activeReel.querySelector('[id="text"] a');
					var channel = channelEl ? channelEl.innerText : "";
					
					var descEl = activeReel.querySelector('#description') || 
					             activeReel.querySelector('.description');
					var desc = descEl ? descEl.innerText : "";
					
					return {
						"title": title,
						"channel": channel,
						"description": desc
					};
				})()
			`, &detailsRaw),
		)

		if err != nil || detailsRaw == nil {
			log.Printf("YTSource: Could not parse details for Short %s", videoID)
			nextShort(ctx)
			time.Sleep(3*time.Second)
			continue
		}

		meta = &ytVideoMeta{
			URL:         currentURL,
			VideoID:     videoID,
			Title:       strings.TrimSpace(detailsRaw["title"]),
			Description: strings.TrimSpace(detailsRaw["description"]),
			Author:      strings.TrimSpace(detailsRaw["channel"]),
		}

		// Clean up titles
		meta.Title = strings.TrimSuffix(meta.Title, " - YouTube")

		// Score metadata
		score := scoreTech(strings.ToLower(meta.Title)) +
			scoreTech(strings.ToLower(meta.Description))
		log.Printf("YTSource: @%s score=%d title=%q",
			meta.Author, score, truncateStr(meta.Title, 80))

		// In pure YT-to-YT mode (no Instagram target), bypass the tech filter entirely
		isPureYtToYt := youtubeMode && !techMode
		if !isPureYtToYt && score < 5 {
			log.Printf("YTSource: Skipping %s — score too low (%d)", videoID, score)
			nextShort(ctx)
			time.Sleep(4*time.Second)
			continue
		}

		log.Printf("YTSource: Downloading qualifying Short %s — %q", videoID, meta.Title)
		videoData, err := ytDownloadVideo(videoID)
		if err != nil {
			log.Printf("YTSource: Download error for %s: %v", videoID, err)
			nextShort(ctx)
			time.Sleep(4*time.Second)
			continue
		}

		// Generate AI caption
		caption := generateAIComment(fmt.Sprintf(
			`You are an energetic tech content creator. Write a short, punchy description (max 30 words) for this tech video repost.

Video title: %q
Creator: %s

Rules:
- Be informative, exciting and enthusiastic
- Sound like a passionate tech enthusiast
- Use 2-4 relevant emojis that match the tech domain (e.g. 🚀 for space, 🤖 for robotics, ⚡ for energy, 🧬 for biotech, 💻 for software)
- NO hashtags at all — zero, none
- Reply with ONLY the description text, nothing else`,
			meta.Title, meta.Author,
		))
		if caption == "" {
			caption = fmt.Sprintf("Mind-blowing tech content! 🚀🔥 via @%s", meta.Author)
		}

		log.Printf("YTSource: Reposting with caption: %q", caption)

		// Post to Instagram
		if techMode {
			if !dev {
				_, err := myInstabot.Insta.Upload(&goinsta.UploadOptions{
					File:    bytes.NewReader(videoData),
					Caption: caption,
				})
				if err != nil {
					log.Printf("YTSource: Instagram upload error: %v", err)
				} else {
					log.Printf("YTSource: Posted to Instagram ✓")
				}
			} else {
				log.Printf("YTSource: [DEV] Would post %d bytes to Instagram", len(videoData))
			}
		}

		// Post to YouTube Shorts as well if -youtube is active
		if youtubeMode {
			localPath := fmt.Sprintf("downloads/yt-source-%s.mp4", videoID)
			if writeErr := writeVideoFile(localPath, videoData); writeErr == nil {
				if !dev {
					if err := uploadToYouTubeShorts(localPath, caption); err != nil {
						log.Printf("YTSource: YouTube upload error: %v", err)
					} else {
						log.Printf("YTSource: Posted to YouTube Shorts ✓")
					}
				} else {
					log.Printf("YTSource: [DEV] Would upload to YouTube Shorts: %s", localPath)
				}
				removeVideoFile(localPath)
			}
		}

		nextShort(ctx)
		time.Sleep(5*time.Second)
	}
}

// nextShort simulates navigating to the next Short video in the feed.
func nextShort(ctx context.Context) {
	var clicked bool
	_ = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				var nextBtn = document.querySelector('[aria-label=\"Next video\"]') || 
				              document.querySelector('#navigation-button-down button') || 
				              document.querySelector('#navigation-button-down');
				if (nextBtn) {
					nextBtn.click();
					return true;
				}
				window.scrollBy(0, window.innerHeight);
				return false;
			})()
		`, &clicked),
	)
}

// writeVideoFile persists video bytes to a local path for browser-based upload.
func writeVideoFile(path string, data []byte) error {
	if err := os.MkdirAll("downloads", 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writeVideoFile: %w", err)
	}
	return nil
}

// removeVideoFile silently deletes a temporary video file.
func removeVideoFile(path string) {
	if err := os.Remove(path); err != nil {
		log.Printf("YTSource: cleanup warning: %v", err)
	}
}

// ytDownloadVideo downloads a YouTube Short video as raw bytes using kkdai/youtube.
func ytDownloadVideo(videoID string) ([]byte, error) {
	client := youtubelib.Client{}

	video, err := client.GetVideo(videoID)
	if err != nil {
		return nil, fmt.Errorf("GetVideo error: %w", err)
	}

	formats := video.Formats.Type("video/mp4")
	if len(formats) == 0 {
		return nil, fmt.Errorf("no mp4 formats available for %s", videoID)
	}

	// Pick stream matching vertical format
	best := formats[0]
	for _, f := range formats {
		if f.Width > 0 && f.Width <= 720 && f.ContentLength > 0 &&
			f.ContentLength < best.ContentLength {
			best = f
		}
	}

	stream, _, err := client.GetStream(video, &best)
	if err != nil {
		return nil, fmt.Errorf("GetStream error: %w", err)
	}
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("read stream error: %w", err)
	}
	return data, nil
}

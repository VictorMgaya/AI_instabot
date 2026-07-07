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

// ytHashtags are the tags explored to find tech Shorts on YouTube.
// They cover the same wide domains as the Instagram keyword scoring system.
var ytHashtags = []string{
	"tech", "technology", "programming", "coding", "software",
	"space", "spacex", "nasa", "robotics", "engineering",
	"developer", "science", "electronics", "3dprinting",
	"futuretech", "computerscience", "hacker", "cybersecurity",
	"gadgets", "automation", "techworld", "quantumphysics",
}

// ytVideoMeta holds scraped metadata for a YouTube Short.
type ytVideoMeta struct {
	URL         string
	VideoID     string
	Title       string
	Description string
	Author      string
}

// ytSourceLoop is the goroutine that continuously crawls YouTube Shorts,
// filters them by tech relevance, downloads qualifying videos, and passes
// them into the repost pipeline. It runs in parallel with the IG Explore loop.
func (myInstabot MyInstabot) ytSourceLoop() {
	rand.Seed(time.Now().UnixNano())
	log.Println("YTSource: YouTube Shorts crawler started")

	seen := make(map[string]bool)

	for {
		tag := ytHashtags[rand.Intn(len(ytHashtags))]
		log.Printf("YTSource: Exploring hashtag feed #%s ...", tag)

		urls, err := ytBrowseShorts(tag)
		if err != nil {
			log.Printf("YTSource: Browse error: %v — retrying in 30s", err)
			time.Sleep(30 * time.Second)
			continue
		}

		log.Printf("YTSource: Found %d Short(s) for #%s", len(urls), tag)

		for _, url := range urls {
			if seen[url] {
				continue
			}
			seen[url] = true

			meta, err := ytGetShortDetails(url)
			if err != nil {
				log.Printf("YTSource: Metadata error for %s: %v", url, err)
				continue
			}

			// Score against our keyword system
			score := scoreTech(strings.ToLower(meta.Title)) +
				scoreTech(strings.ToLower(meta.Description))
			log.Printf("YTSource: @%s score=%d title=%q",
				meta.Author, score, truncateStr(meta.Title, 80))

			// Must meet the same strict threshold as IG Explore
			if score < 5 {
				log.Printf("YTSource: Skipping %s — score too low (%d)", meta.VideoID, score)
				continue
			}

			log.Printf("YTSource: Downloading %s — %q", meta.VideoID, meta.Title)
			videoData, err := ytDownloadVideo(meta.VideoID)
			if err != nil {
				log.Printf("YTSource: Download error for %s: %v", meta.VideoID, err)
				continue
			}

			// Generate AI caption using the YT title as the source
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

			// Post to Instagram only when -tech flag is also active
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

			// Post to YouTube Shorts if -youtube is active
			if youtubeMode {
				localPath := fmt.Sprintf("downloads/yt-source-%s.mp4", meta.VideoID)
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
		}
	}
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

// ytBrowseShorts uses chromedp to navigate a hashtag explore page, scrolls to load more, and scrapes Shorts URLs.
func ytBrowseShorts(tag string) ([]string, error) {
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

	exploreURL := fmt.Sprintf("https://www.youtube.com/hashtag/%s", tag)

	var hrefs []string
	err := chromedp.Run(ctx,
		chromedp.Navigate(exploreURL),
		chromedp.Sleep(4*time.Second),
		// Scroll down to load more content dynamically
		chromedp.Evaluate(`
			(async () => {
				for (let i = 0; i < 4; i++) {
					window.scrollTo(0, document.body.scrollHeight);
					await new Promise(r => setTimeout(r, 1500));
				}
			})()
		`, nil),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('a[href*="/shorts/"]'))
			  .map(a => a.href)
			  .filter(h => h.includes('/shorts/'))
		`, &hrefs),
	)
	if err != nil {
		return nil, fmt.Errorf("chromedp explore error: %w", err)
	}

	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, href := range hrefs {
		// Normalise to base URL
		if idx := strings.Index(href, "?"); idx != -1 {
			href = href[:idx]
		}
		if !seen[href] && strings.Contains(href, "/shorts/") {
			seen[href] = true
			unique = append(unique, href)
		}
	}
	return unique, nil
}

// ytGetShortDetails scrapes the title, description, and author from a Short page.
func ytGetShortDetails(url string) (*ytVideoMeta, error) {
	// Extract video ID from URL e.g. https://www.youtube.com/shorts/AbCdEfGhIjK
	parts := strings.Split(url, "/shorts/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("could not extract video ID from %s", url)
	}
	videoID := parts[1]
	if idx := strings.Index(videoID, "?"); idx != -1 {
		videoID = videoID[:idx]
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("headless", true),
		)...,
	)
	defer cancel()

	ctx, ctxCancel := chromedp.NewContext(allocCtx)
	defer ctxCancel()

	var title, desc, author string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`document.title || ""`, &title),
		chromedp.Evaluate(`
			(function() {
				var d = document.querySelector('meta[name="description"]');
				return d ? d.content : "";
			})()
		`, &desc),
		chromedp.Evaluate(`
			(function() {
				var a = document.querySelector('ytd-channel-name a, #channel-name a');
				return a ? a.innerText : "";
			})()
		`, &author),
	)
	if err != nil {
		return nil, fmt.Errorf("chromedp details error: %w", err)
	}

	// Clean up title (YouTube appends " - YouTube")
	title = strings.TrimSuffix(title, " - YouTube")
	title = strings.TrimSpace(title)

	return &ytVideoMeta{
		URL:         url,
		VideoID:     videoID,
		Title:       title,
		Description: desc,
		Author:      author,
	}, nil
}

// ytDownloadVideo downloads a YouTube Short video as raw bytes using kkdai/youtube.
func ytDownloadVideo(videoID string) ([]byte, error) {
	client := youtubelib.Client{}

	video, err := client.GetVideo(videoID)
	if err != nil {
		return nil, fmt.Errorf("GetVideo error: %w", err)
	}

	// Prefer mp4 video-only stream ≤ 720p for Shorts
	formats := video.Formats.Type("video/mp4")
	if len(formats) == 0 {
		return nil, fmt.Errorf("no mp4 formats available for %s", videoID)
	}

	// Pick the smallest (shortest/lowest res) to keep file sizes manageable
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

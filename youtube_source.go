package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
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
const processedFile = "config/processed_videos.json"

// loadProcessed reads previously processed video IDs from disk.
func loadProcessed() map[string]bool {
	m := make(map[string]bool)
	data, err := os.ReadFile(processedFile)
	if err != nil {
		return m
	}
	var ids []string
	if json.Unmarshal(data, &ids) == nil {
		for _, id := range ids {
			m[id] = true
		}
	}
	log.Printf("YTSource: Loaded %d previously processed video IDs", len(m))
	return m
}

// saveProcessed appends a video ID to the processed list on disk.
func saveProcessed(videoID string) {
	var ids []string
	data, err := os.ReadFile(processedFile)
	if err == nil {
		json.Unmarshal(data, &ids)
	}
	ids = append(ids, videoID)
	out, _ := json.Marshal(ids)
	os.WriteFile(processedFile, out, 0o644)
}

func (myInstabot MyInstabot) ytSourceLoop() {
	rand.Seed(time.Now().UnixNano())
	log.Println("YTSource: Starting organic YouTube Shorts explore crawler...")

	seen := loadProcessed()

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

		// When posting to YouTube, bypass tech filter — upload everything found
		if youtubeMode {
			log.Printf("YTSource: YT upload mode — skipping score filter for %s", videoID)
		} else if score < 5 {
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

		// Generate AI description
		persona := "content creator"
		if techMode && !youtubeMode {
			persona = "tech content creator"
		}

		descPrompt := fmt.Sprintf(
			`You are an energetic %s. Write a short YouTube description (max 300 chars) for this video repost.

Original title: %q
Original description: %q
Creator: %s

Rules:
- Describe what this video is ACTUALLY about — reference the specific content
- Be exciting and enthusiastic
- Use 2-4 relevant emojis
- NO hashtags at all
- Reply with ONLY the description text, nothing else`,
			persona, meta.Title, truncateStr(meta.Description, 200), meta.Author,
		)
		description := generateAIComment(descPrompt)
		if description == "" {
			description = fmt.Sprintf("Check this out! 🔥 via @%s", meta.Author)
		}

		titlePrompt := fmt.Sprintf(
			`Write a short YouTube video title (max 80 chars) for this video.

Original title: %q

Rules:
- Catchy and descriptive
- Match the actual content
- Max 80 characters
- NO emojis, NO hashtags
- Reply with ONLY the title text, nothing else`,
			meta.Title,
		)
		ytTitle := generateAIComment(titlePrompt)
		if ytTitle == "" || len(ytTitle) > 100 {
			ytTitle = meta.Title
			if len(ytTitle) > 95 {
				ytTitle = ytTitle[:92] + "..."
			}
		}

		log.Printf("YTSource: Repost title=%q desc=%q", ytTitle, description)

		// Post to Instagram
		if techMode {
			if !dev {
				_, err := myInstabot.Insta.Upload(&goinsta.UploadOptions{
					File:    bytes.NewReader(videoData),
					Caption: description,
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
					if err := uploadToYouTubeShorts(localPath, ytTitle, description); err != nil {
						log.Printf("YTSource: YouTube upload error: %v", err)
					} else {
						log.Printf("YTSource: Posted to YouTube Shorts ✓")
						saveProcessed(videoID)
						seen[videoID] = true
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

// ffmpegBin returns the path to ffmpeg, checking local dir first then PATH.
func ffmpegBin() string {
	if _, err := os.Stat("./ffmpeg"); err == nil {
		return "./ffmpeg"
	}
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		return "ffmpeg"
	}
	return ""
}

// ytDownloadVideo downloads a YouTube Short video at maximum quality.
// Uses composite download (best video + best audio merged losslessly via ffmpeg -c copy).
// Falls back to best muxed stream if ffmpeg is unavailable.
func ytDownloadVideo(videoID string) ([]byte, error) {
	client := youtubelib.Client{}

	video, err := client.GetVideo(videoID)
	if err != nil {
		return nil, fmt.Errorf("GetVideo error: %w", err)
	}

	ffmpeg := ffmpegBin()
	if ffmpeg != "" {
		return ytDownloadComposite(video, ffmpeg)
	}

	log.Printf("YTSource: ffmpeg not found, falling back to muxed stream for %s", videoID)
	return ytDownloadMuxed(video)
}

// ytDownloadComposite downloads best video-only + audio-only streams and merges via ffmpeg.
// Uses -c copy (stream copy, zero re-encode) to preserve original quality.
func ytDownloadComposite(video *youtubelib.Video, ffmpeg string) ([]byte, error) {
	client := youtubelib.Client{}

	videoFormats := video.Formats.Type("video").AudioChannels(0)
	audioFormats := video.Formats.Type("audio")

	if len(videoFormats) == 0 || len(audioFormats) == 0 {
		return nil, fmt.Errorf("no separate video/audio streams for %s", video.ID)
	}

	videoFormats.Sort()
	audioFormats.Sort()
	bestVideo := videoFormats[0]
	bestAudio := audioFormats[0]

	log.Printf("YTSource: Composite download for %s — video=%s (%dx%d) audio=%s (%dch %dkbps)",
		video.ID, bestVideo.QualityLabel, bestVideo.Width, bestVideo.Height,
		bestAudio.MimeType, bestAudio.AudioChannels, bestAudio.Bitrate/1000)

	tmpDir, err := os.MkdirTemp("", "yt-composite-*")
	if err != nil {
		return nil, fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	vf := filepath.Join(tmpDir, "video.mp4")
	af := filepath.Join(tmpDir, "audio.m4a")
	of := filepath.Join(tmpDir, "output.mp4")

	if err := downloadStream(&client, video, &bestVideo, vf); err != nil {
		return nil, fmt.Errorf("video stream: %w", err)
	}
	if err := downloadStream(&client, video, &bestAudio, af); err != nil {
		return nil, fmt.Errorf("audio stream: %w", err)
	}

	cmd := exec.Command(ffmpeg, "-y", "-i", vf, "-i", af,
		"-c", "copy", "-shortest", "-loglevel", "warning", of)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg merge failed: %w\n%s", err, out)
	}

	data, err := os.ReadFile(of)
	if err != nil {
		return nil, fmt.Errorf("read output: %w", err)
	}
	return data, nil
}

// downloadStream writes a single format stream to a file.
func downloadStream(client *youtubelib.Client, video *youtubelib.Video, format *youtubelib.Format, path string) error {
	stream, _, err := client.GetStream(video, format)
	if err != nil {
		return err
	}
	defer stream.Close()
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, stream)
	return err
}

// ytDownloadMuxed falls back to best muxed (combined audio+video) stream.
func ytDownloadMuxed(video *youtubelib.Video) ([]byte, error) {
	client := youtubelib.Client{}

	formats := video.Formats.WithAudioChannels().Type("video/mp4")
	if len(formats) == 0 {
		formats = video.Formats.WithAudioChannels()
	}
	if len(formats) == 0 {
		return nil, fmt.Errorf("no muxed streams available for %s", video.ID)
	}

	formats.Sort()
	best := formats[0]

	log.Printf("YTSource: Muxed download for %s — %s (%dx%d)", video.ID, best.QualityLabel, best.Width, best.Height)

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

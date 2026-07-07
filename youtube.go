package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const youtubeCookiesFile = "config/youtube-cookies.txt"
const youtubeUploadEndpoint = "https://upload.youtube.com/upload/studio"
const youtubeOrigin = "https://www.youtube.com"

type youtubeCookie struct {
	Domain   string
	Path     string
	Secure   bool
	Expiry   string
	Name     string
	Value    string
	HttpOnly bool
}

// ytEssentialCookies — the only session cookies needed for auth.
var ytEssentialCookies = map[string]bool{
	"SID":               true,
	"HSID":              true,
	"SSID":              true,
	"APISID":            true,
	"SAPISID":           true,
	"LOGIN_INFO":        true,
	"__Secure-1PSID":    true,
	"__Secure-3PSID":    true,
	"__Secure-1PAPISID": true,
	"__Secure-3PAPISID": true,
	"__Secure-1PSIDTS":  true,
	"__Secure-3PSIDTS":  true,
	"SIDCC":             true,
}

// parseYoutubeCookies parses a Netscape-format cookies file.
func parseYoutubeCookies(path string) ([]youtubeCookie, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cookies []youtubeCookie
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 7 {
			continue
		}
		cookies = append(cookies, youtubeCookie{
			Domain:   parts[0],
			Path:     parts[2],
			Secure:   parts[3] == "TRUE",
			Expiry:   parts[4],
			Name:     parts[5],
			Value:    parts[6],
			HttpOnly: strings.HasPrefix(parts[0], "#HttpOnly_"),
		})
	}
	return cookies, nil
}

// buildCookieHeader returns a Cookie header string from essential session cookies.
func buildCookieHeader(cookies []youtubeCookie) string {
	var parts []string
	for _, c := range cookies {
		domain := strings.TrimPrefix(c.Domain, "#HttpOnly_")
		if !strings.Contains(domain, "youtube.com") && !strings.Contains(domain, "google.com") {
			continue
		}
		if !ytEssentialCookies[c.Name] {
			continue
		}
		parts = append(parts, c.Name+"="+c.Value)
	}
	return strings.Join(parts, "; ")
}

// computeSAPISIDHASH computes the Authorization header value required by YouTube Studio.
func computeSAPISIDHASH(sapisid string) string {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	h := sha1.New()
	h.Write([]byte(ts + " " + sapisid + " " + youtubeOrigin))
	return fmt.Sprintf("SAPISIDHASH %s_%x", ts, h.Sum(nil))
}

// privacyStatus returns PUBLIC in live mode, PRIVATE in dev mode.
func privacyStatus() string {
	if dev {
		return "PRIVATE"
	}
	return "PUBLIC"
}

// uploadToYouTubeShorts uploads a local video file to YouTube Shorts using
// YouTube Studio's internal resumable upload API via direct HTTP — no browser needed.
func uploadToYouTubeShorts(videoPath string, description string) error {
	log.Printf("YouTube: Starting HTTP upload for: %s", videoPath)

	cookies, err := parseYoutubeCookies(youtubeCookiesFile)
	if err != nil {
		return fmt.Errorf("failed to read YouTube cookies: %w", err)
	}

	// Find SAPISID for auth
	var sapisid string
	for _, c := range cookies {
		if c.Name == "SAPISID" {
			sapisid = c.Value
			break
		}
	}
	if sapisid == "" {
		for _, c := range cookies {
			if c.Name == "__Secure-3PAPISID" {
				sapisid = c.Value
				break
			}
		}
	}
	if sapisid == "" {
		return fmt.Errorf("SAPISID not found in cookies — please re-export your YouTube cookies")
	}

	title := description
	if len(title) > 95 {
		title = title[:92] + "..."
	}

	cookieHeader := buildCookieHeader(cookies)
	authHeader := computeSAPISIDHASH(sapisid)

	// Read video file
	videoData, err := os.ReadFile(videoPath)
	if err != nil {
		return fmt.Errorf("failed to read video file: %w", err)
	}
	log.Printf("YouTube: Video size: %d bytes", len(videoData))

	// Step 1: Initiate resumable upload session
	log.Println("YouTube: Requesting upload session from YouTube Studio...")
	frontendID := fmt.Sprintf("studio-%d-%d", time.Now().UnixNano(), rand.Intn(99999))
	initBody := map[string]interface{}{
		"frontendUploadId": frontendID,
		"initialMetadata": map[string]interface{}{
			"title":       map[string]string{"newTitle": title},
			"description": map[string]string{"newDescription": description},
			"privacy":     map[string]string{"newPrivacy": privacyStatus()},
			"draftState":  map[string]interface{}{"isDraft": false},
		},
	}
	initJSON, _ := json.Marshal(initBody)

	req, err := http.NewRequest("POST", youtubeUploadEndpoint, bytes.NewReader(initJSON))
	if err != nil {
		return fmt.Errorf("failed to build init request: %w", err)
	}
	setCommonHeaders(req, cookieHeader, authHeader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Upload-Protocol", "resumable")
	req.Header.Set("X-Goog-Upload-Command", "start")
	req.Header.Set("X-Goog-Upload-Header-Content-Length", strconv.Itoa(len(videoData)))
	req.Header.Set("X-Goog-Upload-Header-Content-Type", "video/mp4")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("upload session init failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload init returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	uploadURL := resp.Header.Get("X-Goog-Upload-URL")
	if uploadURL == "" {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("no X-Goog-Upload-URL in response — YouTube may have changed API: %s", string(body))
	}
	log.Printf("YouTube: Upload session created. Uploading %d bytes...", len(videoData))

	// Step 2: Upload video bytes to the resumable URL
	uploadReq, err := http.NewRequest("PUT", uploadURL, bytes.NewReader(videoData))
	if err != nil {
		return fmt.Errorf("failed to build upload request: %w", err)
	}
	setCommonHeaders(uploadReq, cookieHeader, authHeader)
	uploadReq.Header.Set("Content-Type", "video/mp4")
	uploadReq.Header.Set("X-Goog-Upload-Protocol", "resumable")
	uploadReq.Header.Set("X-Goog-Upload-Command", "upload, finalize")
	uploadReq.Header.Set("X-Goog-Upload-Offset", "0")
	uploadReq.ContentLength = int64(len(videoData))

	uploadClient := &http.Client{Timeout: 10 * time.Minute}
	uploadResp, err := uploadClient.Do(uploadReq)
	if err != nil {
		return fmt.Errorf("video upload request failed: %w", err)
	}
	defer uploadResp.Body.Close()

	respBody, _ := io.ReadAll(uploadResp.Body)
	if uploadResp.StatusCode != 200 && uploadResp.StatusCode != 201 {
		return fmt.Errorf("video upload returned HTTP %d: %s", uploadResp.StatusCode, string(respBody))
	}

	// Parse video ID from response if present
	var result map[string]interface{}
	if json.Unmarshal(respBody, &result) == nil {
		if vid, ok := result["videoId"].(string); ok && vid != "" {
			log.Printf("YouTube: Upload successful! Video ID: %s ✓", vid)
			return nil
		}
	}
	log.Printf("YouTube: Upload completed ✓")
	return nil
}

// setCommonHeaders sets request headers shared across all YouTube Studio API calls.
func setCommonHeaders(req *http.Request, cookieHeader, authHeader string) {
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Cookie", cookieHeader)
	req.Header.Set("Origin", youtubeOrigin)
	req.Header.Set("Referer", "https://studio.youtube.com/")
	req.Header.Set("X-Origin", youtubeOrigin)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")
	req.Header.Set("X-Goog-Authuser", "0")
}

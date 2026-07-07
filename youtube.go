package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

const youtubeCookiesFile = "config/youtube-cookies.txt"

type youtubeCookie struct {
	Domain  string
	Path    string
	Secure  bool
	Expires int64
	Name    string
	Value   string
}

// parseYoutubeCookies reads and parses cookies in Netscape format.
func parseYoutubeCookies(path string) ([]youtubeCookie, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cookies []youtubeCookie
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 7 {
			continue
		}
		domain := fields[0]
		pathVal := fields[2]
		secure := fields[3] == "TRUE"
		expires, _ := strconv.ParseInt(fields[4], 10, 64)
		name := fields[5]
		value := fields[6]

		cookies = append(cookies, youtubeCookie{
			Domain:  domain,
			Path:    pathVal,
			Secure:  secure,
			Expires: expires,
			Name:    name,
			Value:   value,
		})
	}
	return cookies, nil
}

// setYoutubeCookies injects YouTube cookies into the browser context.
func setYoutubeCookies(ctx context.Context, cookies []youtubeCookie) error {
	for _, c := range cookies {
		// Only set cookies relevant to youtube.com
		if !strings.Contains(c.Domain, "youtube.com") {
			continue
		}
		expr := fmt.Sprintf(`document.cookie="%s=%s; domain=%s; path=%s; secure=%t; samesite=lax"`,
			c.Name, c.Value, c.Domain, c.Path, c.Secure)
		var out string
		if err := chromedp.Run(ctx, chromedp.EvaluateAsDevTools(expr, &out)); err != nil {
			return fmt.Errorf("setting cookie %s: %w", c.Name, err)
		}
	}
	return nil
}

// uploadToYouTubeShorts uploads a local video to YouTube Shorts using chromedp.
func uploadToYouTubeShorts(videoPath string, description string) error {
	log.Printf("YouTube: Starting upload for video: %s", videoPath)

	cookies, err := parseYoutubeCookies(youtubeCookiesFile)
	if err != nil {
		return fmt.Errorf("failed to read YouTube cookies: %w. Please export cookies to %s", err, youtubeCookiesFile)
	}

	// Prepare short title: max 95 chars (YouTube limit is 100)
	title := description
	if len(title) > 95 {
		title = title[:92] + "..."
	}

	// Anti-detection script
	antiDetectJS := `
		Object.defineProperty(navigator, 'webdriver', {get: () => undefined});
		Object.defineProperty(navigator, 'languages', {get: () => ['en-US', 'en']});
		Object.defineProperty(navigator, 'plugins', {get: () => [1,2,3,4,5]});
		window.chrome = {runtime: {}};
	`

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("disable-blink-features", "AutomationControlled"),
			chromedp.Flag("disable-web-security", false),
			chromedp.Flag("disable-features", "TranslateUI"),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
		)...,
	)
	defer allocCancel()

	ctx, ctxCancel := chromedp.NewContext(allocCtx)
	defer ctxCancel()

	// Inject anti-detection
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		_, err := page.AddScriptToEvaluateOnNewDocument(antiDetectJS).Do(ctx)
		return err
	})); err != nil {
		return fmt.Errorf("failed to inject anti-detect: %w", err)
	}

	// 1. Go to robots.txt to set cookies
	log.Println("YouTube: Initializing browser session...")
	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.youtube.com/robots.txt"),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		return fmt.Errorf("failed to load initial page: %w", err)
	}

	// 2. Set cookies
	if err := setYoutubeCookies(ctx, cookies); err != nil {
		return fmt.Errorf("failed to inject cookies: %w", err)
	}

	// 3. Navigate to YouTube Studio dashboard
	log.Println("YouTube: Loading YouTube Studio...")
	var currentURL string
	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://studio.youtube.com"),
		chromedp.Sleep(5*time.Second),
		chromedp.Location(&currentURL),
	); err != nil {
		return fmt.Errorf("failed to load studio page: %w", err)
	}

	if strings.Contains(currentURL, "accounts.google.com") {
		return fmt.Errorf("login failed: cookies might have expired. Please re-export your cookies to %s", youtubeCookiesFile)
	}

	log.Println("YouTube: Login verified successfully")

	// 4. Click 'Create' then 'Upload videos'
	log.Println("YouTube: Opening upload wizard...")
	var openWizardSuccess bool
	err = chromedp.Run(ctx,
		chromedp.WaitVisible(`[id="create-icon"]`, chromedp.ByQuery),
		chromedp.Evaluate(`
			(function() {
				var createBtn = document.querySelector('#create-icon') || document.querySelector('[id="create-icon"]');
				if (!createBtn) return false;
				createBtn.click();
				return true;
			})()
		`, &openWizardSuccess),
		chromedp.Sleep(1*time.Second),
	)
	if err != nil || !openWizardSuccess {
		return fmt.Errorf("failed to click Create button: %w", err)
	}

	var clickUploadSuccess bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				var items = Array.from(document.querySelectorAll('paper-item, ytcp-text-menu-item, tp-yt-paper-item'));
				var uploadItem = items.find(el => el.innerText.includes('Upload videos'));
				if (!uploadItem) return false;
				uploadItem.click();
				return true;
			})()
		`, &clickUploadSuccess),
		chromedp.Sleep(3*time.Second),
	)
	if err != nil || !clickUploadSuccess {
		return fmt.Errorf("failed to click Upload videos option: %w", err)
	}

	// 5. Select video file
	log.Printf("YouTube: Selecting file %s...", videoPath)
	err = chromedp.Run(ctx,
		chromedp.WaitReady(`input[type="file"]`, chromedp.ByQuery),
		chromedp.SetUploadFiles(`input[type="file"]`, []string{videoPath}, chromedp.ByQuery),
		chromedp.Sleep(5*time.Second),
	)
	if err != nil {
		return fmt.Errorf("failed to select file: %w", err)
	}

	// 6. Enter Metadata
	log.Println("YouTube: Entering title and description...")
	fillMetadataJS := fmt.Sprintf(`
		(function() {
			var titleBox = document.querySelector('#title-textarea #textbox');
			var descBox = document.querySelector('#description-textarea #textbox');
			if (titleBox) {
				titleBox.innerText = %q;
				titleBox.dispatchEvent(new Event('input', { bubbles: true }));
			}
			if (descBox) {
				descBox.innerText = %q;
				descBox.dispatchEvent(new Event('input', { bubbles: true }));
			}
			return !!(titleBox && descBox);
		})()
	`, title, description)

	var success bool
	err = chromedp.Run(ctx,
		chromedp.WaitVisible(`#title-textarea #textbox`, chromedp.ByQuery),
		chromedp.Evaluate(fillMetadataJS, &success),
	)
	if err != nil || !success {
		return fmt.Errorf("failed to fill video metadata details: %w", err)
	}
	chromedp.Run(ctx, chromedp.Sleep(2*time.Second))

	// 7. Select 'Not made for kids' (mandatory)
	log.Println("YouTube: Setting audience details...")
	var kidsSuccess bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				var radio = document.querySelector('[name="VIDEO_MADE_FOR_KIDS_NOT_MFK"]') || document.querySelector('paper-radio-button[name="VIDEO_MADE_FOR_KIDS_NOT_MFK"]') || document.querySelector('tp-yt-paper-radio-button[name="VIDEO_MADE_FOR_KIDS_NOT_MFK"]');
				if (!radio) return false;
				radio.click();
				return true;
			})()
		`, &kidsSuccess),
		chromedp.Sleep(1*time.Second),
	)
	if err != nil || !kidsSuccess {
		return fmt.Errorf("failed to select Not Made for Kids: %w", err)
	}

	// 8. Wizard Step 1 -> Step 2
	log.Println("YouTube: Advancing wizard (Details -> Video Elements)...")
	var next1Success bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				var btn = document.querySelector('[id="next-button"]');
				if (!btn) return false;
				btn.click();
				return true;
			})()
		`, &next1Success),
		chromedp.Sleep(2*time.Second),
	)
	if err != nil || !next1Success {
		return fmt.Errorf("failed to click Next on Details step: %w", err)
	}

	// 9. Wizard Step 2 -> Step 3
	log.Println("YouTube: Advancing wizard (Video Elements -> Checks)...")
	var next2Success bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				var btn = document.querySelector('[id="next-button"]');
				if (!btn) return false;
				btn.click();
				return true;
			})()
		`, &next2Success),
		chromedp.Sleep(2*time.Second),
	)
	if err != nil || !next2Success {
		return fmt.Errorf("failed to click Next on Video Elements step: %w", err)
	}

	// 10. Wizard Step 3 -> Step 4 (Visibility)
	log.Println("YouTube: Advancing wizard (Checks -> Visibility)...")
	var next3Success bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				var btn = document.querySelector('[id="next-button"]');
				if (!btn) return false;
				btn.click();
				return true;
			})()
		`, &next3Success),
		chromedp.Sleep(2*time.Second),
	)
	if err != nil || !next3Success {
		return fmt.Errorf("failed to click Next on Checks step: %w", err)
	}

	// 11. Set Visibility and Publish
	var visibilityVal string
	if dev {
		log.Println("YouTube: [DEV MODE] Setting visibility to PRIVATE")
		visibilityVal = "PRIVATE"
	} else {
		log.Println("YouTube: Setting visibility to PUBLIC")
		visibilityVal = "PUBLIC"
	}

	var visibilitySuccess bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(fmt.Sprintf(`
			(function() {
				var radio = document.querySelector('[name="%s"]') || document.querySelector('paper-radio-button[name="%s"]') || document.querySelector('tp-yt-paper-radio-button[name="%s"]');
				if (!radio) return false;
				radio.click();
				return true;
			})()
		`, visibilityVal, visibilityVal, visibilityVal), &visibilitySuccess),
		chromedp.Sleep(1*time.Second),
	)
	if err != nil || !visibilitySuccess {
		return fmt.Errorf("failed to set visibility: %w", err)
	}

	// Click Done / Save / Publish
	log.Println("YouTube: Publishing video...")
	var doneSuccess bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				var btn = document.querySelector('[id="done-button"]');
				if (!btn) return false;
				btn.click();
				return true;
			})()
		`, &doneSuccess),
		// Wait for publish popup or short delay to allow save request to dispatch
		chromedp.Sleep(7*time.Second),
	)
	if err != nil || !doneSuccess {
		return fmt.Errorf("failed to click publish button: %w", err)
	}

	log.Printf("YouTube: Upload and publish completed successfully ✓")
	return nil
}

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"

	"github.com/Davincible/goinsta/v3"
	"github.com/spf13/viper"
)

const instagramCookiesFile = "config/instagram-cookies.txt"

type parsedCookie struct {
	Domain   string
	Path     string
	Secure   bool
	HTTPOnly bool
	Expires  int64
	Name     string
	Value    string
}

// parseNetscapeCookies parses a Netscape-format cookie file (same format as youtube-cookies.txt).
func parseNetscapeCookies(path string) ([]parsedCookie, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cookies []parsedCookie
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r\n ")
		if line == "" || (line[0] == '#' && !strings.HasPrefix(line, "#HttpOnly_")) {
			continue
		}

		httpOnly := false
		if strings.HasPrefix(line, "#HttpOnly_") {
			httpOnly = true
			line = line[len("#HttpOnly_"):]
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 7 {
			continue
		}

		expires, _ := strconv.ParseInt(parts[4], 10, 64)
		cookies = append(cookies, parsedCookie{
			Domain:   parts[0],
			Path:     parts[2],
			Secure:   parts[3] == "TRUE",
			HTTPOnly: httpOnly,
			Expires:  expires,
			Name:     parts[5],
			Value:    parts[6],
		})
	}
	return cookies, nil
}

// loginViaPlaywrightFallback launches a headful Chrome with Instagram cookies,
// waits for the user to complete any security challenges, then extracts the
// session and creates a goinsta session file.
func loginViaPlaywrightFallback() error {
	logPrefix(PrefixInsta, "Cookie auth fallback — launching browser...")

	if _, err := os.Stat(instagramCookiesFile); os.IsNotExist(err) {
		return fmt.Errorf("Instagram cookies file not found at %s", instagramCookiesFile)
	}

	cookies, err := parseNetscapeCookies(instagramCookiesFile)
	if err != nil {
		return fmt.Errorf("failed to parse cookies: %w", err)
	}
	logPrefix(PrefixInsta, "Loaded %d cookies from %s", len(cookies), instagramCookiesFile)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", false),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("disable-blink-features", "AutomationControlled"),
		)...,
	)
	defer cancel()

	ctx, ctxCancel := chromedp.NewContext(allocCtx)
	defer ctxCancel()

	ctx, timeoutCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer timeoutCancel()

	var sessionCookies []*network.Cookie
	var wwwClaim string

	// Listen for network responses to extract X-Ig-Www-Claim
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *network.EventResponseReceived:
			if strings.Contains(ev.Response.URL, "instagram.com") {
				for k, v := range ev.Response.Headers {
					if strings.EqualFold(k, "x-ig-www-claim") {
						wwwClaim = fmt.Sprintf("%v", v)
					}
				}
			}
		}
	})

	// Navigate to Instagram, set cookies, then reload
	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.instagram.com/"),
		chromedp.Sleep(2*time.Second),

		chromedp.ActionFunc(func(ctx context.Context) error {
			for _, c := range cookies {
				expr := cdp.TimeSinceEpoch(time.Unix(c.Expires, 0))
				if err := network.SetCookie(c.Name, c.Value).
					WithDomain(c.Domain).
					WithPath(c.Path).
					WithSecure(c.Secure).
					WithHTTPOnly(c.HTTPOnly).
					WithExpires(&expr).
					Do(ctx); err != nil {
					logPrefix(PrefixInsta, "Cookie set warning (%s): %v", c.Name, err)
				}
			}
			return nil
		}),

		chromedp.Reload(),
		chromedp.Sleep(3*time.Second),
	); err != nil {
		return fmt.Errorf("browser navigation failed: %w", err)
	}

	// Check if we're on the login/challenge page
	var pageURL string
	chromedp.Run(ctx, chromedp.Location(&pageURL))

	isLoginPage := strings.Contains(pageURL, "accounts/login") ||
		strings.Contains(pageURL, "challenge") ||
		strings.Contains(pageURL, "accounts.google.com")

	if isLoginPage {
		logPrefix(PrefixInsta, "Challenge/login detected — waiting for you to complete it in the browser window...")
		logPrefix(PrefixInsta, "After logging in, the bot will automatically detect it and save the session.")

		// Poll until we reach the Instagram feed
		pollStart := time.Now()
		for time.Since(pollStart) < 4*time.Minute {
			chromedp.Run(ctx, chromedp.Location(&pageURL))
			if !strings.Contains(pageURL, "accounts/login") &&
				!strings.Contains(pageURL, "challenge") &&
				!strings.Contains(pageURL, "accounts.google.com") &&
				strings.Contains(pageURL, "instagram.com") {
				break
			}
			time.Sleep(2 * time.Second)
		}
	} else {
		logPrefix(PrefixInsta, "Cookies auto-logged in — extracting session...")
	}

	// Give a moment for the claim header to arrive
	time.Sleep(2 * time.Second)

	// Extract all cookies via CDP
	chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		sessionCookies, err = network.GetCookies().Do(ctx)
		return err
	}))

	logPrefix(PrefixInsta, "Extracted %d cookies from browser", len(sessionCookies))

	// Build goinsta session from cookies
	if err := buildGoinstaSession(sessionCookies, wwwClaim); err != nil {
		return fmt.Errorf("failed to build session from cookies: %w", err)
	}

	logPrefix(PrefixInsta, "Session saved — you can close the browser window")
	return nil
}

// buildGoinstaSession constructs a goinsta session from browser cookies and saves it.
func buildGoinstaSession(cdpCookies []*network.Cookie, wwwClaim string) error {
	var sessionid, dsUserID, rur, mid string

	for _, c := range cdpCookies {
		switch c.Name {
		case "sessionid":
			sessionid = c.Value
		case "ds_user_id":
			dsUserID = c.Value
		case "rur":
			rur = c.Value
		case "mid":
			mid = c.Value
		}
	}

	if sessionid == "" || dsUserID == "" {
		return fmt.Errorf("missing required cookies (sessionid=%q, ds_user_id=%q)", sessionid, dsUserID)
	}

	userID, _ := strconv.ParseInt(dsUserID, 10, 64)
	uuid := newUUID()
	deviceID := "android-" + randHex(16)

	// Construct the Bearer token: IGT:2:{base64({ds_user_id, sessionid})}
	tokenPayload, _ := json.Marshal(map[string]string{
		"ds_user_id": dsUserID,
		"sessionid":  sessionid,
	})
	bearerToken := "Bearer IGT:2:" + base64.StdEncoding.EncodeToString(tokenPayload)

	headerOpts := map[string]string{
		"Authorization":    bearerToken,
		"Ig-U-Ds-User-Id": dsUserID,
		"Ig-U-Rur":        rur,
	}
	if mid != "" {
		headerOpts["X-Mid"] = mid
	}
	if wwwClaim != "" {
		headerOpts["X-Ig-Www-Claim"] = wwwClaim
	}

	config := goinsta.ConfigFile{
		ID:        userID,
		User:      viper.GetString("user.instagram.username"),
		DeviceID:  deviceID,
		FamilyID:  newUUID(),
		UUID:      uuid,
		RankToken: dsUserID + "_" + uuid,
		Token:     "",
		PhoneID:   newUUID(),
		Device:    goinsta.GalaxyS10,
		HeaderOptions: headerOpts,
		Account: &goinsta.Account{
			ID:       userID,
			Username: viper.GetString("user.instagram.username"),
		},
	}

	insta, err := goinsta.ImportConfig(config, true)
	if err != nil {
		return fmt.Errorf("ImportConfig: %w", err)
	}

	instabot.Insta = insta
	if err := insta.Export("./goinsta-session"); err != nil {
		return fmt.Errorf("Export: %w", err)
	}

	return nil
}

// newUUID generates a v4 UUID string.
func newUUID() string {
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		randHex(8), randHex(4), randHex(4), randHex(4), randHex(12))
}

var cookieFallbackAttempted bool

// tryCookieFallback attempts the cookie-based auth once per process lifecycle
// when Instagram API calls fail with feedback_required.
func tryCookieFallback() {
	if cookieFallbackAttempted {
		return
	}
	cookieFallbackAttempted = true
	logPrefix(PrefixInsta, "Rate-limited — trying cookie auth fallback...")
	if err := loginViaPlaywrightFallback(); err != nil {
		logPrefix(PrefixInsta, "Cookie fallback failed: %v", err)
		return
	}
	logPrefix(PrefixInsta, "Cookie fallback succeeded — session refreshed")
}

// randHex returns n random hex digits.
func randHex(n int) string {
	const hex = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = hex[time.Now().UnixNano()%16]
		time.Sleep(1) // ensure uniqueness
	}
	return string(b)
}

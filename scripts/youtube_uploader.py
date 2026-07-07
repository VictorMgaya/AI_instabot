#!/usr/bin/env python3
import sys
import os
import time
import argparse
from playwright.sync_api import sync_playwright

def parse_netscape_cookies(filepath):
    cookies = []
    if not os.path.exists(filepath):
        print(f"Error: Cookie file not found at {filepath}", file=sys.stderr)
        sys.exit(1)
        
    with open(filepath, "r", encoding="utf-8") as f:
        for line in f:
            line = line.replace("\r", "").strip()
            if not line or line.startswith("#") and not line.startswith("#HttpOnly_"):
                continue
                
            is_httponly = False
            if line.startswith("#HttpOnly_"):
                is_httponly = True
                line = line[len("#HttpOnly_"):]
                
            parts = line.split("\t")
            if len(parts) < 7:
                continue
                
            name = parts[5].strip()
            domain = parts[0].strip()
            path = parts[2].strip()
            secure = parts[3].strip().upper() == "TRUE"
            
            try:
                expires = float(parts[4].strip())
            except ValueError:
                expires = -1
                
            value = parts[6].strip()
            
            cookie = {
                "name": name,
                "value": value,
                "domain": domain,
                "path": path,
                "secure": secure,
                "httpOnly": is_httponly,
            }
            if expires > 0:
                cookie["expires"] = expires
                
            cookies.append(cookie)
            
    return cookies

def main():
    parser = argparse.ArgumentParser(description="Upload videos to YouTube Shorts via Playwright")
    parser.add_argument("--video", required=True, help="Path to video file")
    parser.add_argument("--title", required=True, help="Title of video")
    parser.add_argument("--description", required=True, help="Description of video")
    parser.add_argument("--cookies", required=True, help="Path to netscape cookies file")
    parser.add_argument("--dev", action="store_true", help="Dev mode (upload as Private)")
    args = parser.parse_args()

    if not os.path.exists(args.video):
        print(f"Error: Video file not found: {args.video}", file=sys.stderr)
        sys.exit(1)

    print("YouTube Uploader: Parsing cookies...")
    cookies = parse_netscape_cookies(args.cookies)
    if not cookies:
        print("Error: No authentication cookies parsed from cookie file.", file=sys.stderr)
        sys.exit(1)
        
    print(f"YouTube Uploader: Parsed {len(cookies)} cookies.")
    os.makedirs("downloads", exist_ok=True)
    
    with sync_playwright() as p:
        print("YouTube Uploader: Launching browser...")
        # Launch headful browser so the user can see and complete security challenges
        browser = p.chromium.launch(headless=False)
        context = browser.new_context(
            user_agent="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
            viewport={"width": 1280, "height": 720}
        )
        context.add_cookies(cookies)
        
        page = context.new_page()
        
        try:
            print("YouTube Uploader: Navigating to YouTube Studio...")
            page.goto("https://studio.youtube.com", timeout=60000)
            
            # If Google asks for code/2FA verification, wait for the user to complete it
            start_time = time.time()
            challenge_detected = False
            while "accounts.google.com" in page.url:
                if not challenge_detected:
                    print("YouTube Uploader: [ACTION REQUIRED] Google login verification / security challenge detected!", file=sys.stderr)
                    print("YouTube Uploader: Please complete the verification prompt in the browser window...", file=sys.stderr)
                    challenge_detected = True
                
                elapsed = time.time() - start_time
                if elapsed > 120:
                    print("Error: Verification/login challenge timed out (2 minutes).", file=sys.stderr)
                    page.screenshot(path="downloads/playwright_expired_cookies.png")
                    sys.exit(1)
                
                page.wait_for_timeout(3000)
                
            if challenge_detected:
                print("YouTube Uploader: Challenge verification completed successfully!")
                
            page.wait_for_timeout(5000)
            
            print("YouTube Uploader: Logged in successfully. Dismissing any welcome/onboarding dialogs...")
            # Dismiss any welcome overlays or Swahili onboarding steps (e.g. "Inayofuata" / Next cards)
            for i in range(3):
                page.evaluate("""
                    (function() {
                        var buttons = Array.from(document.querySelectorAll('ytcp-button, paper-button, button, ytcp-button-shape'));
                        var dismissKeywords = ['got it', 'ok', 'close', 'funga', 'inayofuata', 'nimekuelewa', 'dismiss', 'done', 'next', 'save', 'agree'];
                        buttons.forEach(btn => {
                            var text = btn.innerText.toLowerCase();
                            if (dismissKeywords.some(kw => text.includes(kw))) {
                                btn.click();
                            }
                        });
                    })()
                """)
                page.wait_for_timeout(1000)

            print("YouTube Uploader: Opening upload wizard...")
            # Try direct upload arrow icon in top right first, using force=True to bypass overlay backdrops
            upload_btn = page.locator('#upload-button, [id="upload-button"], [aria-label*="upload"], [aria-label*="Pakia"]')
            if upload_btn.is_visible():
                upload_btn.click(force=True)
            else:
                # Fallback to Buni/Create button dropdown click
                create_btn = page.locator('#create-icon, [id="create-icon"], ytcp-button:has-text("Buni"), ytcp-button:has-text("Create")')
                create_btn.click(force=True)
                page.wait_for_timeout(1500)
                # Click the first item in the dropdown list (always 'Upload videos' or 'Pakia video')
                page.locator('paper-item, ytcp-text-menu-item, tp-yt-paper-item').first.click(force=True)
                
            page.wait_for_selector('input[type="file"]', state="attached", timeout=30000)
            print("YouTube Uploader: Selecting video file...")
            page.set_input_files('input[type="file"]', args.video)
            
            print("YouTube Uploader: Waiting for video upload and processing to complete...")
            page.wait_for_timeout(30000)
            
            print("YouTube Uploader: Waiting for metadata inputs to load...")
            page.wait_for_selector('#title-textarea #textbox', timeout=60000)
            page.wait_for_timeout(2000)
            
            # Set Title (YouTube strictly limits titles to 100 characters; truncate to 95 for safety)
            clean_title = args.title
            if len(clean_title) > 95:
                clean_title = clean_title[:92] + "..."

            title_input = page.locator('#title-textarea #textbox')
            title_input.clear()
            title_input.fill(clean_title)
            
            # Set Description
            desc_input = page.locator('#description-textarea #textbox')
            desc_input.clear()
            desc_input.fill(args.description)
            
            print("YouTube Uploader: Setting audience details (Not Made for Kids)...")
            page.evaluate("""
                (() => {
                    var done = false;
                    var sel = 'tp-yt-paper-radio-button[name="VIDEO_MADE_FOR_KIDS_NOT_MFK"], paper-radio-button[name="VIDEO_MADE_FOR_KIDS_NOT_MFK"], [name="VIDEO_MADE_FOR_KIDS_NOT_MFK"]';
                    document.querySelectorAll(sel).forEach(function(r) {
                        if (!done) { r.click(); done = true; }
                    });
                    if (!done) {
                        document.querySelectorAll('ytkc-radio-button, [role=radio]').forEach(function(r) {
                            var txt = (r.textContent || '').toLowerCase();
                            if (!done && (txt.includes('not made for kids') || txt.includes("no, it's not"))) {
                                r.click(); done = true;
                            }
                        });
                    }
                    if (!done) console.warn('Could not click Not Made for Kids');
                })()
            """)
            page.wait_for_timeout(2000)
            
            # Navigate through all upload wizard steps dynamically.
            # YouTube may hide #next-button on some steps; use JS click as fallback.
            max_steps = 5
            for step_idx in range(1, max_steps + 1):
                # If we're already on the visibility step, stop navigating
                public_radio = page.locator('tp-yt-paper-radio-button[name="PUBLIC"], paper-radio-button[name="PUBLIC"]')
                if public_radio.is_visible():
                    print(f"YouTube Uploader: Already on visibility step (step {step_idx}).")
                    break

                print(f"YouTube Uploader: Navigating step {step_idx}...")
                
                # Try clicking #next-button normally
                next_btn = page.locator('#next-button')
                if next_btn.is_visible():
                    next_btn.click(force=True)
                elif next_btn.count() > 0:
                    # Button exists but hidden — click via JavaScript
                    page.evaluate('document.querySelector("#next-button").click()')
                else:
                    # No next button at all; assume we're on the final step
                    print(f"YouTube Uploader: No next button found on step {step_idx}.")
                    break
                    
                page.wait_for_timeout(3000)
            
            # Select Visibility mode
            if args.dev:
                print("YouTube Uploader: Setting visibility to PRIVATE (dev mode)...")
                page.click('tp-yt-paper-radio-button[name="PRIVATE"], paper-radio-button[name="PRIVATE"]', force=True)
            else:
                print("YouTube Uploader: Setting visibility to PUBLIC...")
                page.click('tp-yt-paper-radio-button[name="PUBLIC"], paper-radio-button[name="PUBLIC"]', force=True)
                
            page.wait_for_timeout(1000)
            
            print("YouTube Uploader: Waiting for done-button to be ready (upload processing)...")
            page.wait_for_selector('#done-button', state='visible', timeout=180000)
            page.wait_for_timeout(1000)
            
            print("YouTube Uploader: Submitting and publishing...")
            done_btn = page.locator('#done-button')
            if done_btn.is_visible():
                done_btn.click(force=True)
            else:
                page.evaluate('document.querySelector("#done-button").click()')
            
            # Wait for upload completion dialog or wait a sufficient block for submission to complete
            page.wait_for_timeout(10000)
            print("YouTube Uploader: Upload complete!")
            
            browser.close()
            sys.exit(0)
            
        except Exception as e:
            print(f"Error during browser automation: {e}", file=sys.stderr)
            try:
                page.screenshot(path="downloads/playwright_error.png")
                print("YouTube Uploader: Saved error screenshot to downloads/playwright_error.png", file=sys.stderr)
            except Exception:
                pass
            browser.close()
            sys.exit(1)

if __name__ == "__main__":
    main()

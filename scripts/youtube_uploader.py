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
        print("YouTube Uploader: Launching headless browser...")
        # Use user agent matching modern browser to avoid automated action prompts
        browser = p.chromium.launch(headless=True)
        context = browser.new_context(
            user_agent="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
            viewport={"width": 1280, "height": 720}
        )
        context.add_cookies(cookies)
        
        page = context.new_page()
        
        try:
            print("YouTube Uploader: Navigating to YouTube Studio...")
            page.goto("https://studio.youtube.com", timeout=60000)
            page.wait_for_timeout(5000)
            
            if "accounts.google.com" in page.url:
                print("Error: Authentication failed. Session cookies are expired or invalid.", file=sys.stderr)
                page.screenshot(path="downloads/playwright_expired_cookies.png")
                sys.exit(1)
                
            print("YouTube Uploader: Logged in successfully. Opening upload wizard...")
            
            # Click direct upload button if visible, fallback to Create icon
            upload_btn = page.locator('#upload-button, [id="upload-button"], [aria-label="Upload videos"]')
            if upload_btn.is_visible():
                upload_btn.click()
            else:
                page.click('#create-icon, [id="create-icon"]')
                page.wait_for_timeout(1000)
                # Select Upload option from dropdown
                page.click('text=Upload videos, ytcp-text-menu-item')
                
            page.wait_for_selector('input[type="file"]', timeout=30000)
            print("YouTube Uploader: Selecting video file...")
            page.set_input_files('input[type="file"]', args.video)
            
            print("YouTube Uploader: Waiting for metadata inputs to load...")
            page.wait_for_selector('#title-textarea #textbox', timeout=60000)
            page.wait_for_timeout(2000)
            
            print("YouTube Uploader: Entering title and description...")
            # Set Title
            title_input = page.locator('#title-textarea #textbox')
            title_input.clear()
            title_input.fill(args.title)
            
            # Set Description
            desc_input = page.locator('#description-textarea #textbox')
            desc_input.clear()
            desc_input.fill(args.description)
            
            print("YouTube Uploader: Setting audience details (Not Made for Kids)...")
            page.click('tp-yt-paper-radio-button[name="VIDEO_MADE_FOR_KIDS_NOT_MFK"], paper-radio-button[name="VIDEO_MADE_FOR_KIDS_NOT_MFK"]')
            page.wait_for_timeout(1000)
            
            # Navigate Details -> Video Elements
            print("YouTube Uploader: Navigating Step 1 -> Step 2...")
            page.click('#next-button')
            page.wait_for_timeout(2000)
            
            # Navigate Video Elements -> Checks
            print("YouTube Uploader: Navigating Step 2 -> Step 3...")
            page.click('#next-button')
            page.wait_for_timeout(2000)
            
            # Navigate Checks -> Visibility
            print("YouTube Uploader: Navigating Step 3 -> Step 4...")
            page.click('#next-button')
            page.wait_for_timeout(2000)
            
            # Select Visibility mode
            if args.dev:
                print("YouTube Uploader: Setting visibility to PRIVATE (dev mode)...")
                page.click('tp-yt-paper-radio-button[name="PRIVATE"], paper-radio-button[name="PRIVATE"]')
            else:
                print("YouTube Uploader: Setting visibility to PUBLIC...")
                page.click('tp-yt-paper-radio-button[name="PUBLIC"], paper-radio-button[name="PUBLIC"]')
                
            page.wait_for_timeout(1000)
            
            print("YouTube Uploader: Submitting and publishing...")
            page.click('#done-button')
            
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

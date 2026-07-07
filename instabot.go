package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Davincible/goinsta/v3"
)

// MyInstabot is a wrapper around everything
type MyInstabot struct {
	Insta *goinsta.Instagram
}

var instabot MyInstabot

func main() {
	parseOptions()
	getConfig()

	printBanner()

	if youtubeMode {
		if _, err := os.Stat(youtubeCookiesFile); os.IsNotExist(err) {
			fmt.Printf("  %s✗%s cookies file not found at %s\n", ColorRed, ColorReset, youtubeCookiesFile)
			log.Fatalf("Please export your YouTube/Google cookies in Netscape format to this path to enable YouTube upload.")
		}
		logPrefix(PrefixYT, "Cookies verified at %s", youtubeCookiesFile)
	}

	loggedIn := false
	needsLogin := run || techMode || (ytSourceMode && run) || unfollow

	if ytSourceMode && !run && !techMode {
		// Pure YT-to-YT mode — no Instagram login needed
		printSection("YouTube Shorts Crawler")
		logPrefix(PrefixYTSource, "Starting in YT→YT mode (no Instagram)")
		instabot.ytSourceLoop()
	} else if needsLogin {
		login()
		loggedIn = true
	}

	if run && loggedIn {
		printSection("Engagement Loop")
		logPrefix(PrefixExplore, "Starting explore engagement (like, comment)")
	}

	if techMode && loggedIn {
		printSection("Tech Repost Loop")
		logPrefix(PrefixTech, "Hunting tech videos from Instagram Explore...")
		if ytSourceMode {
			logPrefix(PrefixYTSource, "YouTube Shorts crawler running in parallel")
		}
	}

	if techMode && run {
		go instabot.techExploreLoop()
		instabot.loopRandom()
	} else if techMode {
		instabot.techExploreLoop()
	} else if run {
		instabot.loopRandom()
	} else if unfollow && loggedIn {
		printSection("Unfollow Sync")
		instabot.syncFollowers()
		instabot.updateConfig()
	} else if !run && !techMode && !ytSourceMode && !unfollow {
		fmt.Printf("  %s!%s No mode selected. Use %s-run%s, %s-tech%s, %s-yt-source%s, or %s-sync%s.\n",
			ColorYellow, ColorReset,
			ColorCyan, ColorReset, ColorCyan, ColorReset, ColorCyan, ColorReset, ColorCyan, ColorReset)
	}
}

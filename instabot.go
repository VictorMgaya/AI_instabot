package main

import (
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

	if youtubeMode {
		if _, err := os.Stat(youtubeCookiesFile); os.IsNotExist(err) {
			log.Fatalf("YouTube Error: cookies file not found at %s. Please export your YouTube/Google cookies in Netscape format to this path to enable YouTube upload.", youtubeCookiesFile)
		}
		log.Println("YouTube: Verification of cookie path successful")
	}

	if techMode && run {
		login()
		log.Println("Starting both tech repost and engagement modes simultaneously...")
		go instabot.techExploreLoop()
		instabot.loopRandom()
	} else if techMode {
		login()
		log.Println("Starting tech video repost mode (random explore)...")
		instabot.techExploreLoop()
	} else if ytSourceMode && run {
		// YT source + IG engagement — need Instagram login
		login()
		log.Println("Starting YouTube source + Instagram engagement simultaneously...")
		go instabot.ytSourceLoop()
		instabot.loopRandom()
	} else if ytSourceMode {
		// Pure YT-only mode: crawl YouTube Shorts, post to YouTube — no Instagram needed
		log.Println("Starting YouTube Shorts crawler (YT → YT mode)...")
		instabot.ytSourceLoop()
	} else if run {
		login()
		instabot.loopRandom()
	} else if unfollow {
		login()
		instabot.syncFollowers()
		instabot.updateConfig()
	} else {
		log.Println("No mode selected. Use -run, -tech, -yt-source, or -sync. Add -h for help.")
	}
}

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
	} else if run {
		login()
		if tiktokMode {
			log.Println("Starting both Instagram and TikTok simultaneously...")
			go instabot.loopRandom()
			tiktokLogin()
			instabot.tiktokLoop()
		} else {
			instabot.loopRandom()
		}
	} else if tiktokMode {
		tiktokLogin()
		instabot.tiktokLoop()
	} else if unfollow {
		login()
		instabot.syncFollowers()
		instabot.updateConfig()
	}
}

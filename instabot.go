package main

import "github.com/Davincible/goinsta/v3"

// MyInstabot is a wrapper around everything
type MyInstabot struct {
	Insta *goinsta.Instagram
}

var instabot MyInstabot

func main() {
	// Gets the command line options
	parseOptions()
	// Gets the config
	getConfig()
	// Tries to login
	login()
	if unfollow {
		instabot.syncFollowers()
	} else if run {
		// Browse random explore content ; follows, likes, and comments
		instabot.loopRandom()
	}
	instabot.updateConfig()
}

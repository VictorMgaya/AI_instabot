package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/smtp"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

var (
	// Whether we are in development mode or not
	dev bool

	// Whether we want an email to be sent when the script ends / crashes
	nomail bool

	// Whether we want to launch the unfollow mode
	unfollow bool

	// Acut
	run bool

	// Whether we want to run tech video repost mode
	techMode bool

	// Whether we want to have logging
	logs bool

	// Used to skip following, liking and commenting same user in this session
	noduplicate bool

	// Whether we want to upload tech videos to YouTube Shorts
	youtubeMode bool

	// Whether we want YouTube Shorts as an additional content source
	ytSourceMode bool
)

// Safety limits
var (
	dailyInstagramFollowLimit  int
	dailyInstagramLikeLimit    int
	dailyInstagramCommentLimit int
	sleepStartHour             int
	sleepEndHour               int
	cycleDelayMin              int
	cycleDelayMax              int
)

// An image will be liked if the poster has more followers than likeLowerLimit, and less than likeUpperLimit
var likeLowerLimit int
var likeUpperLimit int

// A user will be followed if he has more followers than followLowerLimit, and less than followUpperLimit
// Needs to be a subset of the like interval
var followLowerLimit int
var followUpperLimit int

// An image will be commented if the poster has more followers than commentLowerLimit, and less than commentUpperLimit
// Needs to be a subset of the like interval
var commentLowerLimit int
var commentUpperLimit int

// Hashtags list. Do not put the '#' in the config file
var tagsList map[string]interface{}

// Limits for the current hashtag
var limits map[string]int

// Comments list
var commentsList []string

// Line is a struct to store one line of the report
type line struct {
	Tag, Action string
}

// Report that will be sent at the end of the script
var report map[line]int

var userBlacklist []string
var userWhitelist []string

// Counters that will be incremented while we like, comment, and follow
var numFollowed int
var numLiked int
var numCommented int

// check will log.Fatal if err is an error
func check(err error) {
	if err != nil {
		log.Fatal("ERROR:", err)
	}
}

// Parses the options given to the script
func parseOptions() {
	flag.BoolVar(&run, "run", false, "Use this option to follow, like and comment")
	flag.BoolVar(&techMode, "tech", false, "Use this option to download and repost tech videos with AI descriptions")
	flag.BoolVar(&unfollow, "sync", false, "Use this option to unfollow those who are not following back")
	flag.BoolVar(&nomail, "nomail", false, "Use this option to disable the email notifications")
	flag.BoolVar(&dev, "dev", false, "Use this option to use the script in development mode : nothing will be done for real")
	flag.BoolVar(&logs, "logs", false, "Use this option to enable the logfile")
	flag.BoolVar(&noduplicate, "noduplicate", false, "Use this option to skip following, liking and commenting same user in this session")
	flag.BoolVar(&youtubeMode, "youtube", false, "Cross-post videos to YouTube Shorts (requires config/youtube-cookies.txt)")
	flag.BoolVar(&ytSourceMode, "yt-source", false, "Crawl YouTube Shorts as a content source (combine with -tech, -youtube, or both)")

	flag.Parse()

	// -logs enables the log file
	if logs {
		// Opens a log file
		t := time.Now()
		logFile, err := os.OpenFile("instabot-"+t.Format("2006-01-02-15-04-05")+".log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
		check(err)
		defer logFile.Close()

		// Duplicates the writer to stdout and logFile
		mw := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(mw)
	}
}

// Gets the conf in the config file
func getConfig() {
	folder := "config"
	if dev {
		folder = "local"
	}
	viper.SetConfigFile(folder + "/config.json")

	// Reads the config file
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	// Check enviroment
	viper.SetEnvPrefix("instabot")
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()

	// Confirms which config file is used
	logPrefix(PrefixSafety, "Config loaded from %s", viper.ConfigFileUsed())

	likeLowerLimit = viper.GetInt("limits.like.min")
	likeUpperLimit = viper.GetInt("limits.like.max")

	followLowerLimit = viper.GetInt("limits.follow.min")
	followUpperLimit = viper.GetInt("limits.follow.max")

	commentLowerLimit = viper.GetInt("limits.comment.min")
	commentUpperLimit = viper.GetInt("limits.comment.max")

	dailyInstagramFollowLimit = viper.GetInt("safety.daily_instagram_follow")
	dailyInstagramLikeLimit = viper.GetInt("safety.daily_instagram_like")
	dailyInstagramCommentLimit = viper.GetInt("safety.daily_instagram_comment")
	sleepStartHour = viper.GetInt("safety.sleep_start_hour")
	sleepEndHour = viper.GetInt("safety.sleep_end_hour")
	cycleDelayMin = viper.GetInt("safety.cycle_delay_min")
	cycleDelayMax = viper.GetInt("safety.cycle_delay_max")

	// Set sensible defaults if not set
	if dailyInstagramFollowLimit == 0 { dailyInstagramFollowLimit = 60 }
	if dailyInstagramLikeLimit == 0 { dailyInstagramLikeLimit = 100 }
	if dailyInstagramCommentLimit == 0 { dailyInstagramCommentLimit = 15 }
	if sleepStartHour == 0 && sleepEndHour == 0 {
		sleepStartHour = 22
		sleepEndHour = 7
	}
	if cycleDelayMin == 0 { cycleDelayMin = 1200 }
	if cycleDelayMax == 0 { cycleDelayMax = 2700 }

	tagsList = viper.GetStringMap("tags")

	commentsList = viper.GetStringSlice("comments")

	userBlacklist = viper.GetStringSlice("blacklist")
	userWhitelist = viper.GetStringSlice("whitelist")

	type Report struct {
		Tag, Action string
	}

	report = make(map[line]int)
}

// Sends an email. Check out the "mail" section of the "config.json" file.
func send(body string, success bool) {
	if !nomail {
		from := viper.GetString("user.mail.from")
		pass := viper.GetString("user.mail.password")
		to := viper.GetString("user.mail.to")

		status := func() string {
			if success {
				return "Success!"
			}
			return "Failure!"
		}()
		msg := "From: " + from + "\n" +
			"To: " + to + "\n" +
			"Subject:" + status + "  go-instabot\n\n" +
			body

		err := smtp.SendMail(viper.GetString("user.mail.smtp"),
			smtp.PlainAuth("", from, pass, viper.GetString("user.mail.server")),
			from, []string{to}, []byte(msg))

		if err != nil {
			log.Printf("smtp error: %s", err)
			return
		}

		log.Print("sent")
	}
}

// Retries the same function [function], a certain number of times (maxAttempts).
// It is exponential : the 1st time it will be (sleep), the 2nd time, (sleep) x 2, the 3rd time, (sleep) x 3, etc.
// If this function fails to recover after an error, it will send an email to the address in the config file.
func retry(maxAttempts int, sleep time.Duration, function func() error) (err error) {
	for currentAttempt := 0; currentAttempt < maxAttempts; currentAttempt++ {
		err = function()
		if err == nil {
			return
		}
		for i := 0; i <= currentAttempt; i++ {
			time.Sleep(sleep)
		}
		log.Println("Retrying after error:", err)
	}

	send(fmt.Sprintf("The script has stopped due to an unrecoverable error :\n%s", err), false)
	return fmt.Errorf("After %d attempts, last error: %s", maxAttempts, err)
}

// Builds the report prints it and sends it
func buildReport() {
	reportAsString := ""
	for index, element := range report {
		var times string
		if element == 1 {
			times = "time"
		} else {
			times = "times"
		}
		reportAsString += fmt.Sprintf("%s %d %s\n", index.Action, element, times)
	}

	fmt.Println(reportAsString)
	send(reportAsString, true)
}

// ---------------------------------------------------------------------------
// Daily action counters — persisted to config/action_counters.json so limits
// survive restarts within the same calendar day.
// ---------------------------------------------------------------------------

const actionCountersFile = "config/action_counters.json"

// DailyCounters tracks how many Instagram actions have been taken today.
type DailyCounters struct {
	Date             string `json:"date"`
	InstagramFollows int    `json:"instagram_follows"`
	InstagramLikes   int    `json:"instagram_likes"`
	InstagramComments int   `json:"instagram_comments"`
}

var (
	dailyCounters    DailyCounters
	dailyCountersMu  sync.Mutex
)

// todayDate returns today's date string (YYYY-MM-DD).
func todayDate() string {
	return time.Now().Format("2006-01-02")
}

// loadOrCreateDailyCounters reads the persisted counters from disk.
// If the stored date differs from today the counters are reset.
func loadOrCreateDailyCounters() {
	dailyCountersMu.Lock()
	defer dailyCountersMu.Unlock()

	data, err := os.ReadFile(actionCountersFile)
	if err == nil {
		var stored DailyCounters
		if json.Unmarshal(data, &stored) == nil {
			if stored.Date == todayDate() {
				dailyCounters = stored
				logPrefix(PrefixSafety, "Loaded daily counters: follows=%d likes=%d comments=%d",
					dailyCounters.InstagramFollows,
					dailyCounters.InstagramLikes,
					dailyCounters.InstagramComments)
				return
			}
		}
	}
	// New day — reset counters
	dailyCounters = DailyCounters{Date: todayDate()}
	logPrefix(PrefixSafety, "New day — daily counters reset")
	saveDailyCounters()
}

// saveDailyCounters writes the current counters to disk (must be called with lock held).
func saveDailyCounters() {
	data, err := json.MarshalIndent(dailyCounters, "", "  ")
	if err != nil {
		log.Printf("[Safety] Failed to marshal counters: %v", err)
		return
	}
	if err := os.WriteFile(actionCountersFile, data, 0644); err != nil {
		log.Printf("[Safety] Failed to save counters: %v", err)
	}
}

// incrementDailyCounter increments the named counter and persists it.
// Valid actions: "follow", "like", "comment".
func incrementDailyCounter(action string) {
	dailyCountersMu.Lock()
	defer dailyCountersMu.Unlock()

	// Roll over if the date has changed mid-run
	if dailyCounters.Date != todayDate() {
		dailyCounters = DailyCounters{Date: todayDate()}
		log.Println("[Safety] Midnight rollover — daily counters reset.")
	}

	switch action {
	case "follow":
		dailyCounters.InstagramFollows++
	case "like":
		dailyCounters.InstagramLikes++
	case "comment":
		dailyCounters.InstagramComments++
	}
	saveDailyCounters()
}

// dailyLimitReached returns true when the given action has hit its daily cap.
func dailyLimitReached(action string) bool {
	dailyCountersMu.Lock()
	defer dailyCountersMu.Unlock()

	switch action {
	case "follow":
		if dailyInstagramFollowLimit > 0 && dailyCounters.InstagramFollows >= dailyInstagramFollowLimit {
			logPrefix(PrefixSafety, "Daily follow limit reached (%d/%d)", dailyCounters.InstagramFollows, dailyInstagramFollowLimit)
			return true
		}
	case "like":
		if dailyInstagramLikeLimit > 0 && dailyCounters.InstagramLikes >= dailyInstagramLikeLimit {
			logPrefix(PrefixSafety, "Daily like limit reached (%d/%d)", dailyCounters.InstagramLikes, dailyInstagramLikeLimit)
			return true
		}
	case "comment":
		if dailyInstagramCommentLimit > 0 && dailyCounters.InstagramComments >= dailyInstagramCommentLimit {
			logPrefix(PrefixSafety, "Daily comment limit reached (%d/%d)", dailyCounters.InstagramComments, dailyInstagramCommentLimit)
			return true
		}
	}
	return false
}

// checkSleepAndSleep puts the bot to sleep if the current local hour falls
// within the configured nighttime window (sleepStartHour – sleepEndHour).
// It sleeps until the wake hour plus a random jitter of 10-30 minutes.
func checkSleepAndSleep() {
	now := time.Now()
	h := now.Hour()

	sleeping := false
	if sleepStartHour < sleepEndHour {
		// e.g. 23:00 – 07:00 wraps midnight, handled below
		sleeping = h >= sleepStartHour || h < sleepEndHour
	} else {
		sleeping = h >= sleepStartHour && h < sleepEndHour
	}

	// Default: sleep between 22:00 and 07:00 if hours wrap midnight
	if sleepStartHour > sleepEndHour {
		sleeping = h >= sleepStartHour || h < sleepEndHour
	}

	if !sleeping {
		return
	}

	// Calculate wake time: next occurrence of sleepEndHour
	wakeBase := time.Date(now.Year(), now.Month(), now.Day(), sleepEndHour, 0, 0, 0, now.Location())
	if !wakeBase.After(now) {
		wakeBase = wakeBase.Add(24 * time.Hour)
	}
	// Add random human jitter: 10–30 minutes
	jitter := time.Duration(10+rand.Intn(21)) * time.Minute
	wakeTime := wakeBase.Add(jitter)

	sleepDuration := time.Until(wakeTime)
	logPrefix(PrefixSafety, "Night sleep — waking at ~%s (%s)",
		wakeTime.Format("15:04"), sleepDuration.Round(time.Minute))
	time.Sleep(sleepDuration)
	logPrefix(PrefixSafety, "Woke up — resuming bot")
}

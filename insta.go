package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"time"
	"unsafe"

	"github.com/Davincible/goinsta/v3"
	"github.com/spf13/viper"
)

// Storing user in session
var checkedUser = make(map[string]bool)

var insta *goinsta.Instagram

// login will try to reload a previous session, and will create a new one if it can't
func login() {
	err := reloadSession()
	if err != nil {
		createAndSaveSession()
	}
}

// reloadSession will attempt to recover a previous session
func reloadSession() error {

	insta, err := goinsta.Import("./goinsta-session")
	if err != nil {
		return errors.New("Couldn't recover the session")
	}

	if insta != nil {
		instabot.Insta = insta
	}

	log.Println("Successfully logged in")
	return nil

}

// Logins and saves the session
func createAndSaveSession() {
	insta := goinsta.New(viper.GetString("user.instagram.username"), viper.GetString("user.instagram.password"))
	instabot.Insta = insta
	err := instabot.Insta.Login()
	check(err)

	err = instabot.Insta.Export("./goinsta-session")
	check(err)
	log.Println("Created and saved the session")
}

func getInput(text string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf(text)
	input, err := reader.ReadString('\n')
	check(err)
	return strings.TrimSpace(input)
}

// Checks if the user is in the slice
func containsUser(slice []goinsta.User, user goinsta.User) bool {
	for _, currentUser := range slice {
		if currentUser.Username == user.Username {
			return true
		}
	}
	return false
}

func getInputf(format string, args ...interface{}) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf(format, args...)
	input, err := reader.ReadString('\n')
	check(err)
	return strings.TrimSpace(input)
}

// Same, with strings
func containsString(slice []string, user string) bool {
	for _, currentUser := range slice {
		if currentUser == user {
			return true
		}
	}
	return false
}

func (myInstabot MyInstabot) syncFollowers() {
	following := myInstabot.Insta.Account.Following("", goinsta.DefaultOrder)
	followers := myInstabot.Insta.Account.Followers("")

	var followerUsers []goinsta.User
	var followingUsers []goinsta.User

	for following.Next() {
		for _, user := range following.Users {
			followingUsers = append(followingUsers, *user)
		}
	}
	for followers.Next() {
		for _, user := range followers.Users {
			followerUsers = append(followerUsers, *user)
		}
	}

	var users []goinsta.User
	for _, user := range followingUsers {
		// Skip whitelisted users.
		if containsString(userWhitelist, user.Username) {
			continue
		}

		if !containsUser(followerUsers, user) {
			users = append(users, user)
		}
	}
	if len(users) == 0 {
		return
	}
	fmt.Printf("\n%d users are not following you back!\n", len(users))
	answer := getInput("Do you want to review these users ? [yN]")
	if answer != "y" {
		fmt.Println("Not unfollowing.")
		os.Exit(0)
	}

	answerUnfollowAll := getInput("Unfollow everyone ? [yN]")

	for _, user := range users {
		if answerUnfollowAll != "y" {
			answerUserUnfollow := getInputf("Unfollow %s ? [yN]", user.Username)
			if answerUserUnfollow != "y" {
				userWhitelist = append(userWhitelist, user.Username)
				continue
			}
		}

		userBlacklist = append(userBlacklist, user.Username)

		if !dev {
			user.Unfollow()
		}
		time.Sleep(6 * time.Second)
	}
}

// Follows a user, if not following already
func (myInstabot MyInstabot) followUser(user *goinsta.User) {
	log.Printf("Following %s\n", user.Username)
	// If not following already
	if !user.Friendship.Following {
		if !dev {
			user.Follow()
		}
		log.Println("Followed")
		numFollowed++
		report[line{"explore", "follow"}]++
	} else {
		log.Println("Already following " + user.Username)
	}
}

func (myInstabot MyInstabot) loopRandom() {
	for tagName := range tagsList {
		limitsConf := viper.GetStringMap("tags." + tagName)
		limits = map[string]int{
			"follow":  int(limitsConf["follow"].(float64)),
			"like":    int(limitsConf["like"].(float64)),
			"comment": int(limitsConf["comment"].(float64)),
		}
		break
	}

	for {
		numFollowed = 0
		numLiked = 0
		numCommented = 0
		log.Println("Fetching random content from explore...")
		myInstabot.browseExplore()
	}
}

// Likes an image, if not liked already
func (myInstabot MyInstabot) likeImage(image goinsta.Item) {
	log.Println("Liking the picture")
	if !image.HasLiked {
		if !dev {
			image.Like()
		}
		log.Println("Liked")
		numLiked++
		report[line{"explore", "like"}]++
	} else {
		log.Println("Image already liked")
	}
}

func (myInstabot MyInstabot) browseExplore() {
	if err := retry(3, 10*time.Second, func() error {
		if myInstabot.Insta.Discover.Next() {
			return nil
		}
		if err := myInstabot.Insta.Discover.Error(); err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Printf("Explore fetch error: %v", err)
		return
	}

	for _, section := range myInstabot.Insta.Discover.Items {
		if numFollowed >= limits["follow"] &&
			numLiked >= limits["like"] &&
			numCommented >= limits["comment"] {
			break
		}
		myInstabot.processSection(section)
	}
}

func (myInstabot MyInstabot) processSection(section goinsta.DiscoverSectionalItem) {
	items := extractExploreItems(section)
	for _, item := range items {
		if numFollowed >= limits["follow"] &&
			numLiked >= limits["like"] &&
			numCommented >= limits["comment"] {
			break
		}
		myInstabot.processItem(item)
	}
}

func extractExploreItems(section goinsta.DiscoverSectionalItem) []goinsta.Item {
	var items []goinsta.Item
	for _, m := range section.LayoutContent.Medias {
		items = append(items, m.Media)
	}
	for _, m := range section.LayoutContent.FillItems {
		items = append(items, m.Media)
	}
	if section.LayoutContent.OneByOneItem.Media.Pk != 0 {
		items = append(items, section.LayoutContent.OneByOneItem.Media)
	}
	if section.LayoutContent.TwoByTwoItem.Media.Pk != 0 {
		items = append(items, section.LayoutContent.TwoByTwoItem.Media)
	}
	return items
}

func (myInstabot MyInstabot) processItem(image goinsta.Item) {
	username := image.User.Username
	if username == "" || username == viper.GetString("user.instagram.username") {
		return
	}

	if checkedUser[username] && noduplicate {
		return
	}

	var userInfo *goinsta.User
	err := retry(10, 20*time.Second, func() (err error) {
		userInfo, err = myInstabot.Insta.Profiles.ByName(username)
		return
	})
	check(err)

	followerCount := userInfo.FollowerCount

	checkedUser[userInfo.Username] = true
	log.Printf("Checking %s — %d followers\n", userInfo.Username, followerCount)

	like := followerCount > likeLowerLimit && followerCount < likeUpperLimit && numLiked < limits["like"]
	follow := followerCount > followLowerLimit && followerCount < followUpperLimit && numFollowed < limits["follow"] && like
	comment := followerCount > commentLowerLimit && followerCount < commentUpperLimit && numCommented < limits["comment"] && like

	skip := false
	following := myInstabot.Insta.Account.Following("", goinsta.DefaultOrder)
	var followingUsers []goinsta.User
	for following.Next() {
		for _, user := range following.Users {
			followingUsers = append(followingUsers, *user)
		}
	}
	for _, user := range followingUsers {
		if user.Username == userInfo.Username {
			skip = true
			break
		}
	}

	if !skip {
		if like {
			myInstabot.likeImage(image)
			if follow && !containsString(userBlacklist, userInfo.Username) {
				myInstabot.followUser(userInfo)
			}
			if comment {
				myInstabot.commentImage(image, userInfo)
			}
		}
	}

	log.Printf("%s done\n\n", userInfo.Username)
	time.Sleep(20 * time.Second)
}

// Comments an image
func (myInstabot MyInstabot) commentImage(image goinsta.Item, userInfo *goinsta.User) {
	text := myInstabot.generateAISuggestion(image, userInfo)
	if text == "" && len(commentsList) > 0 {
		rand.Seed(time.Now().UnixNano())
		text = commentsList[rand.Intn(len(commentsList))]
	}
	if text == "" {
		log.Println("No comment to post")
		return
	}
	if !dev {
		comments := image.Comments
		if comments == nil {
			// monkey patching
			// we need to do that because https://github.com/ahmdrz/goinsta/pull/299 is not in goinsta/v2
			// I know, it's ugly
			newComments := goinsta.Comments{}
			rs := reflect.ValueOf(&newComments).Elem()
			rf := rs.FieldByName("item")
			rf = reflect.NewAt(rf.Type(), unsafe.Pointer(rf.UnsafeAddr())).Elem()
			item := reflect.New(reflect.TypeOf(image))
			item.Elem().Set(reflect.ValueOf(image))
			rf.Set(item)
			newComments.Add(text)
			// end hack!
		} else {
			comments.Add(text)
		}
	}
	log.Println("Commented " + text)
	numCommented++
	report[line{"explore", "comment"}]++
}

func (myInstabot MyInstabot) updateConfig() {
	viper.Set("whitelist", userWhitelist)
	viper.Set("blacklist", userBlacklist)

	err := viper.WriteConfig()
	if err != nil {
		log.Println("Update config file error", err)
		return
	}

	log.Println("Config file updated")
}

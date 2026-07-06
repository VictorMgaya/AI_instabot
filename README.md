<p align="center">
  <img src="docs/instabot.gif" alt="Instabot Demo" width="600"/>
</p>

<h1 align="center">🚀 Instabot</h1>

<p align="center">
  <b>Instagram automation — follow, like, comment, and unfollow on autopilot.</b>
</p>

<p align="center">
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go" alt="Go"></a>
  <img src="https://img.shields.io/badge/status-active-success" alt="Status">
  <img src="https://img.shields.io/github/license/VictorMgaya/AI_instabot" alt="License">
</p>

---

## ✨ Features

- 🤖 **Auto-follow** users from target hashtags  
- ❤️ **Auto-like** posts based on follower thresholds  
- 💬 **Auto-comment** with random picks from your list  
- 👋 **Auto-unfollow** users who don't follow back  
- 🧠 **Smart limits** — avoid bans with configurable delays  
- 📧 **Email reports** after each session  
- 🔐 **Session encryption** — login once, reuse safely  

## ⚙️ How it works

```
config.json  ──►  explore hashtags  ──►  like / follow / comment
                                      ──►  unfollow non-reciprocals
                                      ──►  email summary
```

You define **hashtags**, **actions per tag**, and **follower limits**. The bot browses Instagram through the unofficial API and performs actions that look natural.

## 🚦 Quick start

```bash
# 1. Install Go 1.26+
# 2. Clone & build
git clone https://github.com/VictorMgaya/AI_instabot
cd AI_instabot
go build -o instabot .

# 3. Copy & edit config
cp dist/config.json config/config.json
# edit config/config.json with your credentials and targets

# 4. Run
./instabot -run
```

<details>
<summary><b>📋 Options</b></summary>

| Flag | Description |
|------|-------------|
| `-run` | Run the bot |
| `-dev` | Dry-run mode (no real actions) |
| `-sync` | Unfollow non-reciprocal followers |
| `-logs` | Write logs to file |
| `-nomail` | Disable email reports |
| `-noduplicate` | Skip already-processed users |
| `-h` | Show help |

</details>

## 📁 Config example

```json
{
  "user": {
    "instagram": {
      "username": "your_handle",
      "password": "your_password"
    }
  },
  "limits": {
    "like":    { "min": 0, "max": 10000 },
    "comment": { "min": 100, "max": 10000 },
    "follow":  { "min": 200, "max": 10000 }
  },
  "tags": {
    "golang":  { "like": 3, "comment": 1, "follow": 1 },
    "photography": { "like": 5, "comment": 2, "follow": 1 }
  },
  "comments": ["awesome!", "nice shot!", "🔥"],
  "blacklist": ["spam_account"],
  "whitelist": ["friend_account"]
}
```

## 🛡️ Safety first

- ⏱️ **Random delays** between actions — looks human  
- 🔐 **Encrypted session** — store login once, avoid re-auth  
- 📉 **Follower limits** — avoid targeting big/influential accounts that report  

## 📬 Email reports

Optionally receive a summary after each run:

```
📊 Session Report
   👍 Liked:      24
   👣 Followed:   12
   💬 Commented:   8
   👋 Unfollowed:  5
```

Configure SMTP in `config.json` (Gmail works out of the box).

## 📄 License

GPL v3 — see [LICENSE](LICENSE).

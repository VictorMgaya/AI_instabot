<div align="center">
  <img src="docs/instabot.gif" alt="demo" width="700">
</div>

<br>

<h1 align="center">🤖 AI_Instabot</h1>

<p align="center">
  <i>Your Instagram growth, fully automated.</i>
  <br><br>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.26+-%2300ADD8?logo=go&logoColor=white" alt="Go"></a>
  <img src="https://img.shields.io/badge/status-active-%2322c55e" alt="Status">
  <img src="https://img.shields.io/badge/license-GPLv3-%238b5cf6" alt="License">
  <img src="https://img.shields.io/github/last-commit/VictorMgaya/AI_instabot" alt="Last commit">
  <img src="https://img.shields.io/badge/safety-%E2%9C%85%20human--like-brightgreen" alt="Safety">
</p>

---

## 📖 The Story

You spend hours scrolling, liking, following — hoping people notice you back.  
**This bot does it for you. Better. Faster. 24/7.**

AI_Instabot roams Instagram's hashtag feeds, finding real people in your niche. It likes their posts, drops a comment, follows them — all with human-like timing so your account stays safe. By morning, half of them visit your profile. Some follow back. Some like your stuff.

**Growth on autopilot. No smoke. No mirrors. Just code.**

---

## 🎯 What It Does

| Action | How |
|--------|-----|
| ❤️ **Like** | Likes posts from target hashtag feeds |
| 👣 **Follow** | Follows users who posted those images |
| 💬 **Comment** | Drops a random comment from your list |
| 👋 **Unfollow** | Unfollows users who don't follow back (sync mode) |

Every action is governed by **follower-count thresholds** you set — so you never waste engagement on bots or risk getting flagged by big accounts.

---

## ⚡ Quick Start

```bash
# Prerequisites: Go 1.26+
git clone https://github.com/VictorMgaya/AI_instabot
cd AI_instabot
go build -o instabot .

# Copy the sample config
cp dist/config.json config/config.json
# Edit with your Instagram credentials & targets
vim config/config.json

# Run
./instabot -run
```

---

## 🎮 Flags

```
  -run          Run the bot (like, follow, comment)
  -sync         Unfollow non-reciprocal followers
  -dev          Dry-run — no real actions (safe to test)
  -logs         Write everything to a log file
  -nomail       Skip the end-of-run email report
  -noduplicate  Skip users already processed this session
  -h            Help
```

---

## 📁 Config

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
    "golang": { "like": 3, "comment": 1, "follow": 1 },
    "photography": { "like": 5, "comment": 2, "follow": 1 }
  },
  "comments": ["awesome!", "nice one 🔥", "love this ❤️"],
  "blacklist": ["spam_account"],
  "whitelist": ["friend_account"]
}
```

<details>
<summary>📬 <b>Optional: Email reports</b></summary>

```json
"mail": {
  "from": "you@gmail.com",
  "password": "your_app_password",
  "to": "you@gmail.com",
  "smtp": "smtp.gmail.com:587",
  "server": "smtp.gmail.com"
}
```
</details>

---

## 🧠 How It Works

```
              ┌─────────────────┐
              │   config.json   │
              └────────┬────────┘
                       │
              ┌────────▼────────┐
              │   Random tech   │
              │   hashtag       │
              └────────┬────────┘
                       │
              ┌────────▼────────┐
              │  Fetch images   │
              │  via goinsta    │
              └────────┬────────┘
                       │
              ┌────────▼────────┐
              │  For each user: │
              │  ┌──────────┐   │
              │  │ Check    │   │
              │  │ follower │   │
              │  │ count    │   │
              │  └────┬─────┘   │
              │       │         │
              │  ┌────▼─────┐   │
              │  │ Like ✅  │   │
              │  │ Follow ✅│   │
              │  │Comment ✅│   │
              │  └──────────┘   │
              │       │         │
              │   ⏱️ 20s pause │
              └────────┬────────┘
                       │
              ┌────────▼────────┐
              │  Goals met?     │
              │  ──► yes: done  │
              │  ──► no: retry  │
              └─────────────────┘
```

---

## 🔒 Safety First

| Feature | Why |
|---------|-----|
| ⏱️ **20s delay** between actions | Looks human, avoids rate limits |
| 🔐 **Session encryption** | Login once, reuse. No repeated 2FA |
| 📉 **Follower thresholds** | Avoid bot accounts & report-happy influencers |
| 🔄 **Retry with backoff** | Instagram slow? Waits and retries gracefully |

---

## 🏗️ Tech Stack

- **Go 1.26+** — compiled, fast, single binary
- **goinsta/v3** — unofficial Instagram API (vendored locally)
- **Viper** — config management
- **net/smtp** — email reports

---

## 📄 License

**GPL v3** — Free as in freedom. Use it, modify it, share it.  
See [LICENSE](LICENSE) for details.

---

<div align="center">
  <sub>Built with ❤️ and Go · Not affiliated with Instagram™</sub>
  <br><br>
  <a href="https://www.paypal.com/ncp/payment/3QNCA24DEUXPC">
    <img src="docs/qrcode.png" width="160" alt="Buy me a coffee">
    <br>
    ☕ Buy me a coffee
  </a>
</div>

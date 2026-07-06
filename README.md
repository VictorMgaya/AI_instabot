<div align="center">
  <img src="docs/instabot.gif" alt="demo" width="700">
</div>

<br>

<h1 align="center">🤖 AI_Instabot</h1>

<p align="center">
  <i>Instagram growth on autopilot — powered by AI.</i>
  <br><br>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.26+-%2300ADD8?logo=go&logoColor=white" alt="Go"></a>
  <img src="https://img.shields.io/badge/status-active-%2322c55e" alt="Status">
  <img src="https://img.shields.io/badge/license-GPLv3-%238b5cf6" alt="License">
  <img src="https://img.shields.io/github/last-commit/VictorMgaya/AI_instabot" alt="Last commit">
</p>

---

## 📖 The Story

You scroll. You like. You follow. You hope.

**AI_Instabot doesn't hope.** It roams Instagram's Explore page — no hashtags, no bias, just real content from real accounts. Every comment is written by AI through OpenRouter, contextual to the post it's responding to.

One binary. One config. Set it and forget it.

---

## 🎯 What It Does

| Action | How |
|--------|------|
| ❤️ **Like** | Likes posts from the Explore feed |
| 👣 **Follow** | Follows users whose content appears |
| 💬 **AI Comment** | Every user gets an AI-generated comment based on their post caption |
| 👋 **Unfollow** | Unfollows non-reciprocal followers (`-sync`) |

No hashtag lists. No keyword hunting. Just random, fresh explore content every single run.

---

## 🧠 AI Comments

This is the core. When the bot encounters a user, it sends the post caption + user info to **OpenRouter** (`auto` model) and gets back a short, genuine-sounding comment. No more "nice pic" spam.

**Requires:** `openrouter.api_key` in `config/config.json` (or `OPENROUTER_API_KEY` env var).

---

## ⚡ Quick Start

```bash
git clone https://github.com/VictorMgaya/AI_instabot
cd AI_instabot
go build -o instabot .

cp dist/config.json config/config.json
# Edit config/config.json with your Instagram login & OpenRouter key
vim config/config.json

./instabot -run
```

---

## 🎮 Flags

```
  -run          Like, follow, and AI-comment on random explore content
  -sync         Unfollow users who don't follow back
  -dev          Dry-run (no real API mutations)
  -logs         Write logs to a file
  -nomail       Disable email report
  -noduplicate  Skip already-processed users this session
  -h            Help
```

---

## 📁 Config

```json
{
  "openrouter": {
    "api_key": "sk-or-v1-..."
  },
  "user": {
    "instagram": {
      "username": "your_handle",
      "password": "your_password"
    }
  },
  "limits": {
    "like":    { "min": 0, "max": 10000 },
    "follow":  { "min": 200, "max": 10000 }
  },
  "tags": {
    "session": { "like": 10, "follow": 5, "comment": 15 }
  },
  "blacklist": [],
  "whitelist": []
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

The `tags.session` values set per-run caps:
- `like` — max likes this session
- `follow` — max follows this session
- `comment` — max AI comments this session

---

## 🧠 How It Works

```
         ┌──────────────┐
         │  config.json  │
         └──────┬───────┘
                │
         ┌──────▼───────┐
         │  Explore     │
         │  (Refresh)   │  ← fresh page every run
         └──────┬───────┘
                │
         ┌──────▼───────┐
         │  Extract     │
         │  media items │
         └──────┬───────┘
                │
         ┌──────▼───────┐
         │  For each:   │
         │  ┌─────────┐ │
         │  │ Fetch   │ │
         │  │ profile │ │
         │  └────┬────┘ │
         │       │      │
         │  ┌────▼────┐ │
         │  │ Like ✅ │ │
         │  │Follow ✅│ │
         │  │AI Cmnt✅│ │  ← OpenRouter generates it
         │  └─────────┘ │
         │       │      │
         │   ⏱️ 20s    │
         └──────┬───────┘
                │
         ┌──────▼───────┐
         │  Caps met?   │
         │  ──► loop    │
         │  ──► refresh │
         └──────────────┘
```

---

## 🔒 Safety

| Feature | Why |
|---------|-----|
| ⏱️ **20s delay** | Looks human, avoids rate limits |
| 🔐 **Encrypted session** | Login once, no repeated 2FA |
| 📉 **Follower thresholds** | Avoid bot/scam accounts |
| 🔄 **Retry with backoff** | Handles API hiccups gracefully |
| ♻️ **Fresh explore page** | Never repeats content |

---

## 🏗️ Tech Stack

- **Go 1.26+** — single static binary
- **goinsta/v3** — unofficial Instagram API (vendored)
- **OpenRouter** — AI comment generation (model: `auto`)
- **Viper** — config management
- **net/smtp** — email reports

---

## 📄 License

**GPL v3** — Use it, modify it, share it.  
See [LICENSE](LICENSE).

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

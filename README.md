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

It also hunts for genuinely technical videos across **all of tech** — software, space, EVs, robotics, biotech, energy, quantum physics — downloads them, rewrites the caption with AI, and reposts them automatically.

One binary. One config. Set it and forget it.

---

## 🎯 What It Does

| Action | How |
|--------|------|
| ❤️ **Like** | Likes posts from the Explore feed |
| 👣 **Follow** | Follows users whose content appears on Explore |
| 💬 **AI Comment** | Every user gets a contextual AI-generated comment |
| 📹 **Tech Repost** | Finds, downloads & reposts tech videos with AI captions |
| 👋 **Unfollow** | Unfollows non-reciprocal followers (`-sync`) |

No hashtag lists. No keyword hunting. Just random, fresh explore content every cycle.

---

## 🧠 AI Comments & Captions

When the bot encounters a user it sends the post caption + profile info to **OpenRouter** (`auto` model) and gets back a short, genuine-sounding comment. Same for tech reposts — the AI rewrites the caption in a fresh, informative way.

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
  -tech         Hunt for tech videos and repost them with AI captions
  -sync         Unfollow users who don't follow back
  -dev          Dry-run (no real API mutations)
  -logs         Write logs to a timestamped log file
  -nomail       Disable email report on exit
  -noduplicate  Skip already-processed users this session
  -h            Help
```

Modes can be combined:

```bash
./instabot -run -tech        # engagement + tech repost simultaneously
./instabot -run -tech -dev   # full dry-run, nothing posted
./instabot -sync             # unfollow non-followers only
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
    "like":    { "min": 0,   "max": 10000 },
    "follow":  { "min": 200, "max": 10000 },
    "comment": { "min": 100, "max": 10000 }
  },
  "tags": {
    "session": { "like": 10, "follow": 5, "comment": 5 }
  },
  "tech": {
    "reposts": 5
  },
  "safety": {
    "daily_instagram_follow":  60,
    "daily_instagram_like":    100,
    "daily_instagram_comment": 15,
    "sleep_start_hour": 22,
    "sleep_end_hour":   7,
    "cycle_delay_min":  1200,
    "cycle_delay_max":  2700
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

### Key config fields

| Field | Purpose |
|-------|---------|
| `tags.<name>.like/follow/comment` | Per-cycle action caps for the engagement loop |
| `tech.reposts` | Max tech videos to repost per cycle |
| `safety.daily_instagram_*` | Hard daily caps — counters survive restarts |
| `safety.sleep_start/end_hour` | Bot sleeps during these local hours (night mode) |
| `safety.cycle_delay_min/max` | Seconds to wait between browse cycles |

---

## 🔬 Tech Repost — What Counts as "Tech"?

The bot uses a **weighted keyword scoring system** across two tiers. A video must score **≥ 3** from its caption alone (or ≥ 4 combined with the creator's bio) to qualify — preventing generic posts from slipping through.

| Domain | Examples |
|--------|---------|
| 🖥 Software / AI | `pytorch`, `kubernetes`, `llm`, `graphql`, `compiler` |
| 🤖 Robotics | `humanoid robot`, `exoskeleton`, `swarm robotics`, `slam` |
| 🚀 Space & Aerospace | `spacex`, `starship`, `orbital mechanics`, `james webb` |
| 🚗 Automotive / EVs | `solid state battery`, `adas`, `can bus`, `autonomous driving` |
| ✈️ Aviation / Drones | `vtol`, `pixhawk`, `turbofan`, `scramjet`, `avionics` |
| ⚡ Energy | `tokamak`, `photovoltaic`, `perovskite solar`, `supercapacitor` |
| ⚛️ Quantum / Physics | `qubit`, `qiskit`, `cern`, `gravitational wave` |
| 🧬 Biotech / MedTech | `crispr`, `alphafold`, `neuralink`, `microfluidics` |
| 🔬 Semiconductors | `lithography`, `mosfet`, `risc-v`, `oscilloscope` |
| 🧪 Materials science | `graphene`, `superconductor`, `carbon nanotube`, `additive manufacturing` |

---

## 🛡️ Safety System

The bot is designed to stay under Instagram's radar by mimicking real human behaviour.

| Feature | Detail |
|---------|---------|
| 🌙 **Night sleep mode** | Sleeps between `sleep_start_hour` and `sleep_end_hour` + random 10–30 min jitter |
| 📅 **Daily hard caps** | Persisted to `config/action_counters.json` — reset at midnight, survive restarts |
| ⏱️ **Human-scale delays** | 30–75 s after likes · 45–90 s before follow · 60–120 s after follow · 60–180 s between items |
| 🔄 **Long cycle gaps** | 20–45 minutes between explore crawls (configurable) |
| 🐢 **Slow unfollow** | 60–150 s random delay between each unfollow |
| 🔐 **Session persistence** | Login once, session saved to `goinsta-session` |
| 📉 **Follower thresholds** | Configurable min/max follower count for each action |
| 🔁 **Retry with backoff** | Exponential backoff on API errors |

---

## 🏗️ Tech Stack

- **Go 1.26+** — single static binary, zero dependencies at runtime
- **goinsta/v3** — unofficial Instagram API (vendored under `lib/`)
- **OpenRouter** — AI comment & caption generation (model: `auto`)
- **chromedp** — headless Chrome for TikTok interactions
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

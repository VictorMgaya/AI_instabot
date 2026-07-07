<div align="center">
  <img src="docs/instabot.gif" alt="demo" width="700">
</div>

<br>

<h1 align="center">🤖 Social Media Bot</h1>

<p align="center">
  <i>Instagram + YouTube — growth on autopilot, powered by AI.</i>
  <br><br>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.26+-%2300ADD8?logo=go&logoColor=white" alt="Go"></a>
  <img src="https://img.shields.io/badge/status-active-%2322c55e" alt="Status">
  <img src="https://img.shields.io/badge/license-GPLv3-%238b5cf6" alt="License">
  <img src="https://img.shields.io/github/last-commit/VictorMgaya/AI_instabot" alt="Last commit">
</p>

---

## Overview

One binary that runs three autonomous loops:

| Loop | What it does |
|------|-------------|
| **Engage** | Browses Instagram Explore — likes, follows, and writes AI-generated comments |
| **Tech Repost** | Finds technical videos on Instagram (software, space, EVs, robotics, biotech, AI…), downloads them, rewrites the caption with AI, and reposts |
| **YT Source** | Crawls YouTube Shorts organically, downloads every video at max quality (lossless ffmpeg merge), and cross-posts to YouTube and/or Instagram |

All comments and captions are written by **OpenRouter AI** — contextual to each post, not templated.

---

## Quick Start

```bash
git clone https://github.com/VictorMgaya/AI_instabot
cd AI_instabot

# Build the binary
go build -o instabot .

# Copy and edit config
cp dist/config.json config/config.json
vim config/config.json    # Add Instagram login + OpenRouter key

# Run
./instabot -run
```

---

## Flags

| Flag | Description |
|------|-------------|
| `-run` | Like, follow, and AI-comment on random Explore content |
| `-tech` | Hunt tech videos on IG and repost with AI captions |
| `-yt-source` | Crawl YouTube Shorts as a content source |
| `-youtube` | Cross-post videos to YouTube Shorts |
| `-sync` | Unfollow users who don't follow back |
| `-dev` | Dry-run — no real API mutations |
| `-logs` | Write logs to a timestamped file |
| `-nomail` | Disable email report on exit |
| `-noduplicate` | Skip already-processed users this session |

### Mode combinations

```bash
# Engagement only
./instabot -run

# Tech repost only
./instabot -tech

# Engagement + tech repost
./instabot -run -tech

# YT source → YouTube (no Instagram needed)
./instabot -yt-source -youtube

# YT source → Instagram
./instabot -yt-source -tech

# Everything: engage, tech repost, YT crawl, YT upload
./instabot -run -tech -yt-source -youtube

# Full dry-run
./instabot -run -tech -dev
```

---

## AI Comments & Captions

When the bot encounters a user it sends the post caption + profile info to **OpenRouter** and gets back a short, genuine-sounding comment.

For tech reposts, the AI rewrites the caption in a fresh, informative way with domain-appropriate emojis.

For YT source videos, the AI generates a YouTube-optimized title (no emojis, max 100 chars) and an energetic description.

**Requires:** `openrouter.api_key` in `config/config.json` (or `OPENROUTER_API_KEY` env var).

---

## Tech Repost — What Qualifies as Tech?

The bot uses a **weighted two-tier keyword scoring system**. A video must score ≥ 5 from its caption alone (or ≥ 7 combined with the creator's bio) to qualify.

High-weight keywords (2 pts) — unmistakably technical:

- **Software / AI** — `pytorch`, `kubernetes`, `llm`, `graphql`, `compiler`
- **Robotics** — `humanoid robot`, `exoskeleton`, `swarm robotics`, `slam`
- **Space & Aerospace** — `spacex`, `starship`, `orbital mechanics`, `james webb`
- **Automotive / EVs** — `solid state battery`, `adas`, `can bus`, `autonomous driving`
- **Aviation / Drones** — `vtol`, `pixhawk`, `turbofan`, `scramjet`, `avionics`
- **Energy** — `tokamak`, `photovoltaic`, `perovskite solar`, `supercapacitor`
- **Quantum / Physics** — `qubit`, `qiskit`, `cern`, `gravitational wave`
- **Biotech / MedTech** — `crispr`, `alphafold`, `neuralink`, `microfluidics`
- **Semiconductors** — `lithography`, `mosfet`, `risc-v`, `oscilloscope`
- **Materials science** — `graphene`, `superconductor`, `carbon nanotube`, `additive manufacturing`

Medium-weight keywords (1 pt) — `programming`, `robot`, `drone`, `3d printing`, `linux`, `database`…

A single vague keyword never qualifies on its own.

---

## YouTube Cross-Posting

The bot can post to YouTube Shorts in two ways:

1. **YT Source mode** (`-yt-source -youtube`): Crawls `youtube.com/shorts` using headless Chrome, downloads every Short at max quality (best video-only + audio-only streams merged losslessly via ffmpeg `-c copy`), generates AI title + description, and uploads via Playwright.

2. **Tech Repost + YouTube** (`-tech -youtube`): Tech videos found on Instagram are also uploaded to YouTube Shorts.

### YouTube Setup

1. Export your YouTube/Google cookies in **Netscape format** using a browser extension (e.g., Get cookies.txt for Chrome).
2. Save the file as `config/youtube-cookies.txt`.
3. Make sure you're logged into the YouTube channel you want to post to.

---

## Safety System

The bot is designed to stay under Instagram's radar by mimicking real human behaviour.

| Feature | Detail |
|---------|--------|
| Sleep mode | Sleeps between configurable night hours + random 10–30 min jitter |
| Daily hard caps | Persisted to `config/action_counters.json` — reset at midnight, survive restarts |
| Human-scale delays | 30–75 s after likes · 45–90 s before follow · 60–120 s after follow · 60–180 s between items |
| Long cycle gaps | 20–45 minutes between explore crawls (configurable) |
| Slow unfollow | 60–150 s random delay between each unfollow |
| Session persistence | Login once, session saved to `goinsta-session` |
| Follower thresholds | Configurable min/max follower count for each action |
| Retry with backoff | Exponential backoff on API errors |

---

## Configuration

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

### Key config fields

| Field | Purpose |
|-------|---------|
| `tags.<name>.like/follow/comment` | Per-cycle action caps for the engagement loop |
| `tech.reposts` | Max tech videos to repost per cycle |
| `safety.daily_instagram_*` | Hard daily caps — counters survive restarts |
| `safety.sleep_start/end_hour` | Bot sleeps during these local hours |
| `safety.cycle_delay_min/max` | Seconds to wait between browse cycles |

<details>
<summary><b>📬 Optional: Email reports</b></summary>

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

## Tech Stack

- **Go 1.26+** — single static binary, zero runtime dependencies
- **goinsta/v3** — unofficial Instagram API
- **OpenRouter** — AI comment & caption generation
- **chromedp** — headless Chrome for YouTube Shorts crawling
- **Playwright** — browser automation for YouTube uploads
- **kkdai/youtube** — YouTube video downloading
- **ffmpeg** — lossless stream merging (bundled as `./ffmpeg`)
- **Viper** — configuration management

---

## License

**GPL v3** — Use it, modify it, share it.
See [LICENSE](LICENSE).

---

<div align="center">
  <sub>Built with Go · Not affiliated with Instagram™ or YouTube™</sub>
  <br><br>
  <a href="https://www.paypal.com/ncp/payment/3QNCA24DEUXPC">
    <img src="docs/qrcode.png" width="160" alt="Buy me a coffee">
    <br>
    ☕ Buy me a coffee
  </a>
</div>

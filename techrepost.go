package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/Davincible/goinsta/v3"
)

var processedTechIDs = make(map[string]bool)

var techRepostCount int

// techKeywords are split by weight so vague words don't alone trigger a repost.
// Score needed to qualify: >= 3 (caption) or >= 4 (bio-only path).
// High-weight (2 pts) — unmistakably technical across ALL tech domains
var techKeywordsHigh = []string{
	// ── Software & Programming ─────────────────────────────────────────────
	"python", "javascript", "typescript", "golang", "rust", "kotlin", "swift",
	"c++", "c#", "java", "ruby", "php", "scala", "haskell", "elixir", "dart",
	"bash", "powershell", "assembly", "webassembly", "wasm",
	"react", "nextjs", "next.js", "vue", "angular", "svelte", "django", "flask",
	"fastapi", "spring boot", "laravel", "rails", "express", "nestjs",
	"pytorch", "tensorflow", "keras", "scikit-learn", "hugging face",
	"docker", "kubernetes", "k8s", "terraform", "ansible", "jenkins", "grafana",
	"prometheus", "nginx", "apache", "redis", "kafka", "rabbitmq",
	"algorithm", "data structure", "compiler", "interpreter", "runtime",
	"neural network", "large language model", "llm", "transformer", "gpt",
	"machine learning", "deep learning", "reinforcement learning",
	"natural language processing", "nlp", "computer vision", "diffusion model",
	"generative ai", "stable diffusion", "fine-tuning", "embedding",
	"rest api", "graphql", "grpc", "websocket", "microservices", "serverless",
	"devops", "mlops", "ci/cd", "continuous integration",
	"postgresql", "mysql", "mongodb", "elasticsearch", "vector database",
	"blockchain", "smart contract", "solidity", "web3", "ethereum",
	"cybersecurity", "penetration testing", "ctf", "buffer overflow",
	"reverse engineering", "malware", "vulnerability", "exploit",
	"linux kernel", "shell script", "systemd",
	"raspberry pi", "arduino", "esp32", "fpga", "embedded systems",
	"gpu", "cuda", "tpu", "quantization",
	"open source", "github", "pull request", "leetcode", "system design",

	// ── AI & Robotics Hardware ─────────────────────────────────────────────
	"humanoid robot", "boston dynamics", "quadruped", "robotic arm",
	"lidar", "slam", "autonomous robot", "exoskeleton", "robot dog",
	"computer vision model", "object detection", "yolo", "semantic segmentation",
	"reinforcement learning robot", "soft robotics", "swarm robotics",

	// ── Space & Aerospace ──────────────────────────────────────────────────
	"rocket", "spacecraft", "spacex", "nasa", "blue origin",
	"starship", "falcon 9", "launch vehicle", "orbital mechanics",
	"satellite", "cubesat", "space station", "iss",
	"mars mission", "moon landing", "artemis", "lunar rover",
	"rocket engine", "propulsion", "thrust", "payload fairing",
	"aerodynamics", "hypersonic", "reentry vehicle", "heat shield",
	"telescope", "james webb", "hubble", "radio telescope",
	"astronaut", "spacewalk", "eva suit", "microgravity",

	// ── Automotive & EVs ───────────────────────────────────────────────────
	"electric vehicle", "ev battery", "battery pack", "bms",
	"tesla", "rivian", "lucid motors", "nio",
	"motor controller", "inverter", "regenerative braking",
	"autonomous driving", "self-driving", "lidar sensor", "radar sensor",
	"obd", "can bus", "vehicle ecu", "adas",
	"charging station", "dc fast charging", "solid state battery",
	"fuel cell", "hydrogen vehicle", "range extender",

	// ── Aviation & Drones ──────────────────────────────────────────────────
	"uav", "drone swarm", "fixed-wing drone", "vtol",
	"flight controller", "pixhawk", "autopilot", "inertial navigation",
	"turbofan", "turbojet", "scramjet", "electric aircraft",
	"airfoil", "lift coefficient", "avionics",

	// ── Energy & Power ─────────────────────────────────────────────────────
	"solar panel", "photovoltaic", "wind turbine", "offshore wind",
	"nuclear reactor", "fusion reactor", "iter", "tokamak",
	"energy storage", "grid battery", "power electronics",
	"inverter topology", "pwm", "mppt",
	"hydrogen electrolysis", "fuel cell stack", "perovskite solar",
	"supercapacitor", "flywheel energy", "pumped hydro",

	// ── Quantum & Advanced Physics ─────────────────────────────────────────
	"quantum computing", "qubit", "quantum entanglement",
	"quantum circuit", "quantum error correction", "qiskit",
	"quantum supremacy", "quantum cryptography",
	"particle accelerator", "cern", "hadron collider",
	"dark matter", "gravitational wave", "ligo",
	"plasma physics", "magnetic confinement",

	// ── Biotech & Medical Tech ─────────────────────────────────────────────
	"crispr", "gene editing", "dna sequencing", "genome",
	"mrna vaccine", "bioreactor", "protein folding", "alphafold",
	"medical imaging", "mri machine", "ct scanner", "ultrasound probe",
	"prosthetic limb", "brain-computer interface", "bci", "neuralink",
	"lab on a chip", "microfluidics", "biosensor",
	"surgical robot", "da vinci robot",

	// ── Semiconductors & Electronics ──────────────────────────────────────
	"semiconductor", "chip fabrication", "wafer", "lithography",
	"transistor", "mosfet", "asic", "soc",
	"arm processor", "risc-v", "x86", "microprocessor",
	"pcb design", "schematic", "oscilloscope", "logic analyzer",
	"signal processing", "dsp", "adc", "dac", "rf circuit",

	// ── Materials Science & Nanotechnology ────────────────────────────────
	"graphene", "carbon nanotube", "nanomaterial", "nanoparticle",
	"metamaterial", "superconductor", "topological insulator",
	"3d bioprinting", "additive manufacturing", "sintering",
	"composite material", "carbon fiber", "titanium alloy",
}

// Medium-weight (1 pt) — clearly tech but sometimes context-dependent
var techKeywordsMed = []string{
	// Software
	"programming", "developer", "software engineer", "coding", "coder",
	"tech lead", "backend", "frontend", "fullstack", "cloud", "server",
	"database", "infrastructure", "automation", "framework", "library",
	"vscode", "neovim", "terminal", "cli", "debugging", "refactor",
	"deployment", "repository", "open-source", "linux", "operating system",
	"data science", "data engineering", "analytics", "visualization",
	"internet of things", "hardware", "circuit", "electronics",
	"model training", "dataset", "benchmark", "version control",
	"tech stack", "saas", "paas", "cloud native",
	"infosec", "firewall", "encryption", "hackathon",
	"web development", "mobile app", "microcontroller",

	// Robotics & automation
	"robot", "robotics", "drone", "sensor", "actuator",
	"web development", "mobile app", "cross-platform",
	"3d printing", "pcb", "soldering", "microcontroller",
}

// isTechRelated uses a strict weighted scoring system.
// Requires a high score to avoid posting borderline content.
//
//   Caption alone >= 5  → qualifies  (needs 2+ strong signals)
//   Caption + bio >= 7  → qualifies  (strong combined signal)
//
// A single vague keyword never qualifies on its own.
func isTechRelated(item *goinsta.Item) bool {
	caption := strings.ToLower(item.Caption.Text)
	bio := strings.ToLower(item.User.Biography)

	// Must have a real caption — no caption = skip
	if len(strings.TrimSpace(item.Caption.Text)) < 20 {
		return false
	}

	captionScore := scoreTech(caption)
	if captionScore >= 5 {
		return true
	}
	return captionScore+scoreTech(bio) >= 7
}

// scoreTech returns a weighted tech score for a block of text.
func scoreTech(text string) int {
	if text == "" {
		return 0
	}
	score := 0
	for _, kw := range techKeywordsHigh {
		if strings.Contains(text, kw) {
			score += 2
		}
	}
	for _, kw := range techKeywordsMed {
		if strings.Contains(text, kw) {
			score += 1
		}
	}
	return score
}


func (myInstabot MyInstabot) techExploreLoop() {
	rand.Seed(time.Now().UnixNano())
	os.MkdirAll("downloads", 0o755)

	if ytSourceMode {
		logPrefix(PrefixYTSource, "Spawning YouTube Shorts crawler goroutine")
		go myInstabot.ytSourceLoop()
	}

	for {
		logPrefix(PrefixTech, "Scanning explore for qualifying tech videos")
		myInstabot.techBrowseExplore()
		time.Sleep(30 * time.Second)
	}
}

func (myInstabot MyInstabot) techBrowseExplore() {
	myInstabot.Insta.Discover.Items = nil
	myInstabot.Insta.Discover.SectionalItems = nil

	if err := retry(3, 10*time.Second, func() error {
		if myInstabot.Insta.Discover.Refresh() {
			return nil
		}
		if err := myInstabot.Insta.Discover.Error(); err != nil {
			if strings.Contains(err.Error(), "feedback_required") {
				logPrefix(PrefixTech, "Rate-limited — backing off 60s")
				time.Sleep(60 * time.Second)
			}
			return err
		}
		return nil
	}); err != nil {
		logPrefix(PrefixTech, "Explore fetch error: %v", err)
		return
	}

	for _, section := range myInstabot.Insta.Discover.Items {
		myInstabot.techProcessSection(section)
	}
}

func (myInstabot MyInstabot) techProcessSection(section goinsta.DiscoverSectionalItem) {
	items := extractExploreItems(section)
	for _, item := range items {
		if item.MediaType != 2 { // videos only
			continue
		}

		pk := fmt.Sprintf("%d", item.Pk)
		if processedTechIDs[pk] {
			continue
		}
		processedTechIDs[pk] = true

		score := scoreTech(strings.ToLower(item.Caption.Text)) +
			scoreTech(strings.ToLower(item.User.Biography))
		logPrefix(PrefixTech, "@%s score=%d caption=%q",
			item.User.Username, score, truncateStr(item.Caption.Text, 80))

		if !isTechRelated(&item) {
			logPrefix(PrefixTech, "Skipping @%s — score too low", item.User.Username)
			continue
		}

		myInstabot.downloadAndRepost(&item)
	}
}

// truncateStr trims a string for log display.
func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func (myInstabot MyInstabot) downloadAndRepost(item *goinsta.Item) {
	logPrefix(PrefixTech, "Downloading video %d from @%s", item.Pk, item.User.Username)

	videoData, err := item.Download()
	if err != nil {
		logPrefix(PrefixTech, "Download error: %v", err)
		return
	}

	logPrefix(PrefixTech, "Downloaded %d bytes from @%s", len(videoData), item.User.Username)

	// Save video locally so it can be uploaded via browser automation
	localPath := fmt.Sprintf("downloads/repost-%d.mp4", item.Pk)
	if err := writeVideoFile(localPath, videoData); err != nil {
		logPrefix(PrefixTech, "Local save error: %v", err)
		localPath = ""
	} else {
		logPrefix(PrefixTech, "Saved locally at %s", localPath)
		defer func() {
			removeVideoFile(localPath)
		}()
	}

	caption := myInstabot.generateTechDescription(item)

	logPrefix(PrefixTech, "Uploading → %s", caption)

	if !dev {
		_, err := myInstabot.Insta.Upload(&goinsta.UploadOptions{
			File:    bytes.NewReader(videoData),
			Caption: caption,
		})
		if err != nil {
			logPrefix(PrefixTech, "Upload error: %v", err)
			return
		}
		logPrefix(PrefixTech, "Uploaded successfully ✓")
		techRepostCount++
	} else {
		logPrefix(PrefixTech, "[DEV] Would upload %d bytes", len(videoData))
		techRepostCount++
	}

	if youtubeMode && localPath != "" {
		ytTitle := caption
		if len(ytTitle) > 95 {
			ytTitle = ytTitle[:92] + "..."
		}
		err := uploadToYouTubeShorts(localPath, ytTitle, caption)
		if err != nil {
			logPrefix(PrefixTech, "YouTube upload error: %v", err)
		}
	}
}

func (myInstabot MyInstabot) generateTechDescription(item *goinsta.Item) string {
	caption := strings.TrimSpace(item.Caption.Text)
	if caption == "" {
		caption = "no caption"
	}

	prompt := fmt.Sprintf(
		`You are an energetic tech content creator. Write a short, punchy description (max 30 words) for this tech video repost.

Video caption: "%s"
Username: %s

Rules:
- Be informative, exciting and enthusiastic
- Sound like a passionate tech enthusiast
- Use 2-4 relevant emojis that match the tech domain (e.g. 🚀 for space, 🤖 for robotics, ⚡ for energy, 🧬 for biotech, 💻 for software)
- NO hashtags at all — zero, none
- Reply with ONLY the description text, nothing else`,
		caption, item.User.Username,
	)

	desc := generateAIComment(prompt)
	if desc == "" {
		desc = "This is next-level tech! 🚀🔥"
	}
	return desc
}

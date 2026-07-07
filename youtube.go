package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
)

const youtubeCookiesFile = "config/youtube-cookies.txt"

// uploadToYouTubeShorts runs the Python Playwright uploader script as a subprocess
// to handle the YouTube Studio upload pipeline securely and reliably.
func uploadToYouTubeShorts(videoPath string, description string) error {
	log.Printf("YouTube: Running Python Playwright uploader for: %s", videoPath)

	// Build the command arguments
	args := []string{
		"scripts/youtube_uploader.py",
		"--video", videoPath,
		"--title", description, // title defaults to description
		"--description", description,
		"--cookies", youtubeCookiesFile,
	}

	if dev {
		args = append(args, "--dev")
	}

	// Run using the python execution binary inside the virtual environment
	cmd := exec.Command("./venv/bin/python", args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	stdoutStr := stdoutBuf.String()
	stderrStr := stderrBuf.String()

	// Print stdout to keep the console logging transparent
	if stdoutStr != "" {
		log.Printf("YouTube Uploader Output:\n%s", stdoutStr)
	}

	if err != nil {
		if stderrStr != "" {
			log.Printf("YouTube Uploader Error Output:\n%s", stderrStr)
		}
		return fmt.Errorf("playwright uploader failed: %w", err)
	}

	log.Println("YouTube: Video uploaded successfully ✓")
	return nil
}

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

const youtubeCookiesFile = "config/youtube-cookies.txt"

// uploadToYouTubeShorts runs the Python Playwright uploader script as a subprocess
// and streams its output in real-time.
func uploadToYouTubeShorts(videoPath string, title string, description string) error {
	logPrefix(PrefixYT, "Running Playwright uploader for %s", videoPath)

	args := []string{
		"scripts/youtube_uploader.py",
		"--video", videoPath,
		"--title", title,
		"--description", description,
		"--cookies", youtubeCookiesFile,
	}

	if dev {
		args = append(args, "--dev")
	}

	cmd := exec.Command("./venv/bin/python", args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	cmd.Start()

	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)

	err := cmd.Wait()
	if err != nil {
		return fmt.Errorf("playwright uploader failed: %w", err)
	}

	logPrefix(PrefixYT, "Video uploaded successfully ✓")
	return nil
}

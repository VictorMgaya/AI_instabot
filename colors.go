package main

import (
	"fmt"
	"log"
	"strings"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"

	ColorBgBlue   = "\033[44m"
	ColorBgPurple = "\033[45m"
	ColorBgCyan   = "\033[46m"
	ColorBgDark   = "\033[48;5;235m"
)

type LogPrefix struct {
	Label string
	Color string
}

var (
	PrefixInsta    = LogPrefix{"INSTA", ColorGreen}
	PrefixYT       = LogPrefix{"YOUTUBE", ColorRed}
	PrefixYTSource = LogPrefix{"YT-SOURCE", ColorCyan}
	PrefixTech     = LogPrefix{"TECH", ColorPurple}
	PrefixSafety   = LogPrefix{"SAFETY", ColorYellow}
	PrefixAI       = LogPrefix{"AI", ColorBlue}
	PrefixExplore  = LogPrefix{"EXPLORE", ColorGreen}
)

func colorize(text string, color string) string {
	return color + text + ColorReset
}

func bold(text string) string {
	return ColorBold + text + ColorReset
}

func dim(text string) string {
	return ColorDim + text + ColorReset
}

func formatLog(prefix LogPrefix, format string, args ...interface{}) string {
	tag := colorize(" "+prefix.Label+" ", ColorWhite+ColorBgDark)
	msg := fmt.Sprintf(format, args...)
	return fmt.Sprintf("%s%s %s", tag, dim(" ┊"), msg)
}

func logPrefix(prefix LogPrefix, format string, args ...interface{}) {
	log.Println(formatLog(prefix, format, args...))
}

func printBanner() {
	fmt.Println()
	fmt.Printf("  %s%s╔══════════════════════════════════════════╗%s\n", ColorBold, ColorCyan, ColorReset)
	fmt.Printf("  %s║      Social Media Bot v2.0               ║%s\n", ColorCyan, ColorReset)
	fmt.Printf("  %s║      Instagram + YouTube Auto-Pilot      ║%s\n", ColorCyan, ColorReset)
	fmt.Printf("  %s╚══════════════════════════════════════════╝%s\n", ColorCyan, ColorReset)
	fmt.Println()

	var modes []string
	if run {
		modes = append(modes, colorize("ENGAGE", ColorGreen)+" (like, comment)")
	}
	if techMode {
		modes = append(modes, colorize("TECH REPOST", ColorPurple)+" (AI tech videos → IG)")
	}
	if ytSourceMode {
		modes = append(modes, colorize("YT SOURCE", ColorCyan)+" (YouTube Shorts crawl)")
	}
	if youtubeMode {
		modes = append(modes, colorize("YT UPLOAD", ColorRed)+" (cross-post to YouTube)")
	}
	if unfollow {
		modes = append(modes, colorize("SYNC", ColorYellow)+" (unfollow non-reciprocal)")
	}
	if dev {
		modes = append(modes, colorize("DEV", ColorDim)+" (dry-run)")
	}

	fmt.Printf("  %sActive Modes:%s\n", bold(""), ColorReset)
	for _, m := range modes {
		fmt.Printf("    • %s\n", m)
	}
	if !run && !techMode && !ytSourceMode && !youtubeMode && !unfollow {
		fmt.Printf("    %s(no mode selected — use -h for help)%s\n", dim(""), ColorReset)
	}
	fmt.Println()
}

func printSection(title string) {
	line := strings.Repeat("─", 50)
	fmt.Printf("\n  %s%s%s %s\n", ColorDim, line, ColorReset, bold(title))
}

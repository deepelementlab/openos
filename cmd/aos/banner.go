package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/agentos/aos/internal/version"
)

// ANSI color codes - lime/yellow-green for the mascot (matching the logo image)
const (
	ansiLimeGreen = "\033[93m" // Bright yellow - closest to lime green in standard ANSI
	ansiGreen     = "\033[92m" // Bright green
	ansiYellow    = "\033[93m" // Bright yellow
	ansiReset     = "\033[0m"
	ansiDim       = "\033[2m"
	ansiBold      = "\033[1m"
)

// Unicode block character for solid fill
const blockChar = "\u2588" // U+2588 █

// useANSIColor detects if the terminal supports ANSI colors.
func useANSIColor() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("AOS_NO_COLOR") != "" {
		return false
	}
	if os.Getenv("AOS_FORCE_COLOR") != "" {
		return true
	}
	if runtime.GOOS == "windows" {
		return false
	}
	return true
}

// mascotPatterns returns the ASCII art pattern for the apple monster
// Design: Apple-shaped body with leaf, big eyes, smile with fangs, arms and legs
func mascotPatterns() []string {
	return []string{
		"            @@@@@    @@@@@",    // Leaf base
		"             @@@@@  @@@@@",     // Leaf spread
		"          @@@@@@@@@@@@@@@@",    // Top of head
		"        @@@@@@@@@@@@@@@@@@@@",  // Forehead
		"       @@@@@@@@@@@@@@@@@@@@@@", // Upper face
		"      @@@@@@ @@@@@@@@ @@@@@@",  // Eye area (white space = sclera)
		"      @@@@@  @@@@@@@@  @@@@@",  // Eye pupils
		"      @@@@@   @@@@@@   @@@@@",  // Cheeks
		"       @@@     @@@@     @@@",   // Nose
		"       @@@  @@ @@@@ @@  @@@",   // Smile with fangs ^^
		"        @@@@@@@@@@@@@@@@@",     // Chin
		"          @@@@@@@@@@@@@",       // Neck
		"        @@@@@@@@@@@@@@@@@",     // Shoulders + arms start
		"       @@@@           @@@@",    // Arms extending
		"       @@               @@",    // Hands
		"      @@@               @@@",   // Body sides
		"      @@@               @@@",   // Body middle
		"       @@@             @@@",    // Body taper
		"        @@@@@@@@@@@@@@@",       // Bottom
		"           @@@   @@@",          // Legs
		"           @@@   @@@",          // Feet
	}
}

// openOSLogo returns the OpenOS text logo using raw string literals to avoid escape issues
func openOSLogo() []string {
	return []string{
		`   ____                   ____   _____`,
		`  / __ \/___  ___  ____  / __ \ / ___/`,
		` / / / / __ \/ _ \/ __ \/ / / / \__ \ `,
		`/ /_/ / /_/ /  __/ / / / /_/ / ___/ /`,
		`\____/ .___/\___/_/ /_/\____/ /____/`,
		`    /_/`,
	}
}

func printMascot(out *os.File, color bool) {
	patterns := mascotPatterns()
	if !color {
		for _, line := range patterns {
			fmt.Fprintln(out, strings.ReplaceAll(line, "@", blockChar))
		}
		return
	}
	// With lime green color
	g, r := ansiLimeGreen, ansiReset
	for _, line := range patterns {
		result := strings.ReplaceAll(line, "@", g+blockChar+r)
		fmt.Fprintln(out, result)
	}
}

func printOpenOSLogo(out *os.File, color bool) {
	logo := openOSLogo()
	if !color {
		for _, line := range logo {
			fmt.Fprintln(out, line)
		}
		return
	}
	// Print in lime green
	g, r := ansiLimeGreen, ansiReset
	for _, line := range logo {
		fmt.Fprintln(out, g+line+r)
	}
}

// BannerExplicitlyDisabled is true only when AOS_NO_BANNER is set to 1, true, yes, or on (case-insensitive).
// Unset or other values keep the default: show the welcome banner.
func BannerExplicitlyDisabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("AOS_NO_BANNER")))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func printWelcomeBanner() {
	if BannerExplicitlyDisabled() {
		return
	}
	color := useANSIColor()
	ver := version.GetFullVersion()

	// Print top spacing
	fmt.Fprintln(os.Stdout)

	// Print OpenOS text logo
	printOpenOSLogo(os.Stdout, color)

	// Print spacing
	fmt.Fprintln(os.Stdout)

	// Print mascot
	printMascot(os.Stdout, color)

	// Print spacing
	fmt.Fprintln(os.Stdout)

	if color {
		fmt.Fprintf(os.Stdout, "  %s%s> Agent Operating System <%s\n", ansiBold, ansiLimeGreen, ansiReset)
		fmt.Fprintln(os.Stdout, "  OpenOS / AOS - Control Plane | HTTP API | Health Checks | Metrics | Multi-tenancy")
		fmt.Fprintf(os.Stdout, "  %s%s%s\n", ansiDim, ver, ansiReset)
		fmt.Fprintf(os.Stdout, "  %sTip: AOS_NO_BANNER=1|true|yes|on to hide | NO_COLOR=1 to disable colors | aos doctor%s\n", ansiDim, ansiReset)
		fmt.Fprintln(os.Stdout)
		return
	}

	fmt.Fprintln(os.Stdout, "  > Agent Operating System <")
	fmt.Fprintln(os.Stdout, "  OpenOS / AOS - Control Plane | HTTP API | Health Checks | Metrics | Multi-tenancy")
	fmt.Fprintf(os.Stdout, "  %s\n", ver)
	fmt.Fprintln(os.Stdout, "  Tip: AOS_NO_BANNER=1|true|yes|on to hide | AOS_FORCE_COLOR=1 to force colors on Windows | aos doctor")
	fmt.Fprintln(os.Stdout)
}

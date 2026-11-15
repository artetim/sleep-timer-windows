package main

import (
	"fmt"
	"math"
	"time"
)

const minMinutes = 1
const maxMinutes = 1440

func validMinutes(n int) bool { return n >= minMinutes && n <= maxMinutes }

func formatSeconds(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	return fmt.Sprintf("%02d:%02d:%02d", seconds/3600, (seconds%3600)/60, seconds%60)
}

func secondsRemaining(now, deadline time.Time) int {
	seconds := int(math.Ceil(deadline.Sub(now).Seconds()))
	if seconds < 0 {
		return 0
	}
	return seconds
}

// An active 250ms timer does not require a paint until the displayed second changes.
func countdownChanged(previous, current string) bool { return previous != current }

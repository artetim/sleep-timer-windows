package main

import (
	"testing"
	"time"
)

func TestMinutesLimits(t *testing.T) {
	for _, tc := range []struct {
		minutes int
		valid   bool
	}{{0, false}, {1, true}, {10, true}, {30, true}, {60, true}, {120, true}, {1440, true}, {1441, false}} {
		if validMinutes(tc.minutes) != tc.valid {
			t.Errorf("validMinutes(%d) = %v", tc.minutes, !tc.valid)
		}
	}
}
func TestTimeFormat(t *testing.T) {
	cases := map[int]string{-2: "00:00:00", 0: "00:00:00", 1: "00:00:01", 60: "00:01:00", 3600: "01:00:00", 86400: "24:00:00"}
	for s, want := range cases {
		if got := formatSeconds(s); got != want {
			t.Errorf("formatSeconds(%d): %s != %s", s, got, want)
		}
	}
}
func TestCountdownCeilingAndExpiry(t *testing.T) {
	now := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		d    time.Duration
		want int
	}{{time.Minute, 60}, {time.Minute - time.Nanosecond, 60}, {500 * time.Millisecond, 1}, {0, 0}, {-time.Nanosecond, 0}}
	for _, tc := range cases {
		if got := secondsRemaining(now, now.Add(tc.d)); got != tc.want {
			t.Errorf("duration %s got %d want %d", tc.d, got, tc.want)
		}
	}
}

func TestCountdownDoesNotRedrawWhenSecondUnchanged(t *testing.T) {
	cases := []struct {
		old, next string
		want      bool
	}{
		{"00:01:00", "00:01:00", false},
		{"00:01:00", "00:00:59", true},
		{"00:00:59", "00:00:59", false},
		{"00:00:01", "00:00:00", true},
	}
	for _, tc := range cases {
		if got := countdownChanged(tc.old, tc.next); got != tc.want {
			t.Errorf("countdownChanged(%q, %q) = %v; want %v", tc.old, tc.next, got, tc.want)
		}
	}
}

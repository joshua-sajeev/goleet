package main

import (
	"strings"
	"time"
)

// calculateNextReview implements a simple SM-2-style spaced repetition
// schedule based on the confidence rating the user gives after a review.
func calculateNextReview(confidence string, currentInterval int, currentStreak int) (time.Time, int, int) {
	confidence = strings.ToLower(strings.TrimSpace(confidence))
	today := time.Now()
	var newInterval, newStreak int

	switch confidence {
	case "forgot", "todo":
		newInterval = 1
		newStreak = 0
	case "weak":
		newInterval = 3
		newStreak = 1
	case "good":
		if currentInterval == 0 {
			newInterval = 4
		} else {
			newInterval = currentInterval * 2
		}
		newStreak = currentStreak + 1
	case "easy":
		if currentInterval == 0 {
			newInterval = 7
		} else {
			newInterval = currentInterval * 3
		}
		newStreak = currentStreak + 1
	default:
		newInterval = 1
		newStreak = 0
	}

	if newInterval > 180 {
		newInterval = 180
	}

	return today.AddDate(0, 0, newInterval), newInterval, newStreak
}

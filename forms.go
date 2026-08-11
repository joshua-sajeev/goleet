package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

// newProblemAnswers holds the raw string answers collected from the
// "new problem" form before they're converted into a Problem.
type newProblemAnswers struct {
	Number     string
	Title      string
	Topics     string
	Difficulty string
	Link       string
	Desc       string
}

func requiredValidator(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("this field is required")
	}
	return nil
}

// newProblemForm builds the multi-group huh.Form used to log a new
// LeetCode problem into the vault.
func newProblemForm(a *newProblemAnswers) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Problem Number").
				Placeholder("217").
				Value(&a.Number).
				Validate(requiredValidator),

			huh.NewInput().
				Title("Problem Title").
				Placeholder("Contains Duplicate").
				Value(&a.Title).
				Validate(requiredValidator),

			huh.NewInput().
				Title("Topics (comma separated)").
				Placeholder("Array, HashMap").
				Value(&a.Topics),

			huh.NewSelect[string]().
				Title("Difficulty").
				Options(
					huh.NewOption("Easy", "Easy"),
					huh.NewOption("Medium", "Medium"),
					huh.NewOption("Hard", "Hard"),
				).
				Value(&a.Difficulty),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("LeetCode Link").
				Placeholder("https://leetcode.com/problems/contains-duplicate/").
				Value(&a.Link),

			huh.NewText().
				Title("Description").
				Placeholder("Paste the LeetCode description here").
				Lines(12).
				CharLimit(20000).
				Value(&a.Desc),
		),
	).WithTheme(huh.ThemeCharm())
}

// reviewForm builds the single-question form shown after attempting a
// problem, asking the user to self-rate how it went.
func reviewForm(confidence *string) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("How did it go?").
				Options(
					huh.NewOption("Easy — nailed it instantly", "Easy"),
					huh.NewOption("Good — solved with minor hesitation", "Good"),
					huh.NewOption("Weak — got there but struggled", "Weak"),
					huh.NewOption("Forgot — couldn't solve it", "Forgot"),
				).
				Value(confidence),
		),
	).WithTheme(huh.ThemeCharm())
}

// specificNumberForm builds the single-input form used to look up a
// problem by number for an out-of-schedule practice review.
func specificNumberForm(number *string) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Problem Number").
				Placeholder("217").
				Value(number).
				Validate(requiredValidator),
		),
	).WithTheme(huh.ThemeCharm())
}

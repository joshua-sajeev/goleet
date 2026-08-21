package main

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/huh"
)

// fuzzyMatch returns a score for how well query matches text.
// Higher score = better match. Returns 0 if no match.
// Matching is case-insensitive and rewards:
// - exact substring matches (highest score)
// - prefix matches
// - all characters of query appearing in order in text
func fuzzyMatch(query, text string) int {
	query = strings.ToLower(query)
	text = strings.ToLower(text)

	if query == "" {
		return 0
	}
	if query == text {
		return 10000 // exact match
	}
	if strings.Contains(text, query) {
		return 5000 // substring match
	}
	if strings.HasPrefix(text, query) {
		return 3000 // prefix match
	}

	// Check if all characters of query appear in order in text
	queryIdx := 0
	for i := 0; i < len(text) && queryIdx < len(query); i++ {
		if text[i] == query[queryIdx] {
			queryIdx++
		}
	}
	if queryIdx == len(query) {
		// All characters matched in order; score based on character span
		return 1000 / (len(text) / len(query))
	}

	return 0 // no match
}

// searchResult bundles a problem with its fuzzy match score.
type searchResult struct {
	Row   dashboardRow
	Score int
}

// fuzzySearchProblems searches all problems in the vault against a query
// and returns results sorted by relevance (highest score first).
func fuzzySearchProblems(dir, query string, allRows []dashboardRow) []searchResult {
	if query == "" {
		// Empty query: return all rows in order
		results := make([]searchResult, len(allRows))
		for i, row := range allRows {
			results[i] = searchResult{Row: row, Score: 0}
		}
		return results
	}

	var results []searchResult

	for _, row := range allRows {
		// Score against: number, title, difficulty, topics
		score := 0
		score = maxInt(score, fuzzyMatch(query, row.Number))
		score = maxInt(score, fuzzyMatch(query, row.Title))
		score = maxInt(score, fuzzyMatch(query, row.Problem.Difficulty))
		score = maxInt(score, fuzzyMatch(query, strings.Join(row.Problem.Topics, " ")))

		if score > 0 {
			results = append(results, searchResult{
				Row:   row,
				Score: score,
			})
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// createSearchInput creates a new text input for dashboard search
func createSearchInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "Type to filter... (esc to clear)"
	ti.Focus()
	return ti
}

// searchForm creates a huh form for entering a search query.
func searchForm(query *string) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("Search Problems").
				Placeholder("e.g., 'array', '217', 'hash map'").
				Value(query),
		),
	).WithTheme(huh.ThemeCharm())
}

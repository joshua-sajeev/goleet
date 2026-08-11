package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ensureVaultDir makes sure the vault directory exists on disk.
func ensureVaultDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

// parseNote reads an Obsidian markdown note and splits it into its
// frontmatter (as a Problem) and body content.
func parseNote(filePath string) (Problem, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Problem{}, "", err
	}
	defer file.Close()

	var p Problem
	var bodyBuilder strings.Builder
	scanner := bufio.NewScanner(file)

	inFrontmatter := false
	inTopics := false
	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()

		if line == "---" {
			if lineCount == 0 {
				inFrontmatter = true
				lineCount++
				continue
			} else if inFrontmatter {
				inFrontmatter = false
				lineCount++
				continue
			}
		}

		if inFrontmatter {
			if strings.HasPrefix(strings.ToLower(line), "topics:") {
				inTopics = true
				continue
			}
			if inTopics {
				if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "  - ") {
					topic := strings.TrimPrefix(strings.TrimSpace(line), "- ")
					p.Topics = append(p.Topics, topic)
					continue
				}
				inTopics = false
			}

			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(strings.Trim(parts[1], "\""))

				switch key {
				case "Last Reviewed":
					p.LastReviewed, _ = time.Parse("2006-01-02", value)
				case "Next Review":
					p.NextReview, _ = time.Parse("2006-01-02", value)
				case "Confidence":
					p.Confidence = value
				case "Difficulty":
					p.Difficulty = value
				case "Attempts":
					p.Attempts, _ = strconv.Atoi(value)
				case "Streak":
					p.Streak, _ = strconv.Atoi(value)
				case "Interval":
					p.Interval, _ = strconv.Atoi(value)
				case "Link":
					p.Link = value
				}
			}
		} else {
			bodyBuilder.WriteString(line + "\n")
		}
		lineCount++
	}

	return p, strings.TrimPrefix(bodyBuilder.String(), "\n"), scanner.Err()
}

// saveNote writes a Problem's frontmatter plus body back to disk.
func saveNote(filePath string, p Problem, body string) error {
	const dateLayout = "2006-01-02"
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("Last Reviewed: %s\n", p.LastReviewed.Format(dateLayout)))
	sb.WriteString(fmt.Sprintf("Next Review: %s\n", p.NextReview.Format(dateLayout)))
	sb.WriteString(fmt.Sprintf("Confidence: %s\n", p.Confidence))

	sb.WriteString("topics:\n")
	for _, t := range p.Topics {
		sb.WriteString(fmt.Sprintf("  - %s\n", t))
	}

	sb.WriteString(fmt.Sprintf("Difficulty: %s\n", p.Difficulty))
	sb.WriteString(fmt.Sprintf("Attempts: %d\n", p.Attempts))
	sb.WriteString(fmt.Sprintf("Streak: %d\n", p.Streak))
	sb.WriteString(fmt.Sprintf("Interval: %d\n", p.Interval))
	sb.WriteString(fmt.Sprintf("Link: %s\n", p.Link))
	sb.WriteString("---\n")

	fullContent := sb.String() + body
	return os.WriteFile(filePath, []byte(fullContent), 0o644)
}

// dashboardStats returns the total number of tracked problems and how
// many are currently due for review.
func dashboardStats(dir string) (total, due int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0
	}

	today := time.Now().Truncate(24 * time.Hour)
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".md" {
			continue
		}
		total++
		p, _, err := parseNote(filepath.Join(dir, e.Name()))
		if err == nil {
			noteDate := p.NextReview.Truncate(24 * time.Hour)
			if noteDate.Before(today) || noteDate.Equal(today) {
				due++
			}
		}
	}
	return total, due
}

// dueFileNames returns the file names (not full paths) of every note
// whose Next Review date is today or earlier.
func dueFileNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	today := time.Now().Truncate(24 * time.Hour)
	var due []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".md" {
			continue
		}
		p, _, err := parseNote(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		noteDate := p.NextReview.Truncate(24 * time.Hour)
		if noteDate.Before(today) || noteDate.Equal(today) {
			due = append(due, e.Name())
		}
	}
	return due, nil
}

// findByNumber locates a note by its LeetCode problem number prefix,
// e.g. "217" matches "217-contains-duplicate.md". Returns the file name.
func findByNumber(dir, number string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), number+"-") {
			return e.Name(), nil
		}
	}
	return "", fmt.Errorf("no problem found with number %q", number)
}

// capitalize uppercases just the first letter, e.g. "medium" -> "Medium".
func capitalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

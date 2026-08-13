package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// dashboardRow bundles a parsed Problem with the file it came from and
// a display title pulled from the note body (frontmatter has no title
// field — the filename slug and the "# Title" heading are the only
// places it lives).
type dashboardRow struct {
	FileName string
	Number   string
	Title    string
	Problem  Problem
}

var dashHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))

// loadDashboardRows reads every note in the vault and returns them
// sorted by Next Review date, soonest/most-overdue first. If dueOnly
// is true, only notes due today or earlier are included.
func loadDashboardRows(dir string, dueOnly bool) ([]dashboardRow, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	today := time.Now().Truncate(24 * time.Hour)
	var rows []dashboardRow

	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".md" {
			continue
		}
		fullPath := filepath.Join(dir, e.Name())
		p, body, err := parseNote(fullPath)
		if err != nil {
			continue
		}

		noteDate := p.NextReview.Truncate(24 * time.Hour)
		if dueOnly && noteDate.After(today) {
			continue
		}

		number, slugTitle := splitFileName(e.Name())
		title := extractTitle(body)
		if title == "" {
			title = slugTitle
		}

		rows = append(rows, dashboardRow{
			FileName: e.Name(),
			Number:   number,
			Title:    title,
			Problem:  p,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Problem.NextReview.Before(rows[j].Problem.NextReview)
	})

	return rows, nil
}

// splitFileName pulls the leading problem number and a human-readable
// title guess out of a "217-contains-duplicate.md" style filename.
func splitFileName(fileName string) (number, title string) {
	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	parts := strings.SplitN(base, "-", 2)
	if len(parts) != 2 {
		return base, base
	}
	number = parts[0]
	words := strings.Split(parts[1], "-")
	for i, w := range words {
		if w == "" {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return number, strings.Join(words, " ")
}

// extractTitle finds the first Markdown H1 ("# Title") in a note body.
func extractTitle(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

// statusFor returns a plain-text "3d overdue / due today / in 5d" label
// for how a note's Next Review date relates to today.
//
// Deliberately plain, no lipgloss.Render() here: the table library
// truncates cell content with runewidth.Truncate(value, width, "…"),
// which slices the raw string by byte/rune count with no awareness of
// ANSI escape codes. A colored string that gets cut mid-escape-sequence
// loses its closing reset code, and the color bleeds into every cell
// rendered after it until something else happens to reset it — which
// is why the row highlight color seemed to "smear" across other rows
// as the cursor moved and different-length status strings got truncated
// at different points. Keeping cell values plain avoids the whole class
// of bug; row/cursor highlighting is still handled safely by the
// table's own Selected style.
func statusFor(p Problem) string {
	today := time.Now().Truncate(24 * time.Hour)
	next := p.NextReview.Truncate(24 * time.Hour)
	days := int(next.Sub(today).Hours() / 24)

	switch {
	case days < 0:
		return fmt.Sprintf("%dd overdue", -days)
	case days == 0:
		return "due today"
	default:
		return fmt.Sprintf("in %dd", days)
	}
}

// buildDashboardTable turns rows into a ready-to-render bubbles/table.
func buildDashboardTable(rows []dashboardRow, height int) table.Model {
	columns := []table.Column{
		{Title: "Title", Width: 30},
		{Title: "Diff", Width: 7},
		{Title: "Topics", Width: 24},
		{Title: "Streak", Width: 6},
		{Title: "Interval", Width: 8},
		{Title: "Next Review", Width: 13},
		{Title: "Status", Width: 13},
	}

	trows := make([]table.Row, 0, len(rows))
	for _, r := range rows {
		p := r.Problem
		trows = append(trows, table.Row{
			truncate(r.Title, 30),
			p.Difficulty,
			truncate(strings.Join(p.Topics, ", "), 24),
			fmt.Sprintf("%d", p.Streak),
			fmt.Sprintf("%dd", p.Interval),
			p.NextReview.Format("2006-01-02"),
			statusFor(p),
		})
	}

	if height < 5 {
		height = 5
	}
	if height > 20 {
		height = 20
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(trows),
		table.WithFocused(true),
		table.WithHeight(height),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("212"))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("212")).
		Bold(true)
	t.SetStyles(s)

	return t
}

// truncate shortens s to at most max runes, adding an ellipsis if cut.
// Table library truncation is byte-based on top of this, but since this
// runs on plain (non-ANSI) text the two never conflict.
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 1 {
		return string(runes[:max])
	}
	return string(runes[:max-1]) + "…"
}

// --- wiring into the model ---

// openDashboard loads rows (all or due-only), builds the table, and
// switches into the given dashboard state.
func (m *model) openDashboard(dueOnly bool, state sessionState) (tea.Model, tea.Cmd) {
	rows, err := loadDashboardRows(m.vaultDir, dueOnly)
	if err != nil {
		m.err = err
		return m, nil
	}
	if len(rows) == 0 {
		if dueOnly {
			m.message = "All caught up! No problems due for review today."
		} else {
			m.message = "No problems tracked yet. Add one from the menu."
		}
		m.state = stateDone
		return m, nil
	}

	height := m.height - 10
	m.dashboardRows = rows
	m.dashboardTable = buildDashboardTable(rows, height)
	m.state = state
	return m, nil
}

// updateDashboard handles input while a dashboard table is on screen:
// arrow keys move the cursor (handled by the table itself), enter
// jumps into a review of the highlighted problem, esc/q returns to
// the menu.
func (m *model) updateDashboard(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc", "q":
			m.state = stateMenu
			return m, nil
		case "enter":
			idx := m.dashboardTable.Cursor()
			if idx < 0 || idx >= len(m.dashboardRows) {
				return m, nil
			}
			m.dueFiles = []string{m.dashboardRows[idx].FileName}
			m.reviewIdx = 0
			return m.startReview()
		}
	}

	var cmd tea.Cmd
	m.dashboardTable, cmd = m.dashboardTable.Update(msg)
	return m, cmd
}

// dashboardView renders the table plus a header/help footer. title
// distinguishes the "all problems" and "due today" screens.
func (m *model) dashboardView(title string) string {
	var b strings.Builder
	b.WriteString(dashHeaderStyle.Render(title) + "\n\n")
	b.WriteString(m.dashboardTable.View() + "\n\n")
	b.WriteString(helpStyle.Render("↑/↓ move  •  enter review selected  •  esc/q back to menu"))
	return docStyle.Render(b.String())
}

package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Problem maps directly to the Obsidian frontmatter and structure.
type Problem struct {
	LastReviewed time.Time
	NextReview   time.Time
	Confidence   string
	Topics       []string
	Difficulty   string
	Attempts     int
	Streak       int
	Interval     int
	Link         string
}

type sessionState int

const (
	stateMenu sessionState = iota
	stateNewProblem
	stateReviewForm
	stateSpecificInput
	stateDashboardAll
	stateDashboardDue
	stateDone
)

// --- styles ---

var (
	docStyle       = lipgloss.NewStyle().Margin(1, 2)
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	dashboardStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	menuItemStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	successStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	helpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
)

type model struct {
	vaultDir string
	state    sessionState
	width    int
	height   int

	// dashboard
	total int
	due   int

	// new problem
	newForm    *huh.Form
	newProblem newProblemAnswers

	// review session (shared by "due" and "specific" flows)
	dueFiles    []string
	reviewIdx   int
	current     Problem
	currentBody string
	currentFile string
	reviewForm  *huh.Form
	confidence  string

	// specific-problem lookup
	specificForm *huh.Form
	specificNum  string

	// dashboards
	dashboardTable     table.Model
	dashboardRows      []dashboardRow
	dashboardAllRows   []dashboardRow // original unfiltered rows
	dashboardSearching bool
	dashboardSearch    textinput.Model
	dashboardQuery     string

	message string
	err     error
}

func initialModel(vaultDir string) *model {
	m := &model{
		vaultDir: vaultDir,
		state:    stateMenu,
	}
	m.refreshDashboard()
	return m
}

func (m *model) Init() tea.Cmd {
	return nil
}

// --- update ---
//
// NOTE: every method here uses a pointer receiver (*model), not a value
// receiver. huh.Form fields are bound with e.g. Value(&m.newProblem.Number) —
// a pointer into this specific model instance. If Update used a value
// receiver, Bubble Tea's "return a new model each call" pattern would copy
// the whole struct on every keystroke, and the form's bound pointer would
// keep writing into an orphaned copy that the rest of the app never reads
// again (symptom: fields silently stay empty). Pointer receivers keep one
// stable address for the model's entire lifetime, so the binding stays valid.

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.state == stateMenu {
			return m.handleMenuKey(msg)
		}
		if m.state == stateDone {
			m.state = stateMenu
			m.message = ""
			m.err = nil
			m.refreshDashboard()
			return m, nil
		}
	}

	// Forward anything else (keys not consumed above, blink ticks, etc.)
	// to whichever huh.Form (or dashboard table) is currently active.
	switch m.state {
	case stateNewProblem:
		return m.updateNewForm(msg)
	case stateReviewForm:
		return m.updateReviewForm(msg)
	case stateSpecificInput:
		return m.updateSpecificForm(msg)
	case stateDashboardAll, stateDashboardDue:
		return m.updateDashboard(msg)
	}

	return m, nil
}

func (m *model) handleMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.err = nil
	switch msg.String() {
	case "1":
		m.state = stateNewProblem
		m.newProblem = newProblemAnswers{}
		m.newForm = newProblemForm(&m.newProblem)
		return m, m.newForm.Init()

	case "2":
		return m.openDashboard(false, stateDashboardAll)

	case "3":
		return m.openDashboard(true, stateDashboardDue)

	case "0", "q":
		return m, tea.Quit
	}
	return m, nil
}

// startReview loads m.dueFiles[m.reviewIdx] off disk and wires up a
// fresh confidence-rating form for it.
func (m *model) startReview() (tea.Model, tea.Cmd) {
	fileName := m.dueFiles[m.reviewIdx]
	filePath := filepath.Join(m.vaultDir, fileName)

	p, body, err := parseNote(filePath)
	if err != nil {
		m.err = err
		m.state = stateMenu
		return m, nil
	}

	m.current = p
	m.currentBody = body
	m.currentFile = filePath
	m.confidence = ""
	m.reviewForm = reviewForm(&m.confidence)
	m.state = stateReviewForm
	return m, m.reviewForm.Init()
}

func (m *model) updateNewForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	form, cmd := m.newForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.newForm = f
	}

	switch m.newForm.State {
	case huh.StateCompleted:
		if err := m.saveNewProblem(); err != nil {
			m.err = err
		}
		m.refreshDashboard()
		m.state = stateDone
		return m, nil
	case huh.StateAborted:
		m.state = stateMenu
		return m, nil
	}
	return m, cmd
}

func (m *model) updateReviewForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	form, cmd := m.reviewForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.reviewForm = f
	}

	switch m.reviewForm.State {
	case huh.StateCompleted:
		if err := m.saveReview(); err != nil {
			m.err = err
		}
		m.reviewIdx++
		if m.err == nil && m.reviewIdx < len(m.dueFiles) {
			return m.startReview()
		}
		if m.err == nil {
			m.message = "Review session complete. Great job!"
		}
		m.refreshDashboard()
		m.state = stateDone
		return m, nil
	case huh.StateAborted:
		m.refreshDashboard()
		m.state = stateMenu
		return m, nil
	}
	return m, cmd
}

func (m *model) updateSpecificForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	form, cmd := m.specificForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.specificForm = f
	}

	switch m.specificForm.State {
	case huh.StateCompleted:
		fileName, err := findByNumber(m.vaultDir, m.specificNum)
		if err != nil {
			m.err = err
			m.state = stateMenu
			return m, nil
		}
		m.dueFiles = []string{fileName}
		m.reviewIdx = 0
		return m.startReview()
	case huh.StateAborted:
		m.state = stateMenu
		return m, nil
	}
	return m, cmd
}

// --- persistence helpers ---

func (m *model) saveNewProblem() error {
	a := m.newProblem

	var topics []string
	for _, t := range strings.Split(a.Topics, ",") {
		if trimmed := strings.TrimSpace(t); trimmed != "" {
			topics = append(topics, trimmed)
		}
	}

	p := Problem{
		LastReviewed: time.Now(),
		NextReview:   time.Now(),
		Confidence:   "Todo",
		Topics:       topics,
		Difficulty:   capitalize(a.Difficulty),
		Attempts:     0,
		Streak:       0,
		Interval:     0,
		Link:         a.Link,
	}

	slug := strings.ReplaceAll(strings.ToLower(a.Title), " ", "-")
	fileName := fmt.Sprintf("%s-%s.md", a.Number, slug)
	filePath := filepath.Join(m.vaultDir, fileName)

	body := fmt.Sprintf("\n# %s\n\n## Description\n\n%s\n\n", a.Title, a.Desc)

	if err := saveNote(filePath, p, body); err != nil {
		return err
	}
	m.message = fmt.Sprintf("Created %s", fileName)
	return nil
}

func (m *model) saveReview() error {
	nextReviewDate, newInterval, newStreak := calculateNextReview(m.confidence, m.current.Interval, m.current.Streak)

	m.current.Confidence = m.confidence
	m.current.LastReviewed = time.Now()
	m.current.NextReview = nextReviewDate
	m.current.Interval = newInterval
	m.current.Streak = newStreak
	m.current.Attempts++

	if err := saveNote(m.currentFile, m.current, m.currentBody); err != nil {
		return err
	}
	m.message = fmt.Sprintf("Updated! Next review: %s (in %d days)", m.current.NextReview.Format("2006-01-02"), newInterval)
	return nil
}

func (m *model) refreshDashboard() {
	total, due := dashboardStats(m.vaultDir)
	m.total = total
	m.due = due
}

// --- view ---

func (m *model) View() string {
	switch m.state {
	case stateNewProblem:
		return docStyle.Render(m.newForm.View())

	case stateReviewForm:
		header := fmt.Sprintf(
			"Reviewing %d of %d: %s\nTopics: %s  |  Difficulty: %s\nStreak: %d  |  Attempts: %d\n\n",
			m.reviewIdx+1, len(m.dueFiles), filepath.Base(m.currentFile),
			strings.Join(m.current.Topics, ", "), m.current.Difficulty,
			m.current.Streak, m.current.Attempts,
		)
		return docStyle.Render(dashboardStyle.Render(header) + m.reviewForm.View())

	case stateSpecificInput:
		return docStyle.Render(m.specificForm.View())

	case stateDashboardAll:
		return m.dashboardView("📊 All Problems")

	case stateDashboardDue:
		return m.dashboardView("⏰ Due Today")

	case stateDone:
		var b strings.Builder
		if m.err != nil {
			b.WriteString(errorStyle.Render("Error: "+m.err.Error()) + "\n\n")
		} else {
			b.WriteString(successStyle.Render(m.message) + "\n\n")
		}
		b.WriteString(helpStyle.Render("Press any key to return to the menu."))
		return docStyle.Render(b.String())

	default:
		return m.menuView()
	}
}

func (m *model) menuView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("📘 LeetCode Obsidian CLI") + "\n\n")
	b.WriteString(dashboardStyle.Render(fmt.Sprintf("%d total problems tracked  |  %d due for review today", m.total, m.due)) + "\n\n")
	b.WriteString(menuItemStyle.Render("1. New Problem") + "\n")
	b.WriteString(menuItemStyle.Render("2. Dashboard — All Problems") + "\n")
	b.WriteString(menuItemStyle.Render("3. Dashboard — Due Today") + "\n")
	b.WriteString(menuItemStyle.Render("0. Exit") + "\n\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render("Error: "+m.err.Error()) + "\n\n")
	}

	b.WriteString(helpStyle.Render("Choose an option (1 / 2 / 3 / 0)"))
	return docStyle.Render(b.String())
}

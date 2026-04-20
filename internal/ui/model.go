package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/saddatahmad19/grove/internal/agent"
	"github.com/saddatahmad19/grove/internal/config"
	"github.com/saddatahmad19/grove/internal/worktree"
)

type mode int

const (
	modeNavigate mode = iota
	modeCreatePrompt
)

type viewState int

const (
	stateNoRepo viewState = iota
	stateRepoLoaded
)

type worktreeItem struct{ wt worktree.Worktree }

func (i worktreeItem) Title() string { return i.wt.Name }
func (i worktreeItem) Description() string {
	extra := i.wt.Branch
	if i.wt.Head != "" {
		extra = fmt.Sprintf("%s • %s", extra, i.wt.Head)
	}
	return fmt.Sprintf("%s", extra)
}
func (i worktreeItem) FilterValue() string { return strings.Join([]string{i.wt.Name, i.wt.Branch, i.wt.Head, i.wt.Path}, " ") }

type Model struct {
	cfg       config.Config
	list      list.Model
	mode      mode
	viewState viewState
	agents    []agent.Agent
	ready     bool
	status    string
	target    string
	err       error
	prompt    string
}

func NewModel(cfg config.Config) Model {
	m := Model{cfg: cfg, mode: modeNavigate, agents: agent.Defaults(), status: "ready"}
	m.initList(nil)
	m.loadState()
	return m
}

func (m *Model) initList(items []list.Item) {
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.SetShowHelp(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Title = "Grove Worktrees"
	m.list = l
}

func (m *Model) setStatus(format string, args ...any) { m.status = fmt.Sprintf(format, args...) }

func (m *Model) loadState() {
	m.err = nil
	wts, err := worktree.LoadAll(m.cfg.Root)
	if err != nil {
		m.err = err
		m.status = err.Error()
		m.viewState = stateNoRepo
		m.initList(nil)
		return
	}
	items := make([]list.Item, 0, len(wts))
	for _, wt := range wts {
		items = append(items, worktreeItem{wt: wt})
	}
	m.viewState = stateRepoLoaded
	m.setStatus("%d worktree(s) loaded", len(items))
	m.initList(items)
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r", "f5":
			m.setStatus("refreshing worktrees...")
			m.loadState()
			return m, nil
		case "n", "c":
			if m.viewState == stateRepoLoaded {
				m.mode = modeCreatePrompt
				m.prompt = ""
				m.setStatus("enter a new worktree name, then press enter")
			}
		case "esc":
			m.mode = modeNavigate
		case "enter":
			if m.mode == modeCreatePrompt {
				name := strings.TrimSpace(m.prompt)
				if name == "" {
					m.setStatus("worktree name cannot be empty")
					return m, nil
				}
				m.setStatus("create worktree requested: %s", name)
				m.mode = modeNavigate
				m.prompt = ""
				return m, nil
			}
			if m.viewState == stateRepoLoaded {
				if it, ok := m.list.SelectedItem().(worktreeItem); ok {
					m.target = it.wt.Path
					m.setStatus("selected worktree: %s", it.wt.Name)
				}
			}
		case "tab":
			if len(m.agents) > 0 {
				m.status = "selected agent: " + m.agents[0].Name
			}
		default:
			if m.mode == modeCreatePrompt {
				if msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete {
					if len(m.prompt) > 0 {
						m.prompt = m.prompt[:len(m.prompt)-1]
					}
				} else if len(msg.Runes) > 0 {
					m.prompt += string(msg.Runes)
				}
				return m, nil
			}
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-10)
	}

	if m.viewState == stateRepoLoaded && m.mode == modeNavigate {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) View() string {
	if m.viewState == stateNoRepo {
		return m.renderNoRepo()
	}
	return m.renderRepo()
}

func (m Model) renderNoRepo() string {
	title := lipgloss.NewStyle().Bold(true).Render("Grove")
	panel := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"No Git repository detected in this folder.",
		fmt.Sprintf("Current path: %s", m.cfg.Root),
		"Open Grove from inside a repo or set GROVE_ROOT to a repository path.",
		"Press r to retry after changing folders.",
	)
	return lipgloss.NewStyle().Padding(1, 2).Render(panel)
}

func (m Model) renderRepo() string {
	modeLabel := "Navigate"
	if m.mode == modeCreatePrompt {
		modeLabel = "Create worktree"
	}
	footer := "q quit • r refresh • n new worktree • enter select/confirm • esc cancel"
	prompt := ""
	if m.mode == modeCreatePrompt {
		prompt = fmt.Sprintf("Create worktree name: %s", m.prompt+"_")
	}
	statusBox := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("Status: " + m.status)
	parts := []string{
		lipgloss.NewStyle().Bold(true).Render("Grove"),
		fmt.Sprintf("Mode: %s", modeLabel),
		fmt.Sprintf("Root: %s", filepath.Clean(m.cfg.Root)),
		statusBox,
	}
	if prompt != "" {
		parts = append(parts, prompt)
	}
	parts = append(parts, m.list.View(), footer)
	return lipgloss.NewStyle().Padding(1).Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

package app

import (
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/saddatahmad19/grove/internal/config"
	"github.com/saddatahmad19/grove/internal/ui"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) > 0 {
		switch args[0] {
		case "--help", "-h", "help":
			_, err := fmt.Fprint(stdout, `grove - multi-agent git worktree TUI

Usage:
  grove [flags]

Flags:
  -h, --help   show help

Environment:
  GROVE_ROOT      Override repository root (default: current directory)
  GROVE_CONFIG    Optional config file path
`)
			return err
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	m := ui.NewModel(cfg)
	p := tea.NewProgram(m, tea.WithOutput(stdout), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		if strings.Contains(err.Error(), "program was killed") {
			return nil
		}
		return err
	}
	return nil
}

package agent

type Agent struct {
	Name    string
	Command []string
}

func Defaults() []Agent {
	return []Agent{
		{Name: "hermes", Command: []string{"hermes"}},
		{Name: "claude-code", Command: []string{"claude"}},
		{Name: "codex", Command: []string{"codex", "exec"}},
		{Name: "opencode", Command: []string{"opencode", "run"}},
	}
}

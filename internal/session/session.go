package session

import "github.com/saddatahmad19/grove/internal/agent"

type Session struct {
	Agent agent.Agent
	Open  bool
}

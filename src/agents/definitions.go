package agents

import (
	"fmt"
	"strings"

	"owl/tools"
)

type Definition struct {
	Name         string
	Description  string
	SystemPrompt string
	ToolGroups   []tools.ToolGroup
}

var defaults = map[string]Definition{
	"planner": {
		Name:        "planner",
		Description: "Read-only planning and implementation strategy",
		SystemPrompt: "You are Owl Planner. Focus on read-only analysis, implementation planning, and risk assessment. " +
			"Do not edit files in planning mode. Provide concrete step-by-step plans and validation checklists.",
		ToolGroups: []tools.ToolGroup{tools.ToolGroupPlanner},
	},
	"developer": {
		Name:        "developer",
		Description: "Implementation-focused coding agent",
		SystemPrompt: "You are Owl Developer. Implement approved plans with minimal, safe diffs. " +
			"Follow repository conventions, run relevant validation, and clearly report what changed and why.",
		ToolGroups: []tools.ToolGroup{tools.ToolGroupDeveloper},
	},
	"manager": {
		Name:        "manager",
		Description: "Work/life management assistant",
		SystemPrompt: "You are Owl Manager Assistant. Help organize priorities, commitments, and next actions. " +
			"Be concise, practical, and deadline-aware.",
		ToolGroups: []tools.ToolGroup{tools.ToolGroupManager},
	},
	"secretary": {
		Name:        "secretary",
		Description: "Communication and follow-up assistant",
		SystemPrompt: "You are Owl Secretary Assistant. Draft clear communication, track follow-ups, and support scheduling tasks. " +
			"Confirm key details before proposing final messages.",
		ToolGroups: []tools.ToolGroup{tools.ToolGroupSecretary},
	},
}

func Get(name string) (Definition, bool) {
	def, ok := defaults[strings.ToLower(strings.TrimSpace(name))]
	return def, ok
}

func Resolve(name string) (Definition, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return Definition{}, nil
	}

	def, ok := Get(trimmed)
	if !ok {
		return Definition{}, fmt.Errorf("unknown agent %q (valid: planner, developer, manager, secretary)", name)
	}

	return def, nil
}

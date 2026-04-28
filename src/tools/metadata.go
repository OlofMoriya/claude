package tools

import "strings"

type ToolGroup string

const (
	ToolGroupChat      ToolGroup = "chat"
	ToolGroupDev       ToolGroup = "dev"
	ToolGroupDeveloper ToolGroup = "developer"
	ToolGroupEmail     ToolGroup = "email"
	ToolGroupManager   ToolGroup = "manager"
	ToolGroupPlanner   ToolGroup = "planner"
	ToolGroupSecretary ToolGroup = "secretary"
	ToolGroupWriter    ToolGroup = "writer"
	ToolGroupManage    ToolGroup = "manage"
)

type ToolDependency string

const (
	ToolDependencyLocalExec      ToolDependency = "local_exec"
	ToolDependencyRemoteExec     ToolDependency = "remote_exec"
	ToolDependencyTUIInteractive ToolDependency = "tui_interactive"
)

func NormalizeToolGroup(input string) ToolGroup {
	v := strings.ToLower(strings.TrimSpace(input))
	switch v {
	case "":
		return ""
	case "secretery":
		return ToolGroupSecretary
	default:
		return ToolGroup(v)
	}
}

func ParseToolGroups(inputs []string) []ToolGroup {
	groups := make([]ToolGroup, 0, len(inputs))
	seen := map[ToolGroup]bool{}
	for _, raw := range inputs {
		g := NormalizeToolGroup(raw)
		if g == "" || seen[g] {
			continue
		}
		seen[g] = true
		groups = append(groups, g)
	}
	return groups
}

package tools

import "strings"

type ToolGroup string

const (
	ToolGroupDeveloper ToolGroup = "developer"
	ToolGroupManager   ToolGroup = "manager"
	ToolGroupPlanner   ToolGroup = "planner"
	ToolGroupSecretary ToolGroup = "secretary"
)

type ToolDependency string

const (
	ToolDependencyLocalExec      ToolDependency = "local_exec"
	ToolDependencyRemoteExec     ToolDependency = "remote_exec"
	ToolDependencyTUIInteractive ToolDependency = "tui_interactive"
)

func NormalizeToolGroup(input string) ToolGroup {
	v := strings.ToLower(strings.TrimSpace(input))
	if v == "" {
		return ""
	}
	return ToolGroup(v)
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

func ToolGroupsToStrings(groups []ToolGroup) []string {
	result := make([]string, 0, len(groups))
	for _, g := range groups {
		if g == "" {
			continue
		}
		result = append(result, string(g))
	}
	return result
}

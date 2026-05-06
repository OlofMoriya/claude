package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"owl/data"
	"strings"
	"time"
)

const calendarOverviewTimeout = 15 * time.Second

var allowedCalendarQueries = []string{
	"eventsToday",
	"eventsToday+",
	"eventsNow",
	"eventsFrom:",
	"uncompletedTasks",
	"undatedUncompletedTasks",
	"tasksDueBefore:",
	"calendars",
}

var disallowedCalendarQueryFragments = []string{
	"editConfig",
	"editConfigCLI",
	"strEncodings",
}

var allowedCalendarOptions = map[string]bool{
	"-f": true, "-n": true, "-nc": true, "-nrd": true, "-npn": true,
	"-uid": true, "-ea": true, "-eep": true, "-etp": true,
	"-sc": true, "-sd": true, "-sp": true,
	"-li": true, "-ic": true, "-ec": true,
	"-tf": true, "-df": true,
	"-po": true, "-ps": true,
}

type CalendarOverviewTool struct {
	runner func(ctx context.Context, command string, args ...string) (string, string, error)
}

func (tool *CalendarOverviewTool) SetHistory(repo *data.HistoryRepository, context *data.Context) {}

func (tool *CalendarOverviewTool) GetName() string {
	return "calendar_overview"
}

func (tool *CalendarOverviewTool) GetGroups() []ToolGroup {
	return []ToolGroup{ToolGroupManager, ToolGroupSecretary}
}

func (tool *CalendarOverviewTool) GetDefinition() (Tool, string) {
	return Tool{
		Name:         tool.GetName(),
		Description:  "Read-only calendar and task overview via icalBuddy.",
		Groups:       []ToolGroup{ToolGroupManager, ToolGroupSecretary},
		Dependencies: []ToolDependency{ToolDependencyLocalExec},
		InputSchema: InputSchema{
			Type:     "object",
			Required: []string{"query"},
			Properties: map[string]Property{
				"query": {
					Type:        "string",
					Description: "icalBuddy query command, e.g. eventsToday, eventsToday+7, eventsNow, eventsFrom:2026-05-01 to:2026-05-07, uncompletedTasks, tasksDueBefore:today+3, calendars.",
				},
				"options": {
					Type:        "string",
					Description: "Optional safe icalBuddy flags, e.g. -n -li 20 -ic Work",
				},
			},
		},
	}, LOCAL
}

func (tool *CalendarOverviewTool) Run(i map[string]string) (string, error) {
	query := strings.TrimSpace(i["query"])
	if query == "" {
		return "", fmt.Errorf("query is required")
	}
	if err := validateCalendarQuery(query); err != nil {
		return "", err
	}

	options, err := parseAndValidateOptions(i["options"])
	if err != nil {
		return "", err
	}

	args := append(options, query)
	ctx, cancel := context.WithTimeout(context.Background(), calendarOverviewTimeout)
	defer cancel()

	runner := tool.runner
	if runner == nil {
		runner = runCommand
	}

	stdout, stderr, err := runner(ctx, "icalBuddy", args...)
	if err != nil {
		if stderr != "" {
			return "", fmt.Errorf("calendar_overview failed: %s", strings.TrimSpace(stderr))
		}
		return "", fmt.Errorf("calendar_overview failed: %w", err)
	}

	output := strings.TrimSpace(stdout)
	if output == "" {
		return "No events or tasks found for the requested query.", nil
	}

	return output, nil
}

func validateCalendarQuery(query string) error {
	for _, frag := range disallowedCalendarQueryFragments {
		if strings.Contains(query, frag) {
			return fmt.Errorf("query command is not allowed: %s", frag)
		}
	}

	for _, allowed := range allowedCalendarQueries {
		if strings.HasPrefix(query, allowed) {
			return nil
		}
	}

	return fmt.Errorf("query command is not allowed")
}

func parseAndValidateOptions(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	tokens := strings.Fields(raw)
	validated := make([]string, 0, len(tokens))

	for idx := 0; idx < len(tokens); idx++ {
		tok := tokens[idx]
		if !strings.HasPrefix(tok, "-") {
			if idx == 0 {
				return nil, fmt.Errorf("options must start with a flag, got: %s", tok)
			}
			validated = append(validated, tok)
			continue
		}

		if tok == "-V" || tok == "-u" {
			return nil, fmt.Errorf("option %s is not allowed", tok)
		}

		if !allowedCalendarOptions[tok] {
			return nil, fmt.Errorf("option %s is not allowed", tok)
		}
		validated = append(validated, tok)
	}

	return validated, nil
}

func runCommand(ctx context.Context, command string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func init() {
	Register(&CalendarOverviewTool{})
}

package tools

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestValidateCalendarQuery(t *testing.T) {
	t.Parallel()

	if err := validateCalendarQuery("eventsToday+3"); err != nil {
		t.Fatalf("expected valid query, got %v", err)
	}

	if err := validateCalendarQuery("editConfig"); err == nil {
		t.Fatalf("expected disallowed command error")
	}
}

func TestParseAndValidateOptions(t *testing.T) {
	t.Parallel()

	opts, err := parseAndValidateOptions("-n -li 10 -ic Work")
	if err != nil {
		t.Fatalf("expected valid options, got %v", err)
	}
	if len(opts) != 5 {
		t.Fatalf("expected 5 tokens, got %d", len(opts))
	}

	if _, err := parseAndValidateOptions("-u"); err == nil {
		t.Fatalf("expected disallowed option error")
	}
}

func TestCalendarOverviewRunSuccess(t *testing.T) {
	t.Parallel()

	tool := &CalendarOverviewTool{
		runner: func(ctx context.Context, command string, args ...string) (string, string, error) {
			if command != "icalBuddy" {
				t.Fatalf("unexpected command: %s", command)
			}
			if len(args) == 0 || args[len(args)-1] != "eventsToday" {
				t.Fatalf("unexpected args: %v", args)
			}
			return "meeting at 09:00", "", nil
		},
	}

	out, err := tool.Run(map[string]string{"query": "eventsToday"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "meeting") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCalendarOverviewRunFailureIncludesStderr(t *testing.T) {
	t.Parallel()

	tool := &CalendarOverviewTool{
		runner: func(ctx context.Context, command string, args ...string) (string, string, error) {
			return "", "icalBuddy missing permission", errors.New("exit status 1")
		},
	}

	_, err := tool.Run(map[string]string{"query": "eventsToday"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "missing permission") {
		t.Fatalf("expected stderr in error, got: %v", err)
	}
}

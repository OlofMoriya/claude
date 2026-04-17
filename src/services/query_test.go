package services

import (
	"testing"

	"owl/data"
)

func TestFilterStreamHistory_KeepPromptOrResponse(t *testing.T) {
	history := []data.History{
		{Prompt: "first prompt", Response: "", Archived: false},
		{Prompt: "", Response: "first response", Archived: false},
		{Prompt: "second prompt", Response: "second response", Archived: false},
		{Prompt: "", Response: "", Archived: false},
		{Prompt: "archived prompt", Response: "archived response", Archived: true},
	}

	filtered, droppedArchived, droppedEmpty := filterStreamHistory(history)

	if len(filtered) != 3 {
		t.Fatalf("expected 3 history entries after filtering, got %d", len(filtered))
	}

	if droppedArchived != 1 {
		t.Fatalf("expected 1 archived entry dropped, got %d", droppedArchived)
	}

	if droppedEmpty != 1 {
		t.Fatalf("expected 1 empty entry dropped, got %d", droppedEmpty)
	}

	if filtered[0].Prompt != "first prompt" || filtered[0].Response != "" {
		t.Fatalf("unexpected first filtered entry: %+v", filtered[0])
	}

	if filtered[1].Prompt != "" || filtered[1].Response != "first response" {
		t.Fatalf("unexpected second filtered entry: %+v", filtered[1])
	}
}

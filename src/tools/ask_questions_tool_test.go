package tools

import (
	commontypes "owl/common_types"
	"testing"
)

func TestParseQuestionBatch_AcceptsStringOptionsAndDefaults(t *testing.T) {
	t.Parallel()

	raw := `{
  "title": "Daily",
  "questions": [
    {
      "id": "q1",
      "question": "How was today?",
      "options": ["Good", "Okay", "Bad"]
    }
  ]
}`

	batch, err := parseQuestionBatch(raw)
	if err != nil {
		t.Fatalf("expected parse success, got error: %v", err)
	}

	if len(batch.Questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(batch.Questions))
	}

	q := batch.Questions[0]
	if !q.AllowCustom {
		t.Fatalf("expected allow_custom default true")
	}
	if !q.Required {
		t.Fatalf("expected required default true")
	}
	if len(q.Options) != 3 || q.Options[0].Label != "Good" {
		t.Fatalf("unexpected options parsed: %+v", q.Options)
	}
}

func TestParseQuestionBatch_AllowMultiple(t *testing.T) {
	t.Parallel()

	raw := `{
  "questions": [
    {
      "id": "q1",
      "question": "Which apply?",
      "allow_multiple": true,
      "options": ["A", "B", "C"]
    }
  ]
}`

	batch, err := parseQuestionBatch(raw)
	if err != nil {
		t.Fatalf("expected parse success, got error: %v", err)
	}

	if !batch.Questions[0].AllowMultiple {
		t.Fatalf("expected allow_multiple=true")
	}
}

func TestParseQuestionBatchFromInput_TypedFields(t *testing.T) {
	t.Parallel()

	input := map[string]string{
		"Title": "Daily",
		"Questions": `[
			{"id":"q1","question":"How was today?","options":["Good","Okay"],"allow_multiple":false}
		]`,
	}

	batch, err := parseQuestionBatchFromInput(input)
	if err != nil {
		t.Fatalf("expected parse success, got error: %v", err)
	}

	if batch.Title != "Daily" {
		t.Fatalf("expected title Daily, got %q", batch.Title)
	}
	if len(batch.Questions) != 1 || batch.Questions[0].ID != "q1" {
		t.Fatalf("unexpected questions parsed: %+v", batch.Questions)
	}
}

func TestValidateQuestionBatch(t *testing.T) {
	t.Parallel()

	valid := commontypes.QuestionBatchRequest{
		Questions: []commontypes.QuestionItem{{
			ID:       "q1",
			Question: "Pick one",
			Options:  []commontypes.QuestionOption{{Label: "A"}, {Label: "B"}},
		}},
	}

	if err := validateQuestionBatch(valid); err != nil {
		t.Fatalf("expected valid batch, got error: %v", err)
	}

	tooManyOptions := commontypes.QuestionBatchRequest{
		Questions: []commontypes.QuestionItem{{
			ID:       "q1",
			Question: "Pick one",
			Options: []commontypes.QuestionOption{
				{Label: "1"}, {Label: "2"}, {Label: "3"}, {Label: "4"}, {Label: "5"}, {Label: "6"}, {Label: "7"},
			},
		}},
	}
	if err := validateQuestionBatch(tooManyOptions); err == nil {
		t.Fatalf("expected too many options error")
	}
}

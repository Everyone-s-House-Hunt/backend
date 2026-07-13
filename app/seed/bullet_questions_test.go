package seed

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBulletSeedBankIsPlayable(t *testing.T) {
	questions, err := buildBulletQuestionModels()
	if err != nil {
		t.Fatalf("buildBulletQuestionModels returned error: %v", err)
	}
	if len(questions) != 7 {
		t.Fatalf("expected 7 seed questions, got %d", len(questions))
	}

	ids := make(map[string]bool)
	for _, question := range questions {
		if ids[question.ID] {
			t.Fatalf("duplicate seed ID: %s", question.ID)
		}
		ids[question.ID] = true
		if question.GameMode != "bullet" || question.Status != "approved" {
			t.Fatalf("question is not an approved bullet fixture: %#v", question)
		}
		if strings.Contains(question.Body, "5つ") {
			t.Fatalf("question still describes the solo five-answer rule: %s", question.Body)
		}
		var answerData bulletAnswerData
		if err := json.Unmarshal([]byte(question.AnswerData), &answerData); err != nil {
			t.Fatalf("decode %s: %v", question.ID, err)
		}
		if len(answerData.Answers) < 10 {
			t.Fatalf("question %s has only %d answers", question.ID, len(answerData.Answers))
		}
	}
}

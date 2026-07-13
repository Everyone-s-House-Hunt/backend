package service

import (
	"encoding/json"
	"errors"
	"testing"

	"house-hunt/dto"
	"house-hunt/model"
	"house-hunt/utils"
)

type fakeQuestionStore struct {
	created *model.Question
}

func (store *fakeQuestionStore) GetQestions(string, int) ([]model.Question, error) {
	return nil, nil
}

func (store *fakeQuestionStore) CreateQuestion(question *model.Question) error {
	store.created = question
	return nil
}

func (store *fakeQuestionStore) UpdateQuestionStatus(string, string) error {
	return nil
}

func TestCreateBulletQuestionEncodesCanonicalAnswersAndAliases(t *testing.T) {
	store := &fakeQuestionStore{}
	service := NewQuestionService(store)
	answers := make([]dto.BulletAnswerRequest, 10)
	for index := range answers {
		answers[index] = dto.BulletAnswerRequest{Label: string(rune('A' + index))}
	}
	answers[0].Aliases = []string{"alpha"}

	created, err := service.CreateQuestion(dto.CreateQuestionRequest{
		CreatorUserID: "creator",
		Body:          "10個答えろ",
		Answers:       answers,
		Explanation:   "test",
		GameMode:      "bullet",
		Difficulty:    2,
	})
	if err != nil {
		t.Fatalf("CreateQuestion returned error: %v", err)
	}
	if created != store.created {
		t.Fatal("created question was not persisted")
	}

	var answerData bulletAnswerData
	if err := json.Unmarshal([]byte(created.AnswerData), &answerData); err != nil {
		t.Fatalf("decode answer_data: %v", err)
	}
	if answerData.Question != "10個答えろ" || len(answerData.Answers) != 10 {
		t.Fatalf("unexpected answer_data: %#v", answerData)
	}
	if got := answerData.Aliases["A"]; len(got) != 1 || got[0] != "alpha" {
		t.Fatalf("aliases were not preserved: %#v", got)
	}
}

func TestCreateBulletQuestionRejectsCrossAnswerAliasCollision(t *testing.T) {
	store := &fakeQuestionStore{}
	service := NewQuestionService(store)
	answers := make([]dto.BulletAnswerRequest, 10)
	for index := range answers {
		answers[index] = dto.BulletAnswerRequest{Label: string(rune('A' + index))}
	}
	answers[0] = dto.BulletAnswerRequest{Label: "東京", Aliases: []string{"とうきょう"}}
	answers[1] = dto.BulletAnswerRequest{Label: "大阪", Aliases: []string{"トウキョウ"}}

	_, err := service.CreateQuestion(dto.CreateQuestionRequest{
		CreatorUserID: "creator",
		Body:          "10個答えろ",
		Answers:       answers,
		Explanation:   "test",
		GameMode:      "bullet",
		Difficulty:    2,
	})
	if !errors.Is(err, utils.ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
	if store.created != nil {
		t.Fatal("invalid question must not be persisted")
	}
}

func TestBuildBulletAnswerIndexMapsAliasesToOneCanonicalAnswer(t *testing.T) {
	answerKey, labels, err := buildBulletAnswerIndex(bulletAnswerData{
		Answers: []string{"東京都", "大阪府"},
		Aliases: map[string][]string{
			"東京都": {"東京", "とうきょう"},
		},
	})
	if err != nil {
		t.Fatalf("buildBulletAnswerIndex returned error: %v", err)
	}
	canonical := answerKey[normalizeAnswer("東京都")]
	if canonical == "" || answerKey[normalizeAnswer("東京")] != canonical || answerKey[normalizeAnswer("トウキョウ")] != canonical {
		t.Fatalf("aliases did not resolve to the canonical answer: %#v", answerKey)
	}
	if labels[canonical] != "東京都" {
		t.Fatalf("unexpected canonical label: %#v", labels)
	}
}

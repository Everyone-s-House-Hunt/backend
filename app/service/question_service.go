package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"house-hunt/dto"
	"house-hunt/model"
	"house-hunt/utils"
)

type questionStore interface {
	GetQestions(gameMode string, limit int) ([]model.Question, error)
	CreateQuestion(question *model.Question) error
	UpdateQuestionStatus(id string, status string) error
}

type QuestionService struct {
	repo questionStore
}

func NewQuestionService(repo questionStore) *QuestionService {
	return &QuestionService{repo: repo}
}

func (s *QuestionService) GetQuestions(gameMode string, limit int) ([]model.Question, error) {
	if limit <= 0 {
		limit = 10
	}

	if limit > 100 {
		limit = 100
	}

	return s.repo.GetQestions(gameMode, limit)
}

func (s *QuestionService) CreateQuestion(req dto.CreateQuestionRequest) (*model.Question, error) {
	answerDataBytes, err := buildQuestionAnswerData(req)
	if err != nil {
		return nil, err
	}

	question := &model.Question{
		ID:            utils.GenerateUUID(),
		CreatorUserID: req.CreatorUserID,
		Body:          req.Body,
		AnswerData:    string(answerDataBytes),
		Explanation:   req.Explanation,
		GameMode:      req.GameMode,
		Difficulty:    req.Difficulty,
		Status:        "pending",
	}

	err = s.repo.CreateQuestion(question)
	if err != nil {
		fmt.Println(err)
		return nil, utils.ErrDatabase
	}

	return question, nil
}

func buildQuestionAnswerData(req dto.CreateQuestionRequest) ([]byte, error) {
	if req.GameMode != bulletGameMode {
		if len(req.Choices) < 2 || len(req.Choices) > 4 {
			return nil, fmt.Errorf("%w: choices must contain 2 to 4 items", utils.ErrInvalidInput)
		}
		if req.CorrectIndex == nil || *req.CorrectIndex < 0 || *req.CorrectIndex >= len(req.Choices) {
			return nil, fmt.Errorf("%w: correct_index is out of range", utils.ErrInvalidInput)
		}
		return json.Marshal(map[string]interface{}{
			"choices":       req.Choices,
			"correct_index": *req.CorrectIndex,
		})
	}

	if len(req.Answers) < bulletTargetHits {
		return nil, fmt.Errorf("%w: bullet requires at least %d answers", utils.ErrInvalidInput, bulletTargetHits)
	}

	canonicalAnswers := make([]string, 0, len(req.Answers))
	aliases := make(map[string][]string)
	variantOwner := make(map[string]string)

	for _, answer := range req.Answers {
		label := strings.TrimSpace(answer.Label)
		canonicalKey := normalizeAnswer(label)
		if canonicalKey == "" {
			return nil, fmt.Errorf("%w: answer label must not be empty", utils.ErrInvalidInput)
		}
		if owner, exists := variantOwner[canonicalKey]; exists {
			return nil, fmt.Errorf("%w: answer %q conflicts with %q", utils.ErrInvalidInput, label, owner)
		}
		variantOwner[canonicalKey] = label
		canonicalAnswers = append(canonicalAnswers, label)

		seenForAnswer := map[string]bool{canonicalKey: true}
		for _, rawAlias := range answer.Aliases {
			alias := strings.TrimSpace(rawAlias)
			aliasKey := normalizeAnswer(alias)
			if aliasKey == "" {
				return nil, fmt.Errorf("%w: alias for %q must not be empty", utils.ErrInvalidInput, label)
			}
			if seenForAnswer[aliasKey] {
				return nil, fmt.Errorf("%w: duplicate alias %q for %q", utils.ErrInvalidInput, alias, label)
			}
			if owner, exists := variantOwner[aliasKey]; exists {
				return nil, fmt.Errorf("%w: alias %q for %q conflicts with %q", utils.ErrInvalidInput, alias, label, owner)
			}
			seenForAnswer[aliasKey] = true
			variantOwner[aliasKey] = label
			aliases[label] = append(aliases[label], alias)
		}
	}

	answerData := bulletAnswerData{
		Question: req.Body,
		Answers:  canonicalAnswers,
		Aliases:  aliases,
	}
	return json.Marshal(answerData)
}

func (s *QuestionService) UpdateQuestionStatus(id string, req dto.UpdateQuestionStatusRequest) error {
	err := s.repo.UpdateQuestionStatus(id, req.Status)
	if err != nil {
		if err == utils.ErrNotFoundID {
			return err
		}
		return utils.ErrDatabase
	}
	return nil
}

func IsInvalidQuestion(err error) bool {
	return errors.Is(err, utils.ErrInvalidInput)
}

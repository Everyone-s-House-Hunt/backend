package service

import (
	"house-hunt/model"
	"house-hunt/repository"
	"house-hunt/dto"
	"house-hunt/utils"
	"encoding/json"
)

type QuestionService struct {
	repo *repository.QuestionRepository
}

func NewQuestionService(repo *repository.QuestionRepository) *QuestionService {
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
	answerMap := map[string]interface{}{
		"choices":       req.Choices,
		"correct_index": req.CorrectIndex,
	}
	
	answerDataBytes, err := json.Marshal(answerMap)
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

	if err := s.repo.CreateQuestion(question); err != nil {
		return nil, err
	}

	return question, nil
}
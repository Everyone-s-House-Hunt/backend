package service

import (
	"house-hunt/model"
	"house-hunt/repository"
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
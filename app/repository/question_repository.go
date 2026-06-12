package repository

import (
	"house-hunt/model"
	"gorm.io/gorm"
)

type QuestionRepository struct {
	db *gorm.DB
}

func NewQuestionRepository(db *gorm.DB) *QuestionRepository {
	return &QuestionRepository{db: db}
}

func (r *QuestionRepository) GetQestions(gameMode string, limit int) ([]model.Question, error) {
	var questions []model.Question

	levelLimit := (limit + 4) / 5

	subQuery := r.db.Model(&model.Question{}).Select("*, ROW_NUMBER() OVER (PARTITION BY difficulty ORDER BY RAND()) as rn").Where("game_mode = ? AND status = ?", gameMode, "approved")

	err := r.db.Table("(?) as ranked_questions", subQuery).Select("id", "body", "answer_data", "explanation", "difficulty").Where("rn <= ?", levelLimit).Order("difficulty ASC, rn ASC").Limit(limit).Find(&questions).Error

	if err != nil {
		return nil, err
	}

	return questions, nil
}
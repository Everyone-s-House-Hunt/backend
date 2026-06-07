package repository

import (
	"house-hunt/model"

	"gorm.io/gorm"
)

// questions テーブルへのアクセス担当
type QuestionRepository struct {
	DB *gorm.DB
}

func NewQuestionRepository(db *gorm.DB) *QuestionRepository {
	return &QuestionRepository{DB: db}
}

// 指定モードの承認済み問題をランダムに最大 limit 件取得する
func (r *QuestionRepository) GetRandomByGameMode(gameMode string, limit int) ([]model.Question, error) {
	var questions []model.Question
	err := r.DB.Where("game_mode = ? AND status = ?", gameMode, "approved").
		Order("RAND()"). // 毎回ランダムな順で出題
		Limit(limit).
		Find(&questions).Error
	return questions, err
}

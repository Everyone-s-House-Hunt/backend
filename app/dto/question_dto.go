package dto

import "house-hunt/model"

type QuestionResponse struct {
	ID          string `json:"id"`
	Body        string `json:"body"`
	AnswerData  string `json:"answer_data"`
	Explanation string `json:"explanation"`
	Difficulty  int    `json:"difficulty"`
}

func BuildQuestionResponse(q model.Question) QuestionResponse {
	return QuestionResponse{
		ID:          q.ID,
		Body:        q.Body,
		AnswerData:  q.AnswerData,
		Explanation: q.Explanation,
		Difficulty:  q.Difficulty,
	}
}

type CreateQuestionRequest struct {
	CreatorUserID string                `json:"creator_user_id" binding:"required"`
	Body          string                `json:"body" binding:"required"`
	Choices       []string              `json:"choices"`
	CorrectIndex  *int                  `json:"correct_index"`
	Answers       []BulletAnswerRequest `json:"answers"`
	Explanation   string                `json:"explanation" binding:"required"`
	GameMode      string                `json:"game_mode" binding:"required"`
	Difficulty    int                   `json:"difficulty" binding:"required,min=1,max=5"`
}

// BulletAnswerRequest is one canonical answer and the accepted spellings for it.
// It is used only when game_mode is "bullet".
type BulletAnswerRequest struct {
	Label   string   `json:"label"`
	Aliases []string `json:"aliases"`
}

type UpdateQuestionStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=approved pending"`
}

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
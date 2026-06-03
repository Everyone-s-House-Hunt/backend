package service

import (
	"house-hunt/model"
	"house-hunt/repository"
	"house-hunt/utils"
)

type UserService struct {
	Repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{Repo: repo}
}

func (s *UserService) Register(username, email, password string) (*model.User, error) {

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		ID:       utils.GenerateUUID(),
		Username: username,
		Email:    email,
		PasswordHash: hashedPassword,
		IsPremium: false,
		CreatedAt: utils.GetTimeJST(),
	}
	err = s.Repo.CreateUser(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}
package service

import (
	"errors"
	"house-hunt/dto"
	"house-hunt/model"
	"house-hunt/repository"
	"house-hunt/utils"

	"golang.org/x/crypto/bcrypt"
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
		ID:           utils.GenerateUUID(),
		Username:     username,
		Email:        email,
		PasswordHash: &hashedPassword,
		Provider:     "local",
		IsPremium:    false,
		CreatedAt:    utils.GetTimeJST(),
	}
	err = s.Repo.CreateUser(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) Login(req dto.LoginRequest) (string, string, error) {
	user, err := s.Repo.FindByEmail(req.Email)
	if err != nil {
		return "", "", errors.New("メールアドレスまたはパスワードが間違っています")
	}

	if user.PasswordHash == nil || user.Provider != "local" {
		return "", "", errors.New("このアカウントは外部サービスで登録されています。該当のログイン方法を使用してください")
	}

	err = bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password))
	if err != nil {
		return "", "", errors.New("メールアドレスまたはパスワードが間違っています")
	}

	accessToken, refreshToken, err := utils.GenerateTokenPair(user.ID, "user")
	if err != nil {
		return "", "", errors.New("トークンの生成に失敗しました")
	}

	return accessToken, refreshToken, nil
}

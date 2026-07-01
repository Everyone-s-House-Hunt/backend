package service

import (
	"context"
	"errors"
	"house-hunt/dto"
	"house-hunt/model"
	"house-hunt/repository"
	"house-hunt/utils"
	"os"

	"google.golang.org/api/idtoken"
	"gorm.io/gorm"

	"github.com/go-sql-driver/mysql"
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
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return nil, utils.ErrDuplicateEmail
		}
		return nil, utils.ErrDatabase
	}
	return user, nil
}

func (s *UserService) Login(req dto.LoginRequest) (string, string, error) {
	var mysqlErr *mysql.MySQLError

	user, err := s.Repo.FindByEmail(req.Email)
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return "", "", utils.ErrUserNotFound
	} else if err != nil {
		return "", "", utils.ErrDatabase
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

func (s *UserService) LoginWithGoogle(ctx context.Context, idTokenStr string) (string, string, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		return "", "", utils.ErrInternalServer
	}

	payload, err := idtoken.Validate(ctx, idTokenStr, clientID)
	if err != nil {
		return "", "", utils.ErrInvalidGoogleToken
	}

	email, ok := payload.Claims["email"].(string)
	if !ok || email == "" {
		return "", "", utils.ErrInvalidGoogleToken
	}

	username, ok := payload.Claims["name"].(string)
	if !ok || username == "" {
		username = "User_" + utils.GenerateUUID()[:8]
	}

	user, err := s.Repo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, utils.ErrUserNotFound) || errors.Is(err, gorm.ErrRecordNotFound) {
			user = &model.User{
				ID:           utils.GenerateUUID(),
				Username:     username,
				Email:        email,
				PasswordHash: nil,
				Provider:     "google",
				IsPremium:    false,
				CreatedAt:    utils.GetTimeJST(),
			}
			if createErr := s.Repo.CreateUser(user); createErr != nil {
				return "", "", utils.ErrInternalServer
			}
		} else {
			return "", "", utils.ErrDatabase
		}
	} else {
		if user.Provider != "google" {
			return "", "", utils.ErrProviderConflict
		}
	}

	accessToken, refreshToken, err := utils.GenerateTokenPair(user.ID, "user")
	if err != nil {
		return "", "", utils.ErrInternalServer
	}

	return accessToken, refreshToken, nil
}

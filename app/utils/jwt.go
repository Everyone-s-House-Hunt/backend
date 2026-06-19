package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func getJWTSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("システム構成エラー: JWT_SECRETが環境変数に設定されていません")
	}
	return []byte(secret), nil
}

func GenerateTokenPair(userID, role string) (string, string, error) {
	secretKey, err := getJWTSecret()
	if err != nil {
		return "", "", err
	}

	accessTokenClaims := TokenClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 2)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims)
	accessTokenString, err := accessToken.SignedString(secretKey)
	if err != nil {
		return "", "", errors.New("アクセストークンの署名に失敗しました")
	}

	refreshTokenClaims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 30)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshTokenClaims)
	refreshTokenString, err := refreshToken.SignedString(secretKey)
	if err != nil {
		return "", "", errors.New("リフレッシュトークンの署名に失敗しました")
	}

	return accessTokenString, refreshTokenString, nil
}

func VerifyToken(tokenString string) (*TokenClaims, error) {
	secretKey, err := getJWTSecret()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("予期しない署名方式です")
		}
		return secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("無効なトークンです")
}
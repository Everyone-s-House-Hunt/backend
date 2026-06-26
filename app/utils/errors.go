package utils

import "errors"

// 認証・ユーザー関連
var (
	ErrDuplicateEmail     = errors.New("このメールアドレスは既に登録されています")
	ErrInvalidCredentials = errors.New("メールアドレスまたはパスワードが間違っています")
	ErrUserNotFound       = errors.New("指定されたユーザーが見つかりません")
	ErrExternalProvider   = errors.New("このアカウントは外部サービスで登録されています。該当のログイン方法を使用してください")
	ErrUnauthorized       = errors.New("認証に失敗しました。再度ログインしてください")
	ErrTokenExpired       = errors.New("セッションの有効期限が切れています。再度ログインしてください")
	ErrInvalidToken       = errors.New("無効な認証トークンです")
)

// リクエスト・バリデーション関連
var (
	ErrInvalidInput    = errors.New("入力内容に誤りがあります")
	ErrMissingGameMode = errors.New("ゲームモードが指定されていません")
	ErrInvalidGameMode = errors.New("指定されたゲームモードは無効です")
	ErrNotFoundID      = errors.New("データベースにIDが見つかりません")
)

// リソース関連問題・ゲームルーム
var (
	ErrQuestionNotFound = errors.New("指定された問題が見つかりません")
	ErrRoomNotFound     = errors.New("指定されたルームが見つかりません")
	ErrRoomIsFull       = errors.New("指定されたルームは満員です")
)

// システム・インフラ関連
var (
	ErrInternalServer = errors.New("システムエラーが発生しました。しばらく経ってから再度お試しください")
	ErrDatabase       = errors.New("データベース処理中にエラーが発生しました")
)
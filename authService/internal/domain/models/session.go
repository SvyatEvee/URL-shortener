package models

type Session struct {
	ID                         int64
	UserID                     int64
	RefreshTokenRandomPartHash []byte
	CreatedAt                  int64
	ExpiresAt                  int64
}

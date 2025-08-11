package models

type Session struct {
	ID               int64
	UserID           int64
	RefreshTokenHash []byte
	CreatedAt        int64
	ExpiresAt        int64
}

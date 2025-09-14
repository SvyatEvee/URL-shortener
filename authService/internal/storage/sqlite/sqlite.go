package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"
	"sso/internal/domain/models"
	"sso/internal/storage"
)

type Storage struct {
	db *sql.DB
}

// New creates a new instance of the SQLite storage
func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &Storage{db: db}, nil
}

func (s *Storage) SaveUser(ctx context.Context, email string, passHash []byte) (int64, error) {
	const op = "storage.sqlite.SaveUser"

	stmt, err := s.db.Prepare("INSERT INTO users(email, password_hash, role_id) VALUES(?, ?, 1)")
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, email, passHash)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return 0, fmt.Errorf("%s: timeout reached: %w", op, err)
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get last insert id: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetUser(ctx context.Context, email string) (models.User, error) {
	const op = "storage.sqlite.User"

	stmt, err := s.db.Prepare("SELECT id, password_hash FROM users WHERE email = ?")
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var id int64
	var pass_hash []byte
	err = stmt.QueryRowContext(ctx, email).Scan(&id, &pass_hash)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return models.User{}, storage.ErrUserNotFound
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return models.User{}, fmt.Errorf("%s: timeout reached: %w", op, err)
	case err != nil:
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return models.User{
		ID:       id,
		Email:    email,
		PassHash: pass_hash,
	}, nil
}

//func (s *Storage) IsAdmin(ctx context.Context, userID int64) (bool, error) {
//	const op = "storage.sqlite.IsAdmin"
//
//	stmt, err := s.db.Prepare("SELECT is_admin FROM users WHERE id = ?")
//	if err != nil {
//		return false, fmt.Errorf("%s: %w", op, err)
//	}
//	defer stmt.Close()
//
//	var isAdmin bool
//	err = stmt.QueryRowContext(ctx, userID).Scan(&isAdmin)
//	switch {
//	case errors.Is(err, sql.ErrNoRows):
//		return false, storage.ErrUserNotFound
//	case errors.Is(ctx.Err(), context.DeadlineExceeded):
//		return false, fmt.Errorf("%s: timeout reached: %w", op, err)
//	case err != nil:
//		return false, fmt.Errorf("%s: %w", op, err)
//	}
//
//	return isAdmin, nil
//}

//func (s *Storage) App(ctx context.Context, appID int) (models.App, error) {
//	const op = "storage.sqlite.App"
//
//	stmt, err := s.db.Prepare("SELECT name, secret FROM apps WHERE id = ?")
//	if err != nil {
//		return models.App{}, fmt.Errorf("%s: %w", op, err)
//	}
//	defer stmt.Close()
//
//	app := models.App{ID: appID}
//	err = stmt.QueryRowContext(ctx, appID).Scan(&app.Name, &app.Secret)
//	switch {
//	case errors.Is(err, sql.ErrNoRows):
//		return models.App{}, storage.ErrAppNotFound
//	case errors.Is(ctx.Err(), context.DeadlineExceeded):
//		return models.App{}, fmt.Errorf("%s: timeout reached: %w", op, err)
//	case err != nil:
//		return models.App{}, fmt.Errorf("%s: %w", op, err)
//	}
//
//	return app, nil
//}

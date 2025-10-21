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
			return 0, fmt.Errorf("%s: %w (%v)", op, storage.ErrUserExists, err)
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

func (s *Storage) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	const op = "storage.sqlite.GetUserByEmail"

	stmt, err := s.db.Prepare(`
				SELECT u.id, u.password_hash, r.name
				FROM users u
				LEFT JOIN roles r ON u.role_id = r.id
				WHERE u.email = ?
				`)
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	user := models.User{
		Email: email,
	}
	err = stmt.QueryRowContext(ctx, email).Scan(&user.ID, &user.PassHash, &user.Role)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return models.User{}, fmt.Errorf("%s: %w (%v)", op, storage.ErrUserNotFound, err)
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			return models.User{}, fmt.Errorf("%s: timeout reached: %w", op, err)
		default:
			return models.User{}, fmt.Errorf("%s: %w", op, err)
		}
	}

	return user, nil
}

func (s *Storage) GetUserByID(ctx context.Context, userID int64) (models.User, error) {
	const op = "storage.sqlite.GetUserByID"

	stmt, err := s.db.Prepare(`
				SELECT u.email, u.password_hash, r.name
				FROM users u
				LEFT JOIN roles r ON u.role_id = r.id
				WHERE u.id = ?
				`)
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	user := models.User{
		ID: userID,
	}
	err = stmt.QueryRowContext(ctx, userID).Scan(&user.Email, &user.PassHash, &user.Role)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return models.User{}, fmt.Errorf("%s: %w (%v)", op, storage.ErrUserNotFound, err)
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			return models.User{}, fmt.Errorf("%s: timeout reached: %w", op, err)
		default:
			return models.User{}, fmt.Errorf("%s: %w", op, err)
		}
	}

	return user, nil
}

// TODO: test me?
func (s *Storage) SaveSession(ctx context.Context, session models.Session) (int64, error) {
	const op = "storage.sqlite.SaveSession"

	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("%s: context error: %w", op, err)
	}

	res, err := s.db.ExecContext(ctx, "INSERT INTO sessions(user_id, refresh_token_randnom_part_hash, created_at, expires_at) VALUES(?, ?, ?, ?)",
		session.UserID, session.RefreshTokenRandomPartHash, session.CreatedAt, session.ExpiresAt)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return 0, fmt.Errorf("%s: timeout reached: %w", op, err)
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return 0, fmt.Errorf("%s: context canceled: %w", op, err)
		}

		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) {
			err = fmt.Errorf("sqliteError: %w", err)
			switch sqliteErr.ExtendedCode {
			case sqlite3.ErrConstraintUnique:
				return 0, fmt.Errorf("%s: %w (%v)", op, storage.ErrSessionExists, err)
			default:
				return 0, fmt.Errorf("%s: %w", op, err)
			}
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get last insert id: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetSession(ctx context.Context, sessionID int64) (*models.Session, error) {
	const op = "storage.sqlite.GetSession"

	if err := ctx.Err(); err != nil {
		return &models.Session{}, fmt.Errorf("%s: context error: %w", op, err)
	}

	session := &models.Session{
		ID: sessionID,
	}

	query := "SELECT user_id, refresh_token_randnom_part_hash, created_at, expires_at FROM sessions WHERE id = ?"

	err := s.db.QueryRowContext(ctx, query, sessionID).
		Scan(&session.UserID, &session.RefreshTokenRandomPartHash, &session.CreatedAt, &session.ExpiresAt)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return &models.Session{}, fmt.Errorf("%s: %w (%v)", op, storage.ErrSessionNotFound, err)
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			return &models.Session{}, fmt.Errorf("%s: timeout reached: %w", op, err)
		default:
			return &models.Session{}, fmt.Errorf("%s: %w", op, err)
		}
	}

	return session, nil
}

func (s *Storage) UpdateSession(ctx context.Context, newSession *models.Session) error {
	const op = "storage.sqlite.UpdateSession"

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%s: context error: %w", op, err)
	}

	query := `UPDATE sessions
			  SET refresh_token_randnom_part_hash = ?, created_at = ?, expires_at = ?
			  WHERE id = ?`

	res, err := s.db.ExecContext(ctx, query,
		newSession.RefreshTokenRandomPartHash,
		newSession.CreatedAt,
		newSession.ExpiresAt,
		newSession.ID)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("%s: timeout reached: %w", op, err)
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return fmt.Errorf("%s: context canceled: %w", op, err)
		}

		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrSessionNotFound)
	}
	if rowsAffected > 1 {
		return fmt.Errorf("%s: rowsAffected is greater than 1", op)
	}
	return nil
}

func (s *Storage) DeleteSession(ctx context.Context, sessionID int64) error {
	const op = "storage.sqlite.DeleteSession"

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%s: context error: %w", op, err)
	}

	query := "DELETE FROM sessions WHERE id = ?"

	res, err := s.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("%s: timeout reached: %w", op, err)
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return fmt.Errorf("%s: context canceled: %w", op, err)
		}

		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	switch {
	case rowsAffected == 0:
		return storage.ErrSessionNotFound
	case rowsAffected > 1:
		return fmt.Errorf("%s: rowsAffected is greater than 1", op)
	}

	return nil
}

func (s *Storage) DeleteUser(ctx context.Context, userID int64) error {
	const op = "storage.sqlite.DeleteUser"

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%s: context error: %w", op, err)
	}

	query := "DELETE FROM users WHERE id = ?"

	res, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("%s: timeout reached: %w", op, err)
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return fmt.Errorf("%s: context canceled: %w", op, err)
		}

		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	switch {
	case rowsAffected == 0:
		return storage.ErrUserNotFound
	case rowsAffected > 1:
		return fmt.Errorf("%s: rowsAffected is greater than 1", op)
	}

	return nil
}

func (s *Storage) DeleteAllUserSessions(ctx context.Context, userID int64) (int64, error) {
	const op = "storage.sqlite.DeleteAllUserSessions"

	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("%s: context error: %w", op, err)
	}

	query := "DELETE FROM sessions WHERE user_id = ?"

	res, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return 0, fmt.Errorf("%s: timeout reached: %w", op, err)
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return 0, fmt.Errorf("%s: context canceled: %w", op, err)
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return rowsAffected, nil
}

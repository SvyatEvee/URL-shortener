package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"sso/internal/domain/models"
	"sso/internal/storage"
	"strings"
)

type Storage struct {
	db     *sql.DB
	driver string
}

// New creates a new instance of the SQLite storage
func New(driver string, connStr string) (*Storage, error) {
	const op = "storage.sql.New"

	db, err := sql.Open(driver, connStr)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if driver == "sqlite3" {
		_, err = db.Exec("PRAGMA foreign_keys = ON;")
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
	}
	return &Storage{
		db:     db,
		driver: driver,
	}, nil
}

func (s *Storage) ConvertQuery(query string) string {
	switch s.driver {
	case "postgres":
		var builder strings.Builder
		counter := 1
		for _, ch := range query {

			if ch == '?' {
				builder.WriteString(fmt.Sprintf("$%d", counter))
				counter++
			} else {
				builder.WriteRune(ch)
			}
		}
		return builder.String()
	default:
		return query
	}
}

func (s *Storage) SaveUser(ctx context.Context, email string, passHash []byte) (int64, error) {
	const op = "storage.sql.SaveUser"

	query := s.ConvertQuery(`INSERT INTO users(email, password_hash, role_id) VALUES(?, ?, 1) RETURNING id`)

	var lastInsertID int64
	err := s.db.QueryRowContext(ctx, query, email, passHash).Scan(&lastInsertID)
	if err != nil {

		if storage.IsConstraintUnique(err) {
			return 0, fmt.Errorf("%s: %w (%v)", op, storage.ErrUserExists, err)
		}

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return 0, fmt.Errorf("%s: timeout reached: %w", op, err)
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return lastInsertID, nil
}

func (s *Storage) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	const op = "storage.sql.GetUserByEmail"

	query := s.ConvertQuery(`
				SELECT u.id, u.password_hash, r.name
				FROM users u
				LEFT JOIN roles r ON u.role_id = r.id
				WHERE u.email = ?
				`)

	user := models.User{
		Email: email,
	}
	err := s.db.QueryRowContext(ctx, query, email).Scan(&user.ID, &user.PassHash, &user.Role)
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
	const op = "storage.sql.GetUserByID"

	query := s.ConvertQuery(`
				SELECT u.email, u.password_hash, r.name
				FROM users u
				LEFT JOIN roles r ON u.role_id = r.id
				WHERE u.id = ?
				`)

	user := models.User{
		ID: userID,
	}
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&user.Email, &user.PassHash, &user.Role)
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
	const op = "storage.sql.SaveSession"

	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("%s: context error: %w", op, err)
	}

	query := s.ConvertQuery(`INSERT INTO sessions(user_id, refresh_token_randnom_part_hash, created_at, expires_at) VALUES(?, ?, ?, ?) RETURNING id`)

	var lastInsertID int64
	err := s.db.QueryRowContext(ctx, query,
		session.UserID,
		session.RefreshTokenRandomPartHash,
		session.CreatedAt,
		session.ExpiresAt,
	).Scan(&lastInsertID)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return 0, fmt.Errorf("%s: timeout reached: %w", op, err)
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return 0, fmt.Errorf("%s: context canceled: %w", op, err)
		}

		if storage.IsConstraintUnique(err) {
			return 0, fmt.Errorf("%s: %w (%v)", op, storage.ErrSessionExists, err)
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return lastInsertID, nil
}

func (s *Storage) GetSession(ctx context.Context, sessionID int64) (*models.Session, error) {
	const op = "storage.sql.GetSession"

	if err := ctx.Err(); err != nil {
		return &models.Session{}, fmt.Errorf("%s: context error: %w", op, err)
	}

	session := &models.Session{
		ID: sessionID,
	}

	query := s.ConvertQuery(`SELECT user_id, refresh_token_randnom_part_hash, created_at, expires_at FROM sessions WHERE id = ?`)

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
	const op = "storage.sql.UpdateSession"

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%s: context error: %w", op, err)
	}

	query := s.ConvertQuery(`UPDATE sessions
			  SET refresh_token_randnom_part_hash = ?, created_at = ?, expires_at = ?
			  WHERE id = ?`)

	res, err := s.db.ExecContext(ctx, query,
		newSession.RefreshTokenRandomPartHash,
		newSession.CreatedAt,
		newSession.ExpiresAt,
		newSession.ID,
	)
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
	const op = "storage.sql.DeleteSession"

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%s: context error: %w", op, err)
	}

	query := s.ConvertQuery(`DELETE FROM sessions WHERE id = ?`)

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
	const op = "storage.sql.DeleteUser"

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%s: context error: %w", op, err)
	}

	query := s.ConvertQuery(`DELETE FROM users WHERE id = ?`)

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
	const op = "storage.sql.DeleteAllUserSessions"

	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("%s: context error: %w", op, err)
	}

	query := s.ConvertQuery(`DELETE FROM sessions WHERE user_id = ?`)

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

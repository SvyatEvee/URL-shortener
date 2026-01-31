package sql

import (
	"URLshortener/internal/domain/models"
	"URLshortener/internal/storage"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
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

func (s *Storage) SaveURL(urlToSave string, alias string, userID int64) (int64, error) {
	const op = "storage.sql.SaveURL"

	query := s.ConvertQuery(`INSERT INTO url(url, alias, user_id)  VALUES(?, ?, ?) RETURNING id;`)

	var lastInsertID int64
	err := s.db.QueryRow(query, urlToSave, alias, userID).Scan(&lastInsertID)
	if err != nil {

		if storage.IsConstraintUnique(err) {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrAliasExist)
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return lastInsertID, nil
}

func (s *Storage) GetUserUrls(userID int64) ([]models.AliasNote, error) {
	const op = "storage.sql.GetUserUrls"

	query := s.ConvertQuery(`SELECT id, url, alias FROM url WHERE user_id = ?`)

	rows, err := s.db.Query(query, userID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, storage.ErrAliasNotFound
	case err != nil:
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	aliasNotes := []models.AliasNote{}
	for rows.Next() {
		var aliasNote models.AliasNote
		err = rows.Scan(&aliasNote.ID, &aliasNote.Url, &aliasNote.Alias)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		aliasNotes = append(aliasNotes, aliasNote)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return aliasNotes, nil
}

func (s *Storage) GetURL(alias string, userID int64) (string, error) {
	const op = "storage.sql.GetURL"

	query := s.ConvertQuery(`SELECT url FROM url WHERE alias = ? AND user_id = ?`)

	var resURL string
	err := s.db.QueryRow(query, alias, userID).Scan(&resURL)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", storage.ErrAliasNotFound
	case err != nil:
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return resURL, nil
}

func (s *Storage) DeleteAlias(id int64, userID int64) error {
	const op = "storage.sql.DeleteURL"

	query := s.ConvertQuery(`DELETE FROM url WHERE id = ? AND user_id = ?`)

	result, err := s.db.Exec(query, id, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrAliasNotFound)
	}

	return nil
}

func (s *Storage) UpdateAlias(id int64, newUrl string, userID int64) error {
	const op = "storage.sql.UpdateURL"

	query := s.ConvertQuery(`UPDATE url SET url = ? WHERE id = ? AND user_id = ?`)

	result, err := s.db.Exec(query, newUrl, id, userID)
	if err != nil {
		if storage.IsConstraintUnique(err) {
			return fmt.Errorf("%s: %w", op, storage.ErrAliasExist)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrAliasNotFound)
	}

	return nil
}

func (s *Storage) UpdateAliasURL(newURL string, alias string, userID int64) error {
	const op = "storage.sql.UpdateURL"

	query := s.ConvertQuery(`UPDATE url SET url = ? WHERE alias = ? AND user_id = ?`)

	result, err := s.db.Exec(query, newURL, alias, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrAliasNotFound)
	}
	return nil
}

func (s *Storage) DeleteUserData(userID int64) error {
	const op = "storage.sql.DeleteUserData"

	query := s.ConvertQuery(`DELETE FROM url WHERE user_id = ?`)

	_, err := s.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

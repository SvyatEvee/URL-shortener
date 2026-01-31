package storage

import (
	"errors"
	"github.com/lib/pq"
	"github.com/mattn/go-sqlite3"
)

var (
	ErrAliasNotFound = errors.New("alias not found")
	ErrURLNotFound   = errors.New("url not found")
	ErrAliasExist    = errors.New("alias exist")
)

func IsConstraintUnique(err error) bool {
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
		return true
	}

	var pgErr *pq.Error
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}

	return false
}

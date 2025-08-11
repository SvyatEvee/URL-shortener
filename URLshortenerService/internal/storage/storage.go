package storage

import "errors"

var (
	ErrAliasNotFound = errors.New("alias not found")
	ErrURLNotFound   = errors.New("url not found")
	ErrAliasExist    = errors.New("alias exist")
)

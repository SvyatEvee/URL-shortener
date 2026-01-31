package models

type AliasNote struct {
	ID    int64  `json:"id"`
	Url   string `json:"url"`
	Alias string `json:"alias"`
}

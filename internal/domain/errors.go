package domain

import "errors"

var (
	ErrNotFound  = errors.New("link not found")
	ErrCodeTaken = errors.New("short code already exists")
)

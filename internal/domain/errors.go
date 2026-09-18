package domain

import "errors"

var (
	ErrNotFound          = errors.New("не найдено")
	ErrValidation        = errors.New("некорректные данные")
	ErrConflict          = errors.New("конфликт состояния")
	ErrSourceUnavailable = errors.New("источник тестов недоступен")
	ErrUnauthorized      = errors.New("неверный секрет")
	ErrFormClosed        = errors.New("форма закрыта")
)

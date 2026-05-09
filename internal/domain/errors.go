package domain

import "errors"

var (
	ErrNotFound      = errors.New("entidade não encontrada")
	ErrDuplicateName = errors.New("nome já existe")
	ErrWatchInUse    = errors.New("pasta monitorada está em uso por regras") // decisão D-j
	ErrInvalidInput  = errors.New("entrada inválida")
	ErrNoText        = errors.New("pdf não contém texto extraível")
)

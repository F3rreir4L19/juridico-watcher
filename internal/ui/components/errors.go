package components

import (
	"errors"
	"strings"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
)

// HumanizeError traduz erros conhecidos em mensagens amigáveis em português.
// Para erros desconhecidos, devolve a mensagem original.
func HumanizeError(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return "Item não encontrado. Pode ter sido removido em outra janela."
	case errors.Is(err, domain.ErrDuplicateName):
		return "Já existe um item com esse nome. Escolha um nome diferente."
	case errors.Is(err, domain.ErrWatchInUse):
		return "Esta pasta está sendo usada por uma ou mais regras. Edite ou remova as regras antes de removê-la."
	case errors.Is(err, domain.ErrInvalidInput):
		return "Os dados informados estão incompletos ou inválidos."
	}
	// Erro desconhecido: mostra mensagem do Go limpa
	msg := err.Error()
	if idx := strings.Index(msg, ": "); idx > 0 {
		// pega só a parte mais recente (após último wrap)
		msg = msg[strings.LastIndex(msg, ": ")+2:]
	}
	return "Erro: " + msg
}

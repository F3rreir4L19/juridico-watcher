package ui

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/service"
	uic "github.com/F3rreir4L19/juridico-watcher/internal/ui/components"
)

// showWatchDialog abre o diálogo de criar/editar pasta. Se existing for nil,
// é modo "adicionar"; senão, modo "editar". Ao salvar com sucesso, chama onSaved.
func showWatchDialog(
	parent fyne.Window,
	watchService *service.WatchService,
	existing *domain.Watch,
	onSaved func(),
) {
	isNew := existing == nil
	title := "Adicionar pasta monitorada"
	if !isNew {
		title = "Editar pasta monitorada"
	}

	// --- Campos ---
	nameEntry := widget.NewEntry()
	nameEntry.PlaceHolder = "Ex: Digitalizadora"

	pathEntry := widget.NewEntry()
	pathEntry.PlaceHolder = "Selecione clicando em 'Procurar...'"

	browseBtn := widget.NewButton("Procurar...", func() {
		fd := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			pathEntry.SetText(uri.Path())
		}, parent)
		// Inicia no caminho atual se já houver um válido
		if pathEntry.Text != "" {
			if u, err := storage.ListerForURI(storage.NewFileURI(pathEntry.Text)); err == nil {
				fd.SetLocation(u)
			}
		}
		fd.Show()
	})

	pathRow := container.NewBorder(nil, nil, nil, browseBtn, pathEntry)

	recursiveCheck := widget.NewCheck("Monitorar subpastas (recomendado)", nil)
	recursiveCheck.Checked = true

	activeCheck := widget.NewCheck("Pasta ativa (monitoramento ligado)", nil)
	activeCheck.Checked = true

	if !isNew {
		nameEntry.SetText(existing.Name)
		pathEntry.SetText(existing.Path)
		recursiveCheck.Checked = existing.Recursive
		activeCheck.Checked = existing.Active
		recursiveCheck.Refresh()
		activeCheck.Refresh()
	}

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Nome", Widget: nameEntry, HintText: "Como você quer chamar essa pasta"},
			{Text: "Caminho", Widget: pathRow, HintText: "Pasta no seu computador a ser monitorada"},
			{Text: "", Widget: recursiveCheck},
			{Text: "", Widget: activeCheck},
		},
	}

	content := container.NewVBox(form)

	dlg := dialog.NewCustomConfirm(title, "Salvar", "Cancelar", content,
		func(save bool) {
			if !save {
				return
			}
			if err := saveWatch(isNew, existing, nameEntry, pathEntry, recursiveCheck, activeCheck, watchService); err != nil {
				dialog.ShowError(fmt.Errorf("%s", uic.HumanizeError(err)), parent)
				return
			}
			if onSaved != nil {
				onSaved()
			}
		}, parent,
	)
	dlg.Resize(fyne.NewSize(540, 280))
	dlg.Show()
}

// saveWatch valida os campos e cria/atualiza o Watch via service.
// Retorna erro com mensagens contextuais.
func saveWatch(
	isNew bool,
	existing *domain.Watch,
	nameEntry, pathEntry *widget.Entry,
	recursiveCheck, activeCheck *widget.Check,
	svc *service.WatchService,
) error {
	name := strings.TrimSpace(nameEntry.Text)
	path := strings.TrimSpace(pathEntry.Text)

	if name == "" {
		return errors.New("Por favor, preencha o nome da pasta")
	}
	if path == "" {
		return errors.New("Por favor, selecione o caminho da pasta")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("caminho inválido: %s", path)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("a pasta %s não existe no seu computador", absPath)
		}
		return fmt.Errorf("não foi possível acessar %s", absPath)
	}
	if !info.IsDir() {
		return fmt.Errorf("o caminho %s não é uma pasta", absPath)
	}

	w := &domain.Watch{
		Name:      name,
		Path:      absPath,
		Recursive: recursiveCheck.Checked,
		Active:    activeCheck.Checked,
	}
	if isNew {
		return svc.Create(w)
	}
	w.ID = existing.ID
	return svc.Update(w)
}

// avoid lint warnings for url import (kept for future use of validating remote paths)
var _ = url.Parse

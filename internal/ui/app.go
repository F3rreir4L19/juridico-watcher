package ui

import (
	"database/sql"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
)

// App representa a janela principal da aplicação.
type App struct {
	fyneApp fyne.App
	window  fyne.Window
	db      *sql.DB
}

// NewApp cria a aplicação Fyne e configura as abas.
func NewApp(db *sql.DB) *App {
	a := &App{
		fyneApp: app.New(),
		db:      db,
	}
	a.window = a.fyneApp.NewWindow("Juridico Watcher")
	a.window.Resize(fyne.NewSize(800, 600))

	tabs := container.NewAppTabs(
		container.NewTabItem("Monitorar", newTabWatches(db)),
		container.NewTabItem("Regras", newTabRules(db)),
	)

	a.window.SetContent(tabs)
	return a
}

// Run inicia o loop principal da UI.
func (a *App) Run() {
	a.window.ShowAndRun()
}
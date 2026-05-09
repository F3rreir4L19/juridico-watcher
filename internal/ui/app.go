package ui

import (
	"database/sql"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"

	"github.com/F3rreir4L19/juridico-watcher/internal/service"
	"github.com/F3rreir4L19/juridico-watcher/internal/storage"
)

// App representa a janela principal da aplicação.
type App struct {
	fyneApp fyne.App
	window  fyne.Window
	db      *sql.DB
	logger  *slog.Logger

	watchSvc *service.WatchService
	ruleSvc  *service.RuleService
	monitor  *service.MonitorService
}

// NewApp cria a aplicação Fyne, abre a janela e prepara as abas.
// Não inicia o monitor service ainda — chame Run() para isso.
func NewApp(db *sql.DB, logger *slog.Logger) *App {
	if logger == nil {
		logger = slog.Default()
	}
	a := &App{
		fyneApp: app.NewWithID("com.juridicowatcher.app"),
		db:      db,
		logger:  logger,
	}

	wrepo := storage.NewWatchRepo(db)
	rrepo := storage.NewRuleRepo(db)
	a.watchSvc = service.NewWatchService(wrepo, rrepo)
	a.ruleSvc = service.NewRuleService(rrepo)
	a.monitor = service.NewMonitorService(db, logger)

	a.window = a.fyneApp.NewWindow("Juridico Watcher")
	a.window.Resize(fyne.NewSize(1024, 720))
	a.window.SetMaster()

	// Aba Monitorar — reinicia o monitor service quando watches mudam (D-09)
	watchesTab := newWatchesTab(a.window, a.watchSvc, a.restartMonitor)

	// Aba Regras — também reinicia o monitor quando regras mudam (D-09)
	rulesTab := newRulesTab(a.window, a.ruleSvc, a.watchSvc, a.restartMonitor)

	tabs := container.NewAppTabs(
		container.NewTabItem("Monitorar", watchesTab.Root()),
		container.NewTabItem("Regras", rulesTab.Root()),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	a.window.SetContent(tabs)

	// Garante que o monitor é parado ao fechar a janela
	a.window.SetCloseIntercept(func() {
		a.monitor.StopAll()
		a.window.Close()
	})

	return a
}

// Run inicia o monitoramento e abre a janela. Bloqueia até o usuário fechar.
func (a *App) Run() {
	if err := a.monitor.StartMonitoring(); err != nil {
		a.logger.Error("falha ao iniciar monitoramento", "err", err)
	}
	a.window.ShowAndRun()
}

// restartMonitor é chamado pelas abas após qualquer mudança em pastas
// ou regras, para garantir que o serviço de monitoramento reflete o
// estado atual do banco.
func (a *App) restartMonitor() {
	a.monitor.StopAll()
	if err := a.monitor.StartMonitoring(); err != nil {
		a.logger.Error("falha ao reiniciar monitoramento", "err", err)
	}
}

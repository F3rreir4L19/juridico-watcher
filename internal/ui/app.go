package ui

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/F3rreir4L19/juridico-watcher/internal/service"
	"github.com/F3rreir4L19/juridico-watcher/internal/storage"
	uic "github.com/F3rreir4L19/juridico-watcher/internal/ui/components"
)

const (
	historyTabBaseTitle = "Histórico"
	failureBadgeRefresh = 5 * time.Second // intervalo do refresh automático do badge
)

// App representa a janela principal da aplicação.
type App struct {
	fyneApp fyne.App
	window  fyne.Window
	db      *sql.DB
	logger  *slog.Logger

	watchSvc   *service.WatchService
	ruleSvc    *service.RuleService
	scanSvc    *service.ScanService
	historySvc *service.HistoryService
	monitor    *service.MonitorService

	// Refs guardadas para atualizar título da aba dinamicamente
	tabs        *container.AppTabs
	historyTab  *historyTab
	historyItem *container.TabItem

	// Sinaliza pra goroutine de refresh do badge parar quando a janela fecha
	doneCh chan struct{}
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
		doneCh:  make(chan struct{}),
	}

	wrepo := storage.NewWatchRepo(db)
	rrepo := storage.NewRuleRepo(db)
	a.watchSvc = service.NewWatchService(wrepo, rrepo)
	a.ruleSvc = service.NewRuleService(rrepo)
	a.scanSvc = service.NewScanService(db, logger)
	a.historySvc = service.NewHistoryService(db)
	a.monitor = service.NewMonitorService(db, logger)

	a.window = a.fyneApp.NewWindow("Juridico Watcher")
	a.window.Resize(fyne.NewSize(1024, 720))
	a.window.SetMaster()

	// --- Abas ---
	watchesTab := newWatchesTab(a.window, a.watchSvc, a.restartMonitor)
	a.historyTab = newHistoryTab(a.window, a.historySvc, a.ruleSvc)
	rulesTab := newRulesTab(
		a.window,
		a.ruleSvc,
		a.watchSvc,
		a.scanSvc,
		a.restartMonitor,
		func(_ int) { a.historyTab.Reload(); a.refreshHistoryBadge() },
	)

	a.historyItem = container.NewTabItem(historyTabBaseTitle, a.historyTab.Root())
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Monitorar", watchesTab.Root()),
		container.NewTabItem("Regras", rulesTab.Root()),
		a.historyItem,
	)
	a.tabs.SetTabLocation(container.TabLocationTop)

	// Marca visita ao sair da aba Histórico (não ao entrar — D-15.1):
	// dá tempo do usuário olhar as falhas com calma. Só ao trocar pra outra
	// aba é que consideramos "ele já viu".
	a.tabs.OnChanged = func(tab *container.TabItem) {
		if tab.Text == historyTabBaseTitle ||
			tab.Text == a.historyItem.Text { // pode estar com o badge concatenado
			// Acabou de entrar na aba Histórico — ainda não marcamos como visto.
			// Recarrega pra garantir que o usuário vê o estado mais recente.
			a.historyTab.Reload()
			return
		}
		// Saiu da aba Histórico para outra: marca como visto e zera badge.
		a.historyTab.MarkVisited()
		a.refreshHistoryBadge()
	}

	// --- Barra de topo ---
	topBar := a.buildTopBar()

	// --- Layout final ---
	root := container.NewBorder(
		topBar,
		nil, nil, nil,
		a.tabs,
	)
	a.window.SetContent(root)

	a.window.SetCloseIntercept(func() {
		close(a.doneCh)
		a.monitor.StopAll()
		a.window.Close()
	})

	return a
}

// buildTopBar constrói a barra horizontal acima das abas: título do app à
// esquerda e botão "Atualizar tudo" à direita.
func (a *App) buildTopBar() fyne.CanvasObject {
	title := widget.NewLabelWithStyle(
		"Juridico Watcher",
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	scanAllBtn := widget.NewButtonWithIcon(
		"Atualizar tudo",
		theme.ViewRefreshIcon(),
		func() { a.scanAll() },
	)
	scanAllBtn.Importance = widget.HighImportance

	bar := container.New(layout.NewHBoxLayout(),
		title,
		layout.NewSpacer(),
		scanAllBtn,
	)
	return container.NewVBox(
		container.NewPadded(bar),
		widget.NewSeparator(),
	)
}

// scanAll executa ScanAll do scan service em goroutine, com progress dialog
// e mensagem de resultado. Bloqueia interação com a janela enquanto roda.
func (a *App) scanAll() {
	progress := dialog.NewCustomWithoutButtons(
		"Atualizando todas as pastas",
		container.NewVBox(
			widget.NewLabel("Procurando e processando documentos..."),
			widget.NewProgressBarInfinite(),
		),
		a.window,
	)
	progress.Show()

	go func() {
		count, err := a.scanSvc.ScanAll()
		progress.Hide()

		if err != nil {
			dialog.ShowError(
				fmt.Errorf("Falha ao atualizar: %s", uic.HumanizeError(err)),
				a.window,
			)
			return
		}

		dialog.ShowInformation(
			"Atualização concluída",
			fmt.Sprintf("%d documento(s) processado(s).\n\nVeja a aba Histórico para detalhes.",
				count),
			a.window,
		)
		a.historyTab.Reload()
		a.refreshHistoryBadge()
	}()
}

// refreshHistoryBadge atualiza o título da aba Histórico para incluir o
// contador de falhas novas, ou volta ao título base se zero.
func (a *App) refreshHistoryBadge() {
	count, err := a.historySvc.CountNewFailures()
	if err != nil {
		a.logger.Warn("falha ao contar falhas novas", "err", err)
		return
	}
	if count > 0 {
		if count == 1 {
			a.historyItem.Text = fmt.Sprintf("%s (1 falha)", historyTabBaseTitle)
		} else {
			a.historyItem.Text = fmt.Sprintf("%s (%d falhas)", historyTabBaseTitle, count)
		}
	} else {
		a.historyItem.Text = historyTabBaseTitle
	}
	a.tabs.Refresh()
}

// Run inicia o monitoramento, dispara o refresh periódico do badge e abre a
// janela. Bloqueia até o usuário fechar.
func (a *App) Run() {
	if err := a.monitor.StartMonitoring(); err != nil {
		a.logger.Error("falha ao iniciar monitoramento", "err", err)
	}
	// Cálculo inicial do badge (ex.: falhas que aconteceram quando o programa
	// estava fechado — pelo monitoramento contínuo da última sessão).
	a.refreshHistoryBadge()

	// Refresh periódico — pega falhas novas que aconteceram com o programa
	// aberto na aba Monitorar/Regras (sem o usuário visitar a aba Histórico).
	go a.runBadgeRefresher()

	a.window.ShowAndRun()
}

// runBadgeRefresher roda enquanto a janela estiver aberta, atualizando o
// badge da aba Histórico em intervalos regulares. Para quando doneCh fecha.
func (a *App) runBadgeRefresher() {
	t := time.NewTicker(failureBadgeRefresh)
	defer t.Stop()
	for {
		select {
		case <-a.doneCh:
			return
		case <-t.C:
			// Só atualiza se o usuário NÃO estiver na aba Histórico —
			// senão fica mudando o título embaixo do nariz dele.
			if a.tabs.Selected() == nil {
				continue
			}
			selected := a.tabs.Selected().Text
			if selected == a.historyItem.Text {
				continue
			}
			a.refreshHistoryBadge()
		}
	}
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
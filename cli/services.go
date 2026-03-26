package cli

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/warsmite/gamejanitor/config"
	"github.com/warsmite/gamejanitor/controller"
	"github.com/warsmite/gamejanitor/controller/auth"
	"github.com/warsmite/gamejanitor/controller/backup"
	"github.com/warsmite/gamejanitor/controller/event"
	"github.com/warsmite/gamejanitor/controller/gameserver"
	"github.com/warsmite/gamejanitor/controller/mod"
	"github.com/warsmite/gamejanitor/controller/orchestrator"
	"github.com/warsmite/gamejanitor/controller/schedule"
	"github.com/warsmite/gamejanitor/controller/settings"
	"github.com/warsmite/gamejanitor/controller/status"
	"github.com/warsmite/gamejanitor/controller/webhook"
	"github.com/warsmite/gamejanitor/games"
	"github.com/warsmite/gamejanitor/store"
)

type services struct {
	broadcaster     *controller.EventBus
	settingsSvc     *settings.SettingsService
	gameserverSvc   *gameserver.GameserverService
	querySvc        *status.QueryService
	statsPoller     *status.StatsPoller
	readyWatcher    *status.ReadyWatcher
	consoleSvc      *gameserver.ConsoleService
	fileSvc         *gameserver.FileService
	backupSvc       *backup.BackupService
	scheduler       *schedule.Scheduler
	scheduleSvc     *schedule.ScheduleService
	authSvc         *auth.AuthService
	statusMgr       *status.StatusManager
	statusSub       *status.StatusSubscriber
	eventHistorySvc *event.EventHistoryService
	webhookWorker   *webhook.WebhookWorker
	webhookSvc      *webhook.WebhookEndpointService
	workerNodeSvc   *orchestrator.WorkerNodeService
	modSvc          *mod.ModService
}

func initServices(database *sql.DB, dispatcher *orchestrator.Dispatcher, registry *orchestrator.Registry, gameStore *games.GameStore, cfg config.Config, logger *slog.Logger) (*services, error) {
	broadcaster := controller.NewEventBus()
	db := store.New(database)

	settingsSvc := settings.NewSettingsServiceWithMode(db, logger, cfg.Mode)

	// Apply config file runtime settings to DB on every startup
	settingsSvc.ApplyConfig(cfg.Settings)

	gameserverSvc := gameserver.NewGameserverService(db, dispatcher, broadcaster, settingsSvc, gameStore, cfg.DataDir, logger)
	querySvc := status.NewQueryService(db, broadcaster, gameStore, logger)
	statsPoller := status.NewStatsPoller(db, dispatcher, broadcaster, logger)
	readyWatcher := status.NewReadyWatcher(db, broadcaster, gameStore, logger)
	gameserverSvc.SetReadyWatcher(readyWatcher)
	consoleSvc := gameserver.NewConsoleService(db, dispatcher, gameStore, logger)
	fileSvc := gameserver.NewFileService(db, dispatcher, logger)

	backupStorage, err := initBackupStorage(cfg, logger)
	if err != nil {
		return nil, err
	}

	// Activity tracking for long-running worker dispatches and CRUD events
	activityTracker := gameserver.NewActivityTracker(db, logger)
	gameserverSvc.SetActivityTracker(activityTracker)

	gameserverSvc.SetBackupStore(backupStorage)
	backupSvc := backup.NewBackupService(db, dispatcher, gameserverSvc, gameStore, backupStorage, settingsSvc, broadcaster, logger)
	backupSvc.SetActivityTracker(activityTracker)
	scheduler := schedule.NewScheduler(db, backupSvc, gameserverSvc, consoleSvc, broadcaster, logger)
	scheduleSvc := schedule.NewScheduleService(db, scheduler, broadcaster, logger)
	authSvc := auth.NewAuthService(db, logger)
	statusMgr := status.NewStatusManager(db, broadcaster, querySvc, statsPoller, readyWatcher, dispatcher, registry, gameserverSvc.Start, logger)
	statusSub := status.NewStatusSubscriber(db, broadcaster, querySvc, statsPoller, logger)
	eventHistorySvc := event.NewEventHistoryService(db)
	webhookWorker := webhook.NewWebhookWorker(db, db, broadcaster, logger)
	webhookSvc := webhook.NewWebhookEndpointService(db, logger)
	workerNodeSvc := orchestrator.NewWorkerNodeService(db, registry, broadcaster, logger)
	optionsRegistry := games.NewOptionsRegistry(logger)
	modSvc := mod.NewModService(db, fileSvc, gameStore, settingsSvc, optionsRegistry, broadcaster, logger)

	return &services{
		broadcaster:     broadcaster,
		settingsSvc:     settingsSvc,
		gameserverSvc:   gameserverSvc,
		querySvc:        querySvc,
		statsPoller:     statsPoller,
		readyWatcher:    readyWatcher,
		consoleSvc:      consoleSvc,
		fileSvc:         fileSvc,
		backupSvc:       backupSvc,
		scheduler:       scheduler,
		scheduleSvc:     scheduleSvc,
		authSvc:         authSvc,
		statusMgr:       statusMgr,
		statusSub:       statusSub,
		eventHistorySvc: eventHistorySvc,
		webhookWorker:   webhookWorker,
		webhookSvc:      webhookSvc,
		workerNodeSvc:   workerNodeSvc,
		modSvc:          modSvc,
	}, nil
}

func initBackupStorage(cfg config.Config, logger *slog.Logger) (backup.Storage, error) {
	bs := cfg.BackupStore
	if bs == nil || bs.Type == "" || bs.Type == "local" {
		logger.Info("backup store: local", "path", cfg.DataDir)
		return backup.NewLocalStorage(cfg.DataDir), nil
	}

	if bs.Type == "s3" {
		s3Storage, err := backup.NewS3Storage(bs, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize backup store: %w", err)
		}
		return s3Storage, nil
	}

	return nil, fmt.Errorf("unknown backup_store type: %q (must be \"local\" or \"s3\")", bs.Type)
}

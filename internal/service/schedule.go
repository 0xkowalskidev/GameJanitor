package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/warsmite/gamejanitor/internal/models"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

type ScheduleService struct {
	db          *sql.DB
	scheduler   *Scheduler
	broadcaster *EventBus
	log         *slog.Logger
}

func NewScheduleService(db *sql.DB, scheduler *Scheduler, broadcaster *EventBus, log *slog.Logger) *ScheduleService {
	return &ScheduleService{db: db, scheduler: scheduler, broadcaster: broadcaster, log: log}
}

func (s *ScheduleService) ListSchedules(gameserverID string) ([]models.Schedule, error) {
	return models.ListSchedules(s.db, gameserverID)
}

func (s *ScheduleService) GetSchedule(id string) (*models.Schedule, error) {
	return models.GetSchedule(s.db, id)
}

func (s *ScheduleService) CreateSchedule(ctx context.Context, schedule *models.Schedule) error {
	if err := validateScheduleType(schedule.Type); err != nil {
		return err
	}
	if err := validateCronExpr(schedule.CronExpr); err != nil {
		return err
	}

	schedule.ID = uuid.New().String()

	s.log.Info("creating schedule", "id", schedule.ID, "name", schedule.Name, "type", schedule.Type, "gameserver_id", schedule.GameserverID)

	if err := models.CreateSchedule(s.db, schedule); err != nil {
		return err
	}

	if schedule.Enabled {
		if err := s.scheduler.AddSchedule(*schedule); err != nil {
			if delErr := models.DeleteSchedule(s.db, schedule.ID); delErr != nil {
				s.log.Error("failed to clean up schedule after cron registration failure", "id", schedule.ID, "error", delErr)
			}
			return fmt.Errorf("registering schedule with cron: %w", err)
		}
	}

	s.broadcaster.Publish(ScheduleActionEvent{
		Type:         EventScheduleCreate,
		Timestamp:    time.Now(),
		Actor:        ActorFromContext(ctx),
		GameserverID: schedule.GameserverID,
		ScheduleID:   schedule.ID,
		ScheduleName: schedule.Name,
	})

	return nil
}

func (s *ScheduleService) UpdateSchedule(ctx context.Context, schedule *models.Schedule) error {
	if err := validateScheduleType(schedule.Type); err != nil {
		return err
	}
	if err := validateCronExpr(schedule.CronExpr); err != nil {
		return err
	}

	s.log.Info("updating schedule", "id", schedule.ID)

	if err := models.UpdateSchedule(s.db, schedule); err != nil {
		return err
	}

	if err := s.scheduler.UpdateSchedule(*schedule); err != nil {
		return fmt.Errorf("updating schedule in cron: %w", err)
	}

	s.broadcaster.Publish(ScheduleActionEvent{
		Type:         EventScheduleUpdate,
		Timestamp:    time.Now(),
		Actor:        ActorFromContext(ctx),
		GameserverID: schedule.GameserverID,
		ScheduleID:   schedule.ID,
		ScheduleName: schedule.Name,
	})

	return nil
}

func (s *ScheduleService) DeleteSchedule(ctx context.Context, id string) error {
	schedule, err := models.GetSchedule(s.db, id)
	if err != nil {
		return fmt.Errorf("getting schedule for delete: %w", err)
	}

	s.log.Info("deleting schedule", "id", id)

	s.scheduler.RemoveSchedule(id)
	if err := models.DeleteSchedule(s.db, id); err != nil {
		return err
	}

	gsID := ""
	name := ""
	if schedule != nil {
		gsID = schedule.GameserverID
		name = schedule.Name
	}
	s.broadcaster.Publish(ScheduleActionEvent{
		Type:         EventScheduleDelete,
		Timestamp:    time.Now(),
		Actor:        ActorFromContext(ctx),
		GameserverID: gsID,
		ScheduleID:   id,
		ScheduleName: name,
	})

	return nil
}

func (s *ScheduleService) ToggleSchedule(ctx context.Context, id string) error {
	schedule, err := models.GetSchedule(s.db, id)
	if err != nil {
		return fmt.Errorf("getting schedule %s: %w", id, err)
	}
	if schedule == nil {
		return ErrNotFoundf("schedule %s not found", id)
	}

	schedule.Enabled = !schedule.Enabled

	s.log.Info("toggling schedule", "id", id, "enabled", schedule.Enabled)

	if err := models.UpdateSchedule(s.db, schedule); err != nil {
		return err
	}

	if err := s.scheduler.UpdateSchedule(*schedule); err != nil {
		return fmt.Errorf("updating schedule in cron after toggle: %w", err)
	}

	s.broadcaster.Publish(ScheduleActionEvent{
		Type:         EventScheduleUpdate,
		Timestamp:    time.Now(),
		Actor:        ActorFromContext(ctx),
		GameserverID: schedule.GameserverID,
		ScheduleID:   id,
		ScheduleName: schedule.Name,
	})

	return nil
}

var validScheduleTypes = map[string]bool{
	"restart": true,
	"backup":  true,
	"command": true,
	"update":  true,
}

func validateScheduleType(t string) error {
	if !validScheduleTypes[t] {
		return ErrBadRequestf("invalid schedule type: %s (must be restart, backup, command, or update)", t)
	}
	return nil
}

func validateCronExpr(expr string) error {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := parser.Parse(expr); err != nil {
		return ErrBadRequestf("invalid cron expression %q: %v", expr, err)
	}
	return nil
}

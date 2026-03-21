package service

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/0xkowalskidev/gamejanitor/internal/models"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

type ScheduleService struct {
	db        *sql.DB
	scheduler *Scheduler
	log       *slog.Logger
}

func NewScheduleService(db *sql.DB, scheduler *Scheduler, log *slog.Logger) *ScheduleService {
	return &ScheduleService{db: db, scheduler: scheduler, log: log}
}

func (s *ScheduleService) ListSchedules(gameserverID string) ([]models.Schedule, error) {
	return models.ListSchedules(s.db, gameserverID)
}

func (s *ScheduleService) GetSchedule(id string) (*models.Schedule, error) {
	return models.GetSchedule(s.db, id)
}

func (s *ScheduleService) CreateSchedule(schedule *models.Schedule) error {
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

	return nil
}

func (s *ScheduleService) UpdateSchedule(schedule *models.Schedule) error {
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

	return nil
}

func (s *ScheduleService) DeleteSchedule(id string) error {
	s.log.Info("deleting schedule", "id", id)

	s.scheduler.RemoveSchedule(id)
	return models.DeleteSchedule(s.db, id)
}

func (s *ScheduleService) ToggleSchedule(id string) error {
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
		return fmt.Errorf("invalid cron expression %q: %w", expr, err)
	}
	return nil
}

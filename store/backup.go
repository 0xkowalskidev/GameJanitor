package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/warsmite/gamejanitor/model"
)

type BackupStore struct {
	db *sql.DB
}

func NewBackupStore(db *sql.DB) *BackupStore {
	return &BackupStore{db: db}
}

var backupColumns = "id, gameserver_id, name, size_bytes, created_at"

// Status and ErrorReason are not in the backups table — they are derived from
// the latest backup activity via PopulateBackupStatus/PopulateBackupStatuses.
func scanBackup(row interface{ Scan(dest ...any) error }) (model.Backup, error) {
	var b model.Backup
	err := row.Scan(&b.ID, &b.GameserverID, &b.Name, &b.SizeBytes, &b.CreatedAt)
	// Default status until PopulateBackupStatus is called
	b.Status = model.BackupStatusCompleted
	return b, err
}

func (s *BackupStore) ListBackups(filter model.BackupFilter) ([]model.Backup, error) {
	query := "SELECT " + backupColumns + " FROM backups WHERE gameserver_id = ? ORDER BY created_at DESC"
	query = filter.Pagination.ApplyToQuery(query, 0)
	rows, err := s.db.Query(query, filter.GameserverID)
	if err != nil {
		return nil, fmt.Errorf("listing backups: %w", err)
	}
	defer rows.Close()

	var backups []model.Backup
	for rows.Next() {
		b, err := scanBackup(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning backup row: %w", err)
		}
		backups = append(backups, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	s.PopulateBackupStatuses(backups)
	return backups, nil
}

func (s *BackupStore) GetBackup(id string) (*model.Backup, error) {
	b, err := scanBackup(s.db.QueryRow("SELECT "+backupColumns+" FROM backups WHERE id = ?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting backup %s: %w", id, err)
	}
	s.PopulateBackupStatus(&b)
	return &b, nil
}

func (s *BackupStore) CreateBackup(b *model.Backup) error {
	b.CreatedAt = time.Now()

	_, err := s.db.Exec(
		"INSERT INTO backups (id, gameserver_id, name, size_bytes, created_at) VALUES (?, ?, ?, ?, ?)",
		b.ID, b.GameserverID, b.Name, b.SizeBytes, b.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating backup %s: %w", b.ID, err)
	}
	return nil
}

func (s *BackupStore) UpdateBackupSize(id string, sizeBytes int64) error {
	_, err := s.db.Exec("UPDATE backups SET size_bytes = ? WHERE id = ?", sizeBytes, id)
	if err != nil {
		return fmt.Errorf("updating backup %s size: %w", id, err)
	}
	return nil
}

// PopulateBackupStatus derives a backup's status and error_reason from the latest
// backup activity with this backup_id in its metadata. If no activity exists, defaults to "completed".
func (s *BackupStore) PopulateBackupStatus(b *model.Backup) {
	var activityStatus, activityError string
	err := s.db.QueryRow(
		`SELECT status, COALESCE(error, '')
		 FROM activity WHERE json_extract(data, '$.backup_id') = ?
		 ORDER BY started_at DESC LIMIT 1`, b.ID).Scan(&activityStatus, &activityError)
	if err != nil {
		b.Status = model.BackupStatusCompleted
		b.ErrorReason = ""
		return
	}
	switch activityStatus {
	case "running":
		b.Status = model.BackupStatusInProgress
	case "failed":
		b.Status = model.BackupStatusFailed
		b.ErrorReason = activityError
	default:
		b.Status = model.BackupStatusCompleted
	}
}

// PopulateBackupStatuses derives status for a slice of backups in a single batch query.
func (s *BackupStore) PopulateBackupStatuses(backups []model.Backup) {
	if len(backups) == 0 {
		return
	}

	byID := make(map[string]*model.Backup, len(backups))
	placeholders := make([]string, len(backups))
	args := make([]any, len(backups))
	for i := range backups {
		byID[backups[i].ID] = &backups[i]
		placeholders[i] = "?"
		args[i] = backups[i].ID
	}

	// Get the latest activity per backup_id
	query := `SELECT json_extract(data, '$.backup_id'), status, COALESCE(error, '')
		FROM activity a1
		WHERE json_extract(a1.data, '$.backup_id') IN (` + strings.Join(placeholders, ",") + `)
		AND a1.started_at = (
			SELECT MAX(a2.started_at) FROM activity a2
			WHERE json_extract(a2.data, '$.backup_id') = json_extract(a1.data, '$.backup_id')
		)`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		// Fall back to individual queries
		for i := range backups {
			s.PopulateBackupStatus(&backups[i])
		}
		return
	}
	defer rows.Close()

	for rows.Next() {
		var backupID, activityStatus, activityError string
		if err := rows.Scan(&backupID, &activityStatus, &activityError); err != nil {
			continue
		}
		b, ok := byID[backupID]
		if !ok {
			continue
		}
		switch activityStatus {
		case "running":
			b.Status = model.BackupStatusInProgress
		case "failed":
			b.Status = model.BackupStatusFailed
			b.ErrorReason = activityError
		default:
			b.Status = model.BackupStatusCompleted
		}
	}
}

func (s *BackupStore) DeleteBackupsByGameserver(gameserverID string) error {
	_, err := s.db.Exec("DELETE FROM backups WHERE gameserver_id = ?", gameserverID)
	if err != nil {
		return fmt.Errorf("deleting backups for gameserver %s: %w", gameserverID, err)
	}
	return nil
}

func (s *BackupStore) TotalBackupSizeByGameserver(gameserverID string) (int64, error) {
	var total int64
	err := s.db.QueryRow("SELECT COALESCE(SUM(size_bytes), 0) FROM backups WHERE gameserver_id = ?", gameserverID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("querying total backup size for gameserver %s: %w", gameserverID, err)
	}
	return total, nil
}

func (s *BackupStore) DeleteBackup(id string) error {
	result, err := s.db.Exec("DELETE FROM backups WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting backup %s: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for backup %s: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("backup %s not found", id)
	}
	return nil
}

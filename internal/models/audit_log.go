package models

import (
	"database/sql"
	"fmt"
	"time"
)

type AuditLog struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	TokenID      string    `json:"token_id"`
	TokenName    string    `json:"token_name"`
	IPAddress    string    `json:"ip_address"`
	StatusCode   int       `json:"status_code"`
}

type AuditLogFilter struct {
	Action       *string
	ResourceType *string
	ResourceID   *string
	TokenID      *string
	Limit        int
	Offset       int
}

func CreateAuditLog(db *sql.DB, entry *AuditLog) error {
	_, err := db.Exec(
		`INSERT INTO audit_log (id, timestamp, action, resource_type, resource_id, token_id, token_name, ip_address, status_code)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.Timestamp, entry.Action, entry.ResourceType, entry.ResourceID,
		entry.TokenID, entry.TokenName, entry.IPAddress, entry.StatusCode,
	)
	if err != nil {
		return fmt.Errorf("creating audit log entry: %w", err)
	}
	return nil
}

func ListAuditLogs(db *sql.DB, filter AuditLogFilter) ([]AuditLog, error) {
	query := "SELECT id, timestamp, action, resource_type, resource_id, token_id, token_name, ip_address, status_code FROM audit_log WHERE 1=1"
	var args []any

	if filter.Action != nil {
		query += " AND action = ?"
		args = append(args, *filter.Action)
	}
	if filter.ResourceType != nil {
		query += " AND resource_type = ?"
		args = append(args, *filter.ResourceType)
	}
	if filter.ResourceID != nil {
		query += " AND resource_id = ?"
		args = append(args, *filter.ResourceID)
	}
	if filter.TokenID != nil {
		query += " AND token_id = ?"
		args = append(args, *filter.TokenID)
	}

	query += " ORDER BY timestamp DESC"

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing audit logs: %w", err)
	}
	defer rows.Close()

	var entries []AuditLog
	for rows.Next() {
		var e AuditLog
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Action, &e.ResourceType, &e.ResourceID, &e.TokenID, &e.TokenName, &e.IPAddress, &e.StatusCode); err != nil {
			return nil, fmt.Errorf("scanning audit log row: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func PruneAuditLogs(db *sql.DB, olderThanDays int) (int64, error) {
	result, err := db.Exec("DELETE FROM audit_log WHERE timestamp < datetime('now', ?)", fmt.Sprintf("-%d days", olderThanDays))
	if err != nil {
		return 0, fmt.Errorf("pruning audit logs: %w", err)
	}
	return result.RowsAffected()
}

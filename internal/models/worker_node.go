package models

import (
	"database/sql"
	"fmt"
	"time"
)

type WorkerNode struct {
	ID             string
	LanIP          string
	ExternalIP     string
	PortRangeStart *int
	PortRangeEnd   *int
	MaxMemoryMB    *int
	MaxCPU         *float64
	MaxStorageMB   *int
	SFTPPort       int
	LastSeen       *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// UpsertWorkerNode inserts or updates a worker node's IP and last_seen fields.
// Does not touch port_range columns — those are managed separately via SetWorkerNodePortRange.
func UpsertWorkerNode(db *sql.DB, node *WorkerNode) error {
	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO worker_nodes (id, lan_ip, external_ip, last_seen, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			lan_ip = excluded.lan_ip,
			external_ip = excluded.external_ip,
			last_seen = excluded.last_seen,
			updated_at = excluded.updated_at`,
		node.ID, node.LanIP, node.ExternalIP, now, now, now,
	)
	if err != nil {
		return fmt.Errorf("upserting worker node %s: %w", node.ID, err)
	}
	return nil
}

func GetWorkerNode(db *sql.DB, id string) (*WorkerNode, error) {
	var n WorkerNode
	err := db.QueryRow(
		"SELECT id, lan_ip, external_ip, port_range_start, port_range_end, max_memory_mb, max_cpu, max_storage_mb, sftp_port, last_seen, created_at, updated_at FROM worker_nodes WHERE id = ?",
		id,
	).Scan(&n.ID, &n.LanIP, &n.ExternalIP, &n.PortRangeStart, &n.PortRangeEnd, &n.MaxMemoryMB, &n.MaxCPU, &n.MaxStorageMB, &n.SFTPPort, &n.LastSeen, &n.CreatedAt, &n.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting worker node %s: %w", id, err)
	}
	return &n, nil
}

func ListWorkerNodes(db *sql.DB) ([]WorkerNode, error) {
	rows, err := db.Query("SELECT id, lan_ip, external_ip, port_range_start, port_range_end, max_memory_mb, max_cpu, max_storage_mb, sftp_port, last_seen, created_at, updated_at FROM worker_nodes ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("listing worker nodes: %w", err)
	}
	defer rows.Close()

	var nodes []WorkerNode
	for rows.Next() {
		var n WorkerNode
		if err := rows.Scan(&n.ID, &n.LanIP, &n.ExternalIP, &n.PortRangeStart, &n.PortRangeEnd, &n.MaxMemoryMB, &n.MaxCPU, &n.MaxStorageMB, &n.SFTPPort, &n.LastSeen, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning worker node row: %w", err)
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

func SetWorkerNodePortRange(db *sql.DB, id string, start, end *int) error {
	result, err := db.Exec(
		"UPDATE worker_nodes SET port_range_start = ?, port_range_end = ?, updated_at = ? WHERE id = ?",
		start, end, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("setting port range for worker node %s: %w", id, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for worker node %s: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("worker node %s not found", id)
	}
	return nil
}

func SetWorkerNodeSFTPPort(db *sql.DB, id string, sftpPort int) error {
	result, err := db.Exec(
		"UPDATE worker_nodes SET sftp_port = ?, updated_at = ? WHERE id = ?",
		sftpPort, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("setting sftp port for worker node %s: %w", id, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for worker node %s: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("worker node %s not found", id)
	}
	return nil
}

func SetWorkerNodeLimits(db *sql.DB, id string, maxMemoryMB *int, maxCPU *float64, maxStorageMB *int) error {
	result, err := db.Exec(
		"UPDATE worker_nodes SET max_memory_mb = ?, max_cpu = ?, max_storage_mb = ?, updated_at = ? WHERE id = ?",
		maxMemoryMB, maxCPU, maxStorageMB, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("setting limits for worker node %s: %w", id, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for worker node %s: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("worker node %s not found", id)
	}
	return nil
}

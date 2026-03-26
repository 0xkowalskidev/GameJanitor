package model

import "time"

// Worker lifecycle states
const (
	WorkerStatusOnline  = "online"
	WorkerStatusOffline = "offline"
)

type WorkerNode struct {
	ID           string     `json:"id"`
	GRPCAddress  string     `json:"grpc_address"`
	LanIP        string     `json:"lan_ip"`
	ExternalIP   string     `json:"external_ip"`
	Status       string     `json:"status"`
	MaxMemoryMB  *int       `json:"max_memory_mb"`
	MaxCPU       *float64   `json:"max_cpu"`
	MaxStorageMB *int       `json:"max_storage_mb"`
	Cordoned       bool       `json:"cordoned"`
	Tags           Labels     `json:"tags"`
	PortRangeStart *int       `json:"port_range_start"`
	PortRangeEnd   *int       `json:"port_range_end"`
	SFTPPort       int        `json:"sftp_port"`
	LastSeen     *time.Time `json:"last_seen"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

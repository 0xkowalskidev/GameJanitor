package model

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Container user identity — game processes run as this UID/GID inside containers.
const (
	GameserverUID  = 1001
	GameserverGID  = 1001
	GameserverPerm = 0644
)

type GameserverNode struct {
	ExternalIP string `json:"external_ip"`
	LanIP      string `json:"lan_ip"`
}

type Gameserver struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	GameID        string          `json:"game_id"`
	Ports         json.RawMessage `json:"ports"`
	Env           json.RawMessage `json:"env"`
	MemoryLimitMB  int             `json:"memory_limit_mb"`
	CPULimit       float64         `json:"cpu_limit"`
	CPUEnforced    bool            `json:"cpu_enforced"`
	ContainerID    *string         `json:"container_id"`
	VolumeName     string          `json:"volume_name"`
	Status         string          `json:"status"`
	ErrorReason    string          `json:"error_reason"`
	PortMode       string          `json:"port_mode"`
	NodeID         *string         `json:"node_id"`
	Node           *GameserverNode `json:"node,omitempty"`
	SFTPUsername   string          `json:"sftp_username"`
	HashedSFTPPassword string      `json:"-"`
	Installed      bool            `json:"installed"`
	BackupLimit    *int            `json:"backup_limit"`
	StorageLimitMB *int            `json:"storage_limit_mb"`
	NodeTags       Labels          `json:"node_tags"`
	AutoRestart    bool            `json:"auto_restart"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// FlexInt handles JSON values that may be a number or a string containing a number.
// Used for port mappings where values come from user-provided JSON.
type FlexInt int

func (fi *FlexInt) UnmarshalJSON(b []byte) error {
	var n int
	if err := json.Unmarshal(b, &n); err == nil {
		*fi = FlexInt(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("cannot unmarshal %s into int or string", string(b))
	}
	if s == "" {
		*fi = 0
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("cannot parse %q as int: %w", s, err)
	}
	*fi = FlexInt(n)
	return nil
}

type GameserverFilter struct {
	GameID *string
	Status *string
	NodeID *string
	IDs    []string // restrict results to these IDs (used for scoped token filtering)
	Pagination
}

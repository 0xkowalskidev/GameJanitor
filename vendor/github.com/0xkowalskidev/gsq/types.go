package gsq

import (
	"time"

	"github.com/0xkowalskidev/gsq/internal/protocol"
)

type Duration = protocol.Duration
type ServerInfo = protocol.ServerInfo
type PlayerInfo = protocol.PlayerInfo

type QueryOptions struct {
	Game            string
	Timeout         time.Duration
	Players         bool
	EOSClientID     string
	EOSClientSecret string
}

type DiscoverOptions struct {
	Timeout    time.Duration
	PortRanges []PortRange
	Players    bool
}

type PortRange struct {
	Start uint16
	End   uint16
}

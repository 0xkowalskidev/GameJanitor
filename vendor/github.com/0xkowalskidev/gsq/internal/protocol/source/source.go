package source

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/0xkowalskidev/gsq/internal/protocol"
)

const (
	a2sInfoRequest  = 0x54
	a2sInfoResponse = 0x49

	a2sPlayerRequest  = 0x55
	a2sPlayerResponse = 0x44

	challengeResponse = 0x41
)

var a2sInfoPayload = append(
	[]byte{0xFF, 0xFF, 0xFF, 0xFF, a2sInfoRequest},
	append([]byte("Source Engine Query"), 0x00)...,
)

type SourceQuerier struct{}

func init() {
	protocol.Register("source", &SourceQuerier{})
}

func (q *SourceQuerier) Query(ctx context.Context, address string, port uint16, opts protocol.QueryOpts) (*protocol.ServerInfo, error) {
	dialHost := address
	if opts.ResolvedIP != "" {
		dialHost = opts.ResolvedIP
	}
	addr := net.JoinHostPort(dialHost, fmt.Sprintf("%d", port))

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second)
	}

	conn, err := net.DialTimeout("udp", addr, time.Until(deadline))
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", addr, err)
	}
	defer conn.Close()

	conn.SetDeadline(deadline)

	info, ping, err := queryInfo(conn)
	if err != nil {
		return nil, fmt.Errorf("a2s_info %s: %w", addr, err)
	}

	info.Ping = protocol.Duration{Duration: ping}
	info.Address = address
	info.GamePort = port
	info.QueryPort = port

	if opts.Players {
		// Short deadline for player query — some servers don't respond to it
		playerDeadline := time.Now().Add(200 * time.Millisecond)
		if playerDeadline.After(deadline) {
			playerDeadline = deadline
		}
		conn.SetDeadline(playerDeadline)

		players, err := queryPlayers(conn)
		if err != nil {
			return info, nil
		}
		info.PlayerList = players
	}

	return info, nil
}

func queryInfo(conn net.Conn) (*protocol.ServerInfo, time.Duration, error) {
	start := time.Now()
	if _, err := conn.Write(a2sInfoPayload); err != nil {
		return nil, 0, fmt.Errorf("send request: %w", err)
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, 0, fmt.Errorf("read response: %w", err)
	}
	ping := time.Since(start)

	data := buf[:n]

	if n < 5 {
		return nil, 0, fmt.Errorf("response too short: %d bytes", n)
	}

	if data[4] == challengeResponse {
		if n < 9 {
			return nil, 0, fmt.Errorf("challenge response too short: %d bytes", n)
		}
		challenge := data[5:9]
		retryPayload := make([]byte, len(a2sInfoPayload), len(a2sInfoPayload)+4)
		copy(retryPayload, a2sInfoPayload)
		retryPayload = append(retryPayload, challenge...)

		start = time.Now()
		if _, err := conn.Write(retryPayload); err != nil {
			return nil, 0, fmt.Errorf("send challenge response: %w", err)
		}

		n, err = conn.Read(buf)
		if err != nil {
			return nil, 0, fmt.Errorf("read challenge reply: %w", err)
		}
		ping = time.Since(start)
		data = buf[:n]
	}

	if data[4] != a2sInfoResponse {
		return nil, 0, fmt.Errorf("unexpected response type: 0x%02x", data[4])
	}

	info, err := parseInfoResponse(data[5:])
	return info, ping, err
}

func parseInfoResponse(data []byte) (*protocol.ServerInfo, error) {
	r := bytes.NewReader(data)

	if _, err := r.ReadByte(); err != nil {
		return nil, fmt.Errorf("read protocol version: %w", err)
	}

	name, err := readString(r)
	if err != nil {
		return nil, fmt.Errorf("read server name: %w", err)
	}

	mapName, err := readString(r)
	if err != nil {
		return nil, fmt.Errorf("read map: %w", err)
	}

	// folder — not used
	if _, err := readString(r); err != nil {
		return nil, fmt.Errorf("read folder: %w", err)
	}

	game, err := readString(r)
	if err != nil {
		return nil, fmt.Errorf("read game: %w", err)
	}

	// Fixed-size fields after the strings
	var fields struct {
		AppID       uint16
		Players     uint8
		MaxPlayers  uint8
		Bots        uint8
		ServerType  uint8
		Environment uint8
		Visibility  uint8
		VAC         uint8
	}
	if err := binary.Read(r, binary.LittleEndian, &fields); err != nil {
		return nil, fmt.Errorf("read server fields: %w", err)
	}

	version, _ := readString(r)

	info := &protocol.ServerInfo{
		Protocol:    "source",
		Name:        name,
		Map:         mapName,
		Game:        game,
		Players:     int(fields.Players),
		MaxPlayers:  int(fields.MaxPlayers),
		Bots:        int(fields.Bots),
		ServerType:  serverTypeString(fields.ServerType),
		Environment: environmentString(fields.Environment),
		Visibility:  visibilityString(fields.Visibility),
		VAC:         fields.VAC == 1,
		Version:     version,
	}

	// Parse Extra Data Flag (EDF) fields in spec order
	if edf, err := r.ReadByte(); err == nil {
		if edf&0x80 != 0 {
			r.Seek(2, io.SeekCurrent) // skip EDF game port — unreliable in containerized setups
		}
		if edf&0x10 != 0 {
			r.Seek(8, io.SeekCurrent) // skip SteamID (uint64)
		}
		if edf&0x40 != 0 {
			r.Seek(2, io.SeekCurrent) // skip spectator port (uint16)
			readString(r)             // skip spectator name
		}
		if edf&0x20 != 0 {
			readString(r) // skip keywords
		}
		if edf&0x01 != 0 {
			var gameID uint64
			if err := binary.Read(r, binary.LittleEndian, &gameID); err == nil {
				info.AppID = uint32(gameID & 0xFFFFFF)
			}
		}
	}

	// Fall back to truncated uint16 AppID if EDF GameID wasn't present
	if info.AppID == 0 {
		info.AppID = uint32(fields.AppID)
	}

	return info, nil
}

func queryPlayers(conn net.Conn) ([]protocol.PlayerInfo, error) {
	// Send initial request with 0xFFFFFFFF challenge to get the real challenge
	challengeReq := []byte{0xFF, 0xFF, 0xFF, 0xFF, a2sPlayerRequest, 0xFF, 0xFF, 0xFF, 0xFF}

	if _, err := conn.Write(challengeReq); err != nil {
		return nil, fmt.Errorf("send player challenge: %w", err)
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("read player challenge: %w", err)
	}

	data := buf[:n]

	if n >= 9 && data[4] == challengeResponse {
		challenge := data[5:9]
		playerReq := []byte{0xFF, 0xFF, 0xFF, 0xFF, a2sPlayerRequest}
		playerReq = append(playerReq, challenge...)

		if _, err := conn.Write(playerReq); err != nil {
			return nil, fmt.Errorf("send player request: %w", err)
		}

		n, err = conn.Read(buf)
		if err != nil {
			return nil, fmt.Errorf("read player response: %w", err)
		}
		data = buf[:n]
	}

	if len(data) < 6 || data[4] != a2sPlayerResponse {
		return nil, fmt.Errorf("unexpected player response type")
	}

	return parsePlayerResponse(data[5:])
}

func parsePlayerResponse(data []byte) ([]protocol.PlayerInfo, error) {
	r := bytes.NewReader(data)

	var playerCount uint8
	if err := binary.Read(r, binary.LittleEndian, &playerCount); err != nil {
		return nil, fmt.Errorf("read player count: %w", err)
	}

	players := make([]protocol.PlayerInfo, 0, playerCount)

	for i := 0; i < int(playerCount); i++ {
		var index uint8
		if err := binary.Read(r, binary.LittleEndian, &index); err != nil {
			break
		}
		_ = index

		name, err := readString(r)
		if err != nil {
			break
		}

		var score int32
		if err := binary.Read(r, binary.LittleEndian, &score); err != nil {
			break
		}

		var duration float32
		if err := binary.Read(r, binary.LittleEndian, &duration); err != nil {
			break
		}

		players = append(players, protocol.PlayerInfo{
			Name:     name,
			Score:    int(score),
			Duration: protocol.Duration{Duration: time.Duration(float64(time.Second) * float64(duration))},
		})
	}

	return players, nil
}

func readString(r *bytes.Reader) (string, error) {
	var result []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			return string(result), err
		}
		if b == 0 {
			return string(result), nil
		}
		result = append(result, b)
	}
}

func serverTypeString(b uint8) string {
	switch b {
	case 'd':
		return "dedicated"
	case 'l':
		return "listen"
	case 'p':
		return "proxy"
	default:
		return "unknown"
	}
}

func environmentString(b uint8) string {
	switch b {
	case 'l':
		return "linux"
	case 'w':
		return "windows"
	case 'm', 'o':
		return "mac"
	default:
		return "unknown"
	}
}

func visibilityString(b uint8) string {
	if b == 0 {
		return "public"
	}
	return "private"
}

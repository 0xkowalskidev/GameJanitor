package raknet

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/0xkowalskidev/gsq/internal/protocol"
)

var raknetMagic = []byte{
	0x00, 0xff, 0xff, 0x00, 0xfe, 0xfe, 0xfe, 0xfe,
	0xfd, 0xfd, 0xfd, 0xfd, 0x12, 0x34, 0x56, 0x78,
}

type raknetQuerier struct{}

func init() {
	protocol.Register("raknet", &raknetQuerier{})
}

func (q *raknetQuerier) Query(ctx context.Context, address string, port uint16, opts protocol.QueryOpts) (*protocol.ServerInfo, error) {
	host := address
	if opts.ResolvedIP != "" {
		host = opts.ResolvedIP
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second)
	}

	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP(host), Port: int(port)})
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	conn.SetDeadline(deadline)

	// Build Unconnected Ping packet (33 bytes)
	ping := make([]byte, 33)
	ping[0] = 0x01
	binary.BigEndian.PutUint64(ping[1:9], uint64(time.Now().UnixNano()))
	copy(ping[9:25], raknetMagic)
	// bytes 25-32: client GUID (zeros)

	start := time.Now()
	if _, err := conn.Write(ping); err != nil {
		return nil, fmt.Errorf("send ping: %w", err)
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("read pong: %w", err)
	}
	elapsed := time.Since(start)
	buf = buf[:n]

	slog.Debug("raknet: received pong", "bytes", n, "ping", elapsed)

	return parsePong(buf, address, port, elapsed)
}

func parsePong(buf []byte, address string, port uint16, ping time.Duration) (*protocol.ServerInfo, error) {
	// Minimum pong: 1 (id) + 8 (time) + 8 (guid) + 16 (magic) + 2 (strlen) = 35
	if len(buf) < 35 {
		return nil, fmt.Errorf("pong too short: %d bytes", len(buf))
	}
	if buf[0] != 0x1C {
		return nil, fmt.Errorf("unexpected packet id: 0x%02x", buf[0])
	}

	strLen := binary.BigEndian.Uint16(buf[33:35])
	if int(35+strLen) > len(buf) {
		return nil, fmt.Errorf("status string length %d exceeds packet", strLen)
	}
	status := string(buf[35 : 35+strLen])

	slog.Debug("raknet: status string", "status", status)

	fields := strings.Split(status, ";")
	if len(fields) < 6 {
		return nil, fmt.Errorf("expected at least 6 fields, got %d", len(fields))
	}

	players, _ := strconv.Atoi(fields[4])
	maxPlayers, _ := strconv.Atoi(fields[5])

	info := &protocol.ServerInfo{
		Protocol:   "raknet",
		Name:       fields[1],
		Players:    players,
		MaxPlayers: maxPlayers,
		Version:    fields[3],
		Ping:       protocol.Duration{Duration: ping},
		Address:    fmt.Sprintf("%s:%d", address, port),
		GamePort:   port,
		QueryPort:  port,
	}

	// Optional fields
	if len(fields) > 8 {
		info.Map = fields[8] // game mode string (Survival/Creative/Adventure)
	}
	if len(fields) > 10 {
		if p, err := strconv.Atoi(fields[10]); err == nil && p > 0 && p <= 65535 {
			info.GamePort = uint16(p)
		}
	}

	return info, nil
}

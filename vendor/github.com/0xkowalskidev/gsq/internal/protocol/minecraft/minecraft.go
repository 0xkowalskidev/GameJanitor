package minecraft

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/0xkowalskidev/gsq/internal/protocol"
)

type MinecraftQuerier struct{}

func init() {
	protocol.Register("minecraft", &MinecraftQuerier{})
}

// statusResponse is the JSON structure returned by the Minecraft server.
type statusResponse struct {
	Description interface{} `json:"description"`
	Players     struct {
		Max    int `json:"max"`
		Online int `json:"online"`
		Sample []struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"sample"`
	} `json:"players"`
	Version struct {
		Name     string `json:"name"`
		Protocol int    `json:"protocol"`
	} `json:"version"`
}

func (q *MinecraftQuerier) Query(ctx context.Context, address string, port uint16, opts protocol.QueryOpts) (*protocol.ServerInfo, error) {
	dialHost := address
	if opts.ResolvedIP != "" {
		dialHost = opts.ResolvedIP
	}
	addr := net.JoinHostPort(dialHost, fmt.Sprintf("%d", port))

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second)
	}

	dialer := net.Dialer{Deadline: deadline}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", addr, err)
	}
	defer conn.Close()

	conn.SetDeadline(deadline)

	start := time.Now()

	if err := sendHandshake(conn, address, port); err != nil {
		return nil, fmt.Errorf("handshake: %w", err)
	}

	if err := sendStatusRequest(conn); err != nil {
		return nil, fmt.Errorf("status request: %w", err)
	}

	status, err := readStatusResponse(conn)
	if err != nil {
		return nil, fmt.Errorf("read status: %w", err)
	}

	info := &protocol.ServerInfo{
		Ping: protocol.Duration{Duration: time.Since(start)},
		Protocol:   "minecraft",
		Name:       extractDescription(status.Description),
		Game:       "Minecraft",
		Players:    status.Players.Online,
		MaxPlayers: status.Players.Max,
		Version:    status.Version.Name,
		Address:    address,
		GamePort:   port,
		QueryPort:  port,
	}

	if opts.Players {
		for _, p := range status.Players.Sample {
			info.PlayerList = append(info.PlayerList, protocol.PlayerInfo{
				Name: p.Name,
			})
		}
	}

	return info, nil
}

func sendHandshake(conn net.Conn, host string, port uint16) error {
	var payload bytes.Buffer

	// Packet ID: 0x00 (Handshake)
	payload.WriteByte(0x00)

	// Protocol version (-1 for status ping)
	writeVarInt(&payload, -1)

	// Server address
	writeVarInt(&payload, int32(len(host)))
	payload.WriteString(host)

	// Server port
	binary.Write(&payload, binary.BigEndian, port)

	// Next state: 1 (status)
	writeVarInt(&payload, 1)

	return writePacket(conn, payload.Bytes())
}

func sendStatusRequest(conn net.Conn) error {
	// Packet ID: 0x00 (Status Request) with no payload
	return writePacket(conn, []byte{0x00})
}

func readStatusResponse(conn net.Conn) (*statusResponse, error) {
	data, err := readPacket(conn)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(data)

	// Packet ID
	packetID, err := readVarInt(r)
	if err != nil {
		return nil, fmt.Errorf("read packet id: %w", err)
	}
	if packetID != 0x00 {
		return nil, fmt.Errorf("unexpected packet id: 0x%02x", packetID)
	}

	// JSON string length
	strLen, err := readVarInt(r)
	if err != nil {
		return nil, fmt.Errorf("read string length: %w", err)
	}

	jsonData := make([]byte, strLen)
	if _, err := r.Read(jsonData); err != nil {
		return nil, fmt.Errorf("read json data: %w", err)
	}

	var status statusResponse
	if err := json.Unmarshal(jsonData, &status); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	return &status, nil
}

func writePacket(conn net.Conn, data []byte) error {
	var buf bytes.Buffer
	writeVarInt(&buf, int32(len(data)))
	buf.Write(data)
	_, err := conn.Write(buf.Bytes())
	return err
}

type connByteReader struct {
	conn net.Conn
	buf  [1]byte
}

func (r *connByteReader) ReadByte() (byte, error) {
	_, err := r.conn.Read(r.buf[:])
	return r.buf[0], err
}

func readPacket(conn net.Conn) ([]byte, error) {
	length, err := readVarInt(&connByteReader{conn: conn})
	if err != nil {
		return nil, fmt.Errorf("read packet length: %w", err)
	}

	if length < 0 || length > 1<<20 {
		return nil, fmt.Errorf("invalid packet length: %d", length)
	}

	data := make([]byte, length)
	total := 0
	for total < int(length) {
		n, err := conn.Read(data[total:])
		if err != nil {
			return nil, fmt.Errorf("read packet data: %w", err)
		}
		total += n
	}

	return data, nil
}

func writeVarInt(buf *bytes.Buffer, value int32) {
	uval := uint32(value)
	for {
		if uval&^0x7F == 0 {
			buf.WriteByte(byte(uval))
			return
		}
		buf.WriteByte(byte(uval&0x7F) | 0x80)
		uval >>= 7
	}
}

func readVarInt(r io.ByteReader) (int32, error) {
	var result int32
	var shift uint
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		result |= int32(b&0x7F) << shift
		if b&0x80 == 0 {
			return result, nil
		}
		shift += 7
		if shift >= 32 {
			return 0, fmt.Errorf("varint too long")
		}
	}
}

// extractDescription handles the various formats Minecraft servers use for the description field.
func extractDescription(desc interface{}) string {
	switch v := desc.(type) {
	case string:
		return v
	case map[string]interface{}:
		if text, ok := v["text"].(string); ok {
			return text
		}
	}
	return ""
}

package sftp

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"

	"github.com/0xkowalskidev/gamejanitor/internal/service"
	gosftp "github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type Server struct {
	listener  net.Listener
	sshConfig *ssh.ServerConfig
	fileSvc   *service.FileService
	authSvc   *service.AuthService
	settings  *service.SettingsService
	log       *slog.Logger
	done      chan struct{}
}

// NewServer creates an embedded SFTP server.
// Username = gameserver ID, password = auth token (or anything if auth is disabled).
func NewServer(fileSvc *service.FileService, authSvc *service.AuthService, settingsSvc *service.SettingsService, hostKeyPath string, log *slog.Logger) (*Server, error) {
	hostKey, err := loadOrCreateHostKey(hostKeyPath)
	if err != nil {
		return nil, fmt.Errorf("loading sftp host key: %w", err)
	}

	s := &Server{
		fileSvc:  fileSvc,
		authSvc:  authSvc,
		settings: settingsSvc,
		log:      log,
		done:     make(chan struct{}),
	}

	config := &ssh.ServerConfig{
		PasswordCallback: s.passwordCallback,
	}
	config.AddHostKey(hostKey)
	s.sshConfig = config

	return s, nil
}

func (s *Server) passwordCallback(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	gameserverID := conn.User()

	if !s.settings.GetAuthEnabled() {
		return &ssh.Permissions{
			Extensions: map[string]string{"gameserver_id": gameserverID},
		}, nil
	}

	token := s.authSvc.ValidateToken(string(password))
	if token == nil {
		return nil, fmt.Errorf("invalid token")
	}

	if !service.IsAdmin(token) && !service.HasPermission(token, gameserverID, "files") {
		return nil, fmt.Errorf("no files permission on gameserver %s", gameserverID)
	}

	return &ssh.Permissions{
		Extensions: map[string]string{"gameserver_id": gameserverID},
	}, nil
}

func (s *Server) ListenAndServe(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("sftp listen on %s: %w", addr, err)
	}
	s.listener = listener
	s.log.Info("sftp server listening", "addr", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return nil
			default:
				s.log.Error("accepting sftp connection", "error", err)
				continue
			}
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) Close() error {
	close(s.done)
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *Server) handleConnection(conn net.Conn) {
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.sshConfig)
	if err != nil {
		s.log.Debug("sftp handshake failed", "remote", conn.RemoteAddr(), "error", err)
		conn.Close()
		return
	}
	defer sshConn.Close()

	gameserverID := sshConn.Permissions.Extensions["gameserver_id"]
	s.log.Info("sftp session started", "remote", sshConn.RemoteAddr(), "gameserver_id", gameserverID)

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			s.log.Error("accepting sftp channel", "error", err)
			continue
		}

		go s.handleChannel(channel, requests, gameserverID)
	}
}

func (s *Server) handleChannel(channel ssh.Channel, requests <-chan *ssh.Request, gameserverID string) {
	defer channel.Close()

	for req := range requests {
		if req.Type != "subsystem" || string(req.Payload[4:]) != "sftp" {
			if req.WantReply {
				req.Reply(false, nil)
			}
			continue
		}
		req.Reply(true, nil)

		h := newHandler(s.fileSvc, gameserverID, s.log)
		server := gosftp.NewRequestServer(channel, h.Handlers())
		if err := server.Serve(); err != nil && err != io.EOF {
			s.log.Error("sftp session error", "gameserver_id", gameserverID, "error", err)
		}
		server.Close()
		return
	}
}

func loadOrCreateHostKey(keyPath string) (ssh.Signer, error) {
	keyBytes, err := os.ReadFile(keyPath)
	if err == nil {
		return ssh.ParsePrivateKey(keyBytes)
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating sftp host key: %w", err)
	}

	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("marshaling sftp host key: %w", err)
	}

	pemBlock := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})

	if err := os.MkdirAll(filepath.Dir(keyPath), 0755); err != nil {
		return nil, fmt.Errorf("creating sftp host key directory: %w", err)
	}
	if err := os.WriteFile(keyPath, pemBlock, 0600); err != nil {
		return nil, fmt.Errorf("writing sftp host key: %w", err)
	}

	return ssh.ParsePrivateKey(pemBlock)
}

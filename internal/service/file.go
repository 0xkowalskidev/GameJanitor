package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/0xkowalskidev/gamejanitor/internal/docker"
	"github.com/0xkowalskidev/gamejanitor/internal/models"
)

type FileService struct {
	db     *sql.DB
	docker *docker.Client
	log    *slog.Logger
}

type FileEntry struct {
	Name        string
	IsDir       bool
	Size        int64
	ModTime     string
	Permissions string
}

func NewFileService(db *sql.DB, dockerClient *docker.Client, log *slog.Logger) *FileService {
	return &FileService{db: db, docker: dockerClient, log: log}
}

func (s *FileService) ListDirectory(ctx context.Context, gameserverID string, dirPath string) ([]FileEntry, error) {
	dirPath, err := validatePath(dirPath)
	if err != nil {
		return nil, err
	}

	var entries []FileEntry
	err = s.withContainer(ctx, gameserverID, func(containerID string) error {
		exitCode, stdout, stderr, execErr := s.docker.Exec(ctx, containerID, []string{"ls", "-la", dirPath})
		if execErr != nil {
			return fmt.Errorf("listing directory %s: %w", dirPath, execErr)
		}
		if exitCode != 0 {
			return fmt.Errorf("listing directory %s failed: %s", dirPath, stderr)
		}
		entries = parseLsOutput(stdout)
		return nil
	})
	return entries, err
}

func (s *FileService) ReadFile(ctx context.Context, gameserverID string, filePath string) ([]byte, error) {
	filePath, err := validatePath(filePath)
	if err != nil {
		return nil, err
	}

	var content []byte
	err = s.withContainer(ctx, gameserverID, func(containerID string) error {
		var copyErr error
		content, copyErr = s.docker.CopyFromContainer(ctx, containerID, filePath)
		return copyErr
	})
	return content, err
}

func (s *FileService) WriteFile(ctx context.Context, gameserverID string, filePath string, content []byte) error {
	filePath, err := validatePath(filePath)
	if err != nil {
		return err
	}

	return s.withContainer(ctx, gameserverID, func(containerID string) error {
		return s.docker.CopyToContainer(ctx, containerID, filePath, content)
	})
}

func (s *FileService) DeletePath(ctx context.Context, gameserverID string, targetPath string) error {
	targetPath, err := validatePath(targetPath)
	if err != nil {
		return err
	}
	if targetPath == "/data" {
		return fmt.Errorf("cannot delete the root data directory")
	}

	return s.withContainer(ctx, gameserverID, func(containerID string) error {
		exitCode, _, stderr, execErr := s.docker.Exec(ctx, containerID, []string{"rm", "-rf", targetPath})
		if execErr != nil {
			return fmt.Errorf("deleting %s: %w", targetPath, execErr)
		}
		if exitCode != 0 {
			return fmt.Errorf("deleting %s failed: %s", targetPath, stderr)
		}
		return nil
	})
}

func (s *FileService) CreateDirectory(ctx context.Context, gameserverID string, dirPath string) error {
	dirPath, err := validatePath(dirPath)
	if err != nil {
		return err
	}

	return s.withContainer(ctx, gameserverID, func(containerID string) error {
		exitCode, _, stderr, execErr := s.docker.Exec(ctx, containerID, []string{"mkdir", "-p", dirPath})
		if execErr != nil {
			return fmt.Errorf("creating directory %s: %w", dirPath, execErr)
		}
		if exitCode != 0 {
			return fmt.Errorf("creating directory %s failed: %s", dirPath, stderr)
		}
		return nil
	})
}

// withContainer runs fn against either the gameserver's running container
// or a temporary container for stopped gameservers.
func (s *FileService) withContainer(ctx context.Context, gameserverID string, fn func(containerID string) error) error {
	gs, err := models.GetGameserver(s.db, gameserverID)
	if err != nil {
		return fmt.Errorf("getting gameserver %s: %w", gameserverID, err)
	}
	if gs == nil {
		return fmt.Errorf("gameserver %s not found", gameserverID)
	}

	if isRunningStatus(gs.Status) && gs.ContainerID != nil {
		s.log.Debug("file operation on running container", "gameserver_id", gameserverID)
		return fn(*gs.ContainerID)
	}

	if gs.Status != StatusStopped {
		return fmt.Errorf("cannot access files while gameserver is %s", gs.Status)
	}

	// Stopped gameserver — spin up a temp container
	game, err := models.GetGame(s.db, gs.GameID)
	if err != nil {
		return fmt.Errorf("getting game for gameserver %s: %w", gameserverID, err)
	}
	if game == nil {
		return fmt.Errorf("game %s not found for gameserver %s", gs.GameID, gameserverID)
	}

	tempName := "gamejanitor-files-" + gameserverID
	s.log.Info("creating temp container for file operation", "gameserver_id", gameserverID, "container_name", tempName)

	tempID, err := s.docker.CreateContainer(ctx, docker.ContainerOptions{
		Name:       tempName,
		Image:      game.Image,
		Env:        []string{},
		VolumeName: gs.VolumeName,
		Entrypoint: []string{"sleep", "infinity"},
	})
	if err != nil {
		return fmt.Errorf("creating temp container for file operation: %w", err)
	}
	defer func() {
		if stopErr := s.docker.StopContainer(ctx, tempID, 5); stopErr != nil {
			s.log.Warn("failed to stop temp file container", "error", stopErr)
		}
		if rmErr := s.docker.RemoveContainer(ctx, tempID); rmErr != nil {
			s.log.Warn("failed to remove temp file container", "error", rmErr)
		}
	}()

	if err := s.docker.StartContainer(ctx, tempID); err != nil {
		return fmt.Errorf("starting temp container for file operation: %w", err)
	}

	return fn(tempID)
}

// validatePath ensures the path is within /data and contains no traversal.
func validatePath(p string) (string, error) {
	cleaned := path.Clean(p)
	if !strings.HasPrefix(cleaned, "/data") {
		return "", fmt.Errorf("path must be within /data, got: %s", p)
	}
	return cleaned, nil
}

// parseLsOutput parses `ls -la` output into FileEntry structs.
func parseLsOutput(output string) []FileEntry {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var entries []FileEntry

	for _, line := range lines {
		if strings.HasPrefix(line, "total ") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		name := strings.Join(fields[8:], " ")
		// Handle symlinks: "name -> target"
		if idx := strings.Index(name, " -> "); idx >= 0 {
			name = name[:idx]
		}

		// Skip . and .. entries
		if name == "." || name == ".." {
			continue
		}

		perms := fields[0]
		isDir := len(perms) > 0 && perms[0] == 'd'
		size, _ := strconv.ParseInt(fields[4], 10, 64)
		modTime := fields[5] + " " + fields[6] + " " + fields[7]

		entries = append(entries, FileEntry{
			Name:        name,
			IsDir:       isDir,
			Size:        size,
			ModTime:     modTime,
			Permissions: perms,
		})
	}

	// Sort: directories first, then alphabetical by name (case-insensitive)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	return entries
}

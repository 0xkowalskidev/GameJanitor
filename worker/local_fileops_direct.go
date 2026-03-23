package worker

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// --- Direct access detection ---

// hasDirectAccess probes once whether we can read Docker volume mountpoints.
func (w *LocalWorker) hasDirectAccess(ctx context.Context, volumeName string) bool {
	w.directAccessOnce.Do(func() {
		mp, err := w.docker.VolumeMountpoint(ctx, volumeName)
		if err != nil {
			w.log.Warn("cannot resolve volume mountpoint, using sidecar fallback for file operations", "error", err)
			return
		}
		_, err = os.Stat(mp)
		if err != nil {
			w.log.Info("volume mountpoint not accessible, using sidecar fallback for file operations", "mountpoint", mp, "error", err)
			return
		}
		w.log.Info("direct volume access available, using fast path for file operations", "mountpoint", mp)
		w.directAccess = true
	})
	return w.directAccess
}

// --- Direct filesystem implementation ---

func (w *LocalWorker) volumePath(ctx context.Context, volumeName string, relPath string) (string, error) {
	mountpoint, err := w.volumeMountpoint(ctx, volumeName)
	if err != nil {
		return "", err
	}

	resolved := filepath.Join(mountpoint, filepath.Clean(relPath))
	if !strings.HasPrefix(resolved, mountpoint) {
		return "", fmt.Errorf("path %q escapes volume root", relPath)
	}
	return resolved, nil
}

func (w *LocalWorker) volumeMountpoint(ctx context.Context, volumeName string) (string, error) {
	w.mountMu.RLock()
	if mp, ok := w.mountCache[volumeName]; ok {
		w.mountMu.RUnlock()
		return mp, nil
	}
	w.mountMu.RUnlock()

	mp, err := w.docker.VolumeMountpoint(ctx, volumeName)
	if err != nil {
		return "", err
	}

	w.mountMu.Lock()
	w.mountCache[volumeName] = mp
	w.mountMu.Unlock()
	return mp, nil
}

func (w *LocalWorker) listFilesDirect(ctx context.Context, volumeName string, path string) ([]FileEntry, error) {
	hostPath, err := w.volumePath(ctx, volumeName, path)
	if err != nil {
		return nil, err
	}

	dirEntries, err := os.ReadDir(hostPath)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", path, err)
	}

	entries := make([]FileEntry, 0, len(dirEntries))
	for _, de := range dirEntries {
		info, err := de.Info()
		if err != nil {
			continue
		}
		entries = append(entries, FileEntry{
			Name:        de.Name(),
			IsDir:       de.IsDir(),
			Size:        info.Size(),
			ModTime:     info.ModTime(),
			Permissions: info.Mode().String(),
		})
	}

	sortFileEntries(entries)
	return entries, nil
}

func (w *LocalWorker) readFileDirect(ctx context.Context, volumeName string, path string) ([]byte, error) {
	hostPath, err := w.volumePath(ctx, volumeName, path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(hostPath)
}

func (w *LocalWorker) writeFileDirect(ctx context.Context, volumeName string, path string, content []byte, perm os.FileMode) error {
	hostPath, err := w.volumePath(ctx, volumeName, path)
	if err != nil {
		return err
	}
	if err := os.WriteFile(hostPath, content, perm); err != nil {
		return err
	}
	return os.Chown(hostPath, 1001, 1001)
}

func (w *LocalWorker) deletePathDirect(ctx context.Context, volumeName string, path string) error {
	hostPath, err := w.volumePath(ctx, volumeName, path)
	if err != nil {
		return err
	}
	return os.RemoveAll(hostPath)
}

func (w *LocalWorker) createDirectoryDirect(ctx context.Context, volumeName string, path string) error {
	hostPath, err := w.volumePath(ctx, volumeName, path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(hostPath, fs.ModePerm); err != nil {
		return err
	}
	return filepath.WalkDir(hostPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Chown(p, 1001, 1001)
	})
}

func (w *LocalWorker) renamePathDirect(ctx context.Context, volumeName string, from string, to string) error {
	fromPath, err := w.volumePath(ctx, volumeName, from)
	if err != nil {
		return err
	}
	toPath, err := w.volumePath(ctx, volumeName, to)
	if err != nil {
		return err
	}
	return os.Rename(fromPath, toPath)
}

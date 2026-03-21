package worker

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// --- Volume-level backup operations ---

func (w *LocalWorker) BackupVolume(ctx context.Context, volumeName string) (io.ReadCloser, error) {
	if w.hasDirectAccess(ctx, volumeName) {
		return w.backupVolumeDirect(ctx, volumeName)
	}
	return w.backupVolumeSidecar(ctx, volumeName)
}

func (w *LocalWorker) backupVolumeDirect(ctx context.Context, volumeName string) (io.ReadCloser, error) {
	mountpoint, err := w.volumeMountpoint(ctx, volumeName)
	if err != nil {
		return nil, fmt.Errorf("resolving volume mountpoint: %w", err)
	}

	pr, pw := io.Pipe()
	tw := tar.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer tw.Close()

		err := filepath.Walk(mountpoint, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(mountpoint, path)
			if err != nil {
				return err
			}
			// tar paths should be under "data/" to match container layout
			tarPath := filepath.Join("data", relPath)

			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			header.Name = tarPath

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(tw, f)
			return err
		})
		if err != nil {
			pw.CloseWithError(fmt.Errorf("creating tar from volume: %w", err))
		}
	}()

	return pr, nil
}

func (w *LocalWorker) backupVolumeSidecar(ctx context.Context, volumeName string) (io.ReadCloser, error) {
	sidecarID, err := w.ensureSidecar(ctx, volumeName)
	if err != nil {
		return nil, fmt.Errorf("ensuring sidecar for backup: %w", err)
	}
	return w.docker.CopyDirFromContainer(ctx, sidecarID, "/data")
}

func (w *LocalWorker) RestoreVolume(ctx context.Context, volumeName string, tarStream io.Reader) error {
	if w.hasDirectAccess(ctx, volumeName) {
		return w.restoreVolumeDirect(ctx, volumeName, tarStream)
	}
	return w.restoreVolumeSidecar(ctx, volumeName, tarStream)
}

func (w *LocalWorker) restoreVolumeDirect(ctx context.Context, volumeName string, tarStream io.Reader) error {
	mountpoint, err := w.volumeMountpoint(ctx, volumeName)
	if err != nil {
		return fmt.Errorf("resolving volume mountpoint: %w", err)
	}

	// Clear existing contents
	entries, err := os.ReadDir(mountpoint)
	if err != nil {
		return fmt.Errorf("reading volume directory: %w", err)
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(mountpoint, entry.Name())); err != nil {
			return fmt.Errorf("clearing volume: %w", err)
		}
	}

	// Extract tar
	tr := tar.NewReader(tarStream)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		// Strip "data/" prefix to get path relative to volume root
		relPath := header.Name
		if strings.HasPrefix(relPath, "data/") {
			relPath = strings.TrimPrefix(relPath, "data/")
		} else if relPath == "data" {
			continue
		}
		if relPath == "" || relPath == "." {
			continue
		}

		targetPath := filepath.Join(mountpoint, filepath.Clean(relPath))
		if !strings.HasPrefix(targetPath, mountpoint) {
			continue // path traversal protection
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("creating directory %s: %w", relPath, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("creating parent dir for %s: %w", relPath, err)
			}
			f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("creating file %s: %w", relPath, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("writing file %s: %w", relPath, err)
			}
			f.Close()
		}
	}

	return nil
}

func (w *LocalWorker) restoreVolumeSidecar(ctx context.Context, volumeName string, tarStream io.Reader) error {
	// Clear volume via remove + recreate
	if err := w.RemoveVolume(ctx, volumeName); err != nil {
		return fmt.Errorf("removing volume for restore: %w", err)
	}
	if err := w.CreateVolume(ctx, volumeName); err != nil {
		return fmt.Errorf("recreating volume for restore: %w", err)
	}

	// Get a fresh sidecar with the new volume
	sidecarID, err := w.ensureSidecar(ctx, volumeName)
	if err != nil {
		return fmt.Errorf("ensuring sidecar for restore: %w", err)
	}

	// Extract tar into sidecar's /data mount
	return w.docker.CopyTarToContainer(ctx, sidecarID, "/", tarStream)
}

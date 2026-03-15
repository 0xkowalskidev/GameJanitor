package sftp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"time"

	"github.com/0xkowalskidev/gamejanitor/internal/service"
	"github.com/0xkowalskidev/gamejanitor/internal/worker"
	gosftp "github.com/pkg/sftp"
)

type handler struct {
	fileSvc      *service.FileService
	gameserverID string
	log          *slog.Logger
}

func newHandler(fileSvc *service.FileService, gameserverID string, log *slog.Logger) *handler {
	return &handler{fileSvc: fileSvc, gameserverID: gameserverID, log: log}
}

func (h *handler) Handlers() gosftp.Handlers {
	return gosftp.Handlers{
		FileGet:  h,
		FilePut:  h,
		FileCmd:  h,
		FileList: h,
	}
}

// toDataPath converts an SFTP path (rooted at /) to a FileService path (rooted at /data).
func toDataPath(sftpPath string) string {
	cleaned := path.Clean("/" + sftpPath)
	return "/data" + cleaned
}

// Fileread implements gosftp.FileReader — reads the entire file into memory.
func (h *handler) Fileread(r *gosftp.Request) (io.ReaderAt, error) {
	ctx := context.Background()
	data, err := h.fileSvc.ReadFile(ctx, h.gameserverID, toDataPath(r.Filepath))
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// Filewrite implements gosftp.FileWriter — buffers writes, flushes on Close.
func (h *handler) Filewrite(r *gosftp.Request) (io.WriterAt, error) {
	return &writeBuffer{
		fileSvc:      h.fileSvc,
		gameserverID: h.gameserverID,
		path:         toDataPath(r.Filepath),
	}, nil
}

// Filecmd implements gosftp.FileCmder.
func (h *handler) Filecmd(r *gosftp.Request) error {
	ctx := context.Background()

	switch r.Method {
	case "Rename":
		return h.fileSvc.RenamePath(ctx, h.gameserverID, toDataPath(r.Filepath), toDataPath(r.Target))
	case "Remove", "Rmdir":
		return h.fileSvc.DeletePath(ctx, h.gameserverID, toDataPath(r.Filepath))
	case "Mkdir":
		return h.fileSvc.CreateDirectory(ctx, h.gameserverID, toDataPath(r.Filepath))
	case "Setstat":
		return nil
	default:
		return fmt.Errorf("unsupported sftp command: %s", r.Method)
	}
}

// Filelist implements gosftp.FileLister.
func (h *handler) Filelist(r *gosftp.Request) (gosftp.ListerAt, error) {
	ctx := context.Background()
	dataPath := toDataPath(r.Filepath)

	switch r.Method {
	case "List":
		entries, err := h.fileSvc.ListDirectory(ctx, h.gameserverID, dataPath)
		if err != nil {
			return nil, err
		}
		infos := make([]os.FileInfo, len(entries))
		for i, e := range entries {
			infos[i] = fileEntryInfo(e)
		}
		return listAt(infos), nil

	case "Stat":
		if dataPath == "/data" || dataPath == "/data/" {
			return listAt([]os.FileInfo{&syntheticFileInfo{
				name: "/", isDir: true, mode: os.ModeDir | 0755,
			}}), nil
		}
		dir := path.Dir(dataPath)
		base := path.Base(dataPath)
		entries, err := h.fileSvc.ListDirectory(ctx, h.gameserverID, dir)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.Name == base {
				return listAt([]os.FileInfo{fileEntryInfo(e)}), nil
			}
		}
		return nil, os.ErrNotExist

	default:
		return nil, fmt.Errorf("unsupported sftp list method: %s", r.Method)
	}
}

// writeBuffer accumulates WriteAt calls and flushes to FileService on Close.
type writeBuffer struct {
	fileSvc      *service.FileService
	gameserverID string
	path         string
	buf          []byte
	maxOffset    int64
}

func (w *writeBuffer) WriteAt(p []byte, off int64) (int, error) {
	end := off + int64(len(p))
	if end > w.maxOffset {
		w.maxOffset = end
	}
	if int64(len(w.buf)) < end {
		newBuf := make([]byte, end)
		copy(newBuf, w.buf)
		w.buf = newBuf
	}
	copy(w.buf[off:], p)
	return len(p), nil
}

func (w *writeBuffer) Close() error {
	ctx := context.Background()
	return w.fileSvc.WriteFile(ctx, w.gameserverID, w.path, w.buf[:w.maxOffset])
}

// listAt implements gosftp.ListerAt over a slice of os.FileInfo.
type listAt []os.FileInfo

func (l listAt) ListAt(ls []os.FileInfo, offset int64) (int, error) {
	if offset >= int64(len(l)) {
		return 0, io.EOF
	}
	n := copy(ls, l[offset:])
	if n < len(ls) {
		return n, io.EOF
	}
	return n, nil
}

func fileEntryInfo(e worker.FileEntry) os.FileInfo {
	mode := os.FileMode(0644)
	if e.IsDir {
		mode = os.ModeDir | 0755
	}
	return &syntheticFileInfo{
		name: e.Name, size: e.Size, mode: mode, modTime: e.ModTime, isDir: e.IsDir,
	}
}

type syntheticFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (f *syntheticFileInfo) Name() string      { return f.name }
func (f *syntheticFileInfo) Size() int64       { return f.size }
func (f *syntheticFileInfo) Mode() os.FileMode { return f.mode }
func (f *syntheticFileInfo) ModTime() time.Time {
	if f.modTime.IsZero() {
		return time.Now()
	}
	return f.modTime
}
func (f *syntheticFileInfo) IsDir() bool      { return f.isDir }
func (f *syntheticFileInfo) Sys() interface{} { return nil }

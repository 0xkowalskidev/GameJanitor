package sftp

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"time"

	gosftp "github.com/pkg/sftp"
)

type handler struct {
	fileOp     FileOperator
	volumeName string
	log        *slog.Logger
}

func newHandler(fileOp FileOperator, volumeName string, log *slog.Logger) *handler {
	return &handler{fileOp: fileOp, volumeName: volumeName, log: log}
}

func (h *handler) Handlers() gosftp.Handlers {
	return gosftp.Handlers{
		FileGet:  h,
		FilePut:  h,
		FileCmd:  h,
		FileList: h,
	}
}

func cleanPath(sftpPath string) string {
	return path.Clean("/" + sftpPath)
}

func (h *handler) Fileread(r *gosftp.Request) (io.ReaderAt, error) {
	data, err := h.fileOp.ReadFile(h.volumeName, cleanPath(r.Filepath))
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (h *handler) Filewrite(r *gosftp.Request) (io.WriterAt, error) {
	return &writeBuffer{
		fileOp:     h.fileOp,
		volumeName: h.volumeName,
		path:       cleanPath(r.Filepath),
	}, nil
}

func (h *handler) Filecmd(r *gosftp.Request) error {
	switch r.Method {
	case "Rename":
		return h.fileOp.RenamePath(h.volumeName, cleanPath(r.Filepath), cleanPath(r.Target))
	case "Remove", "Rmdir":
		return h.fileOp.DeletePath(h.volumeName, cleanPath(r.Filepath))
	case "Mkdir":
		return h.fileOp.CreateDirectory(h.volumeName, cleanPath(r.Filepath))
	case "Setstat":
		return nil
	default:
		return fmt.Errorf("unsupported sftp command: %s", r.Method)
	}
}

func (h *handler) Filelist(r *gosftp.Request) (gosftp.ListerAt, error) {
	p := cleanPath(r.Filepath)

	switch r.Method {
	case "List":
		entries, err := h.fileOp.ListFiles(h.volumeName, p)
		if err != nil {
			return nil, err
		}
		infos := make([]os.FileInfo, len(entries))
		for i, e := range entries {
			infos[i] = fileEntryInfo(e)
		}
		return listAt(infos), nil

	case "Stat":
		if p == "/" {
			return listAt([]os.FileInfo{&syntheticFileInfo{
				name: "/", isDir: true, mode: os.ModeDir | 0755,
			}}), nil
		}
		dir := path.Dir(p)
		base := path.Base(p)
		entries, err := h.fileOp.ListFiles(h.volumeName, dir)
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

type writeBuffer struct {
	fileOp     FileOperator
	volumeName string
	path       string
	buf        []byte
	maxOffset  int64
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
	return w.fileOp.WriteFile(w.volumeName, w.path, w.buf[:w.maxOffset], 0644)
}

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

func fileEntryInfo(e FileEntry) os.FileInfo {
	mode := os.FileMode(0644)
	if e.IsDir {
		mode = os.ModeDir | 0755
	}
	var modTime time.Time
	if e.ModTime > 0 {
		modTime = time.Unix(e.ModTime, 0)
	}
	return &syntheticFileInfo{
		name: e.Name, size: e.Size, mode: mode, modTime: modTime, isDir: e.IsDir,
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

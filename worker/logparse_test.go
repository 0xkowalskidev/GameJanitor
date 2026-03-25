package worker

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func buildDockerFrame(streamType byte, text string) []byte {
	var buf bytes.Buffer
	header := make([]byte, 8)
	header[0] = streamType
	binary.BigEndian.PutUint32(header[4:8], uint32(len(text)))
	buf.Write(header)
	buf.WriteString(text)
	return buf.Bytes()
}

func TestLogParse_RawFormat_SimpleLines(t *testing.T) {
	t.Parallel()
	input := "line one\nline two\nline three\n"
	lines := ParseLogLines(strings.NewReader(input))
	assert.Equal(t, []string{"line one", "line two", "line three"}, lines)
}

func TestLogParse_RawFormat_EmptyLinesFiltered(t *testing.T) {
	t.Parallel()
	input := "line one\n\n\nline two\n\n"
	lines := ParseLogLines(strings.NewReader(input))
	assert.Equal(t, []string{"line one", "line two"}, lines)
}

func TestLogParse_DockerMultiplexed_Stdout(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	buf.Write(buildDockerFrame(1, "hello world\n"))
	buf.Write(buildDockerFrame(1, "second line\n"))
	lines := ParseLogLines(&buf)
	assert.Equal(t, []string{"hello world", "second line"}, lines)
}

func TestLogParse_DockerMultiplexed_Stderr(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	buf.Write(buildDockerFrame(2, "error occurred\n"))
	lines := ParseLogLines(&buf)
	assert.Equal(t, []string{"[ERR] error occurred"}, lines)
}

func TestLogParse_DockerMultiplexed_MixedStreams(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	buf.Write(buildDockerFrame(1, "stdout line\n"))
	buf.Write(buildDockerFrame(2, "stderr line\n"))
	buf.Write(buildDockerFrame(1, "another stdout\n"))
	lines := ParseLogLines(&buf)
	assert.Equal(t, []string{"stdout line", "[ERR] stderr line", "another stdout"}, lines)
}

func TestLogParse_DockerMultiplexed_MultilineFrame(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	buf.Write(buildDockerFrame(1, "line one\nline two\nline three\n"))
	lines := ParseLogLines(&buf)
	assert.Equal(t, []string{"line one", "line two", "line three"}, lines)
}

func TestLogParse_AutoDetection_Raw(t *testing.T) {
	t.Parallel()
	input := "Server starting...\nReady!\n"
	lines := ParseLogLines(strings.NewReader(input))
	assert.Equal(t, []string{"Server starting...", "Ready!"}, lines)
}

func TestLogParse_AutoDetection_Docker(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	buf.Write(buildDockerFrame(1, "docker line\n"))
	lines := ParseLogLines(&buf)
	assert.Equal(t, []string{"docker line"}, lines)
}

func TestLogParse_EmptyInput(t *testing.T) {
	t.Parallel()
	lines := ParseLogLines(strings.NewReader(""))
	assert.Empty(t, lines)
}

func TestLogParse_ParseLogStream_SendsToChannel(t *testing.T) {
	t.Parallel()
	input := "line one\nline two\n"
	ch := make(chan string, 10)
	ParseLogStream(strings.NewReader(input), ch)
	close(ch)
	var received []string
	for line := range ch {
		received = append(received, line)
	}
	assert.Equal(t, []string{"line one", "line two"}, received)
}

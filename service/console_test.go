package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/warsmite/gamejanitor/models"
	"github.com/warsmite/gamejanitor/testutil"
)

func TestConsole_SendCommand_GameserverNotFound(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)
	testutil.RegisterFakeWorker(t, svc, "worker-1")
	ctx := testutil.TestContext()

	_, err := svc.ConsoleSvc.SendCommand(ctx, "nonexistent", "say hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestConsole_SendCommand_NotRunning(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)
	testutil.RegisterFakeWorker(t, svc, "worker-1")
	ctx := testutil.TestContext()

	gs := testutil.CreateTestGameserver(t, svc)

	// Gameserver is stopped — command should fail
	_, err := svc.ConsoleSvc.SendCommand(ctx, gs.ID, "say hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no container")
}

func TestConsole_SendCommand_HappyPath(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)
	testutil.RegisterFakeWorker(t, svc, "worker-1")
	ctx := testutil.TestContext()

	gs := testutil.CreateTestGameserver(t, svc)

	// Start to get a container
	require.NoError(t, svc.GameserverSvc.Start(ctx, gs.ID))

	// Manually set status to running so the check passes
	fetched, _ := svc.GameserverSvc.GetGameserver(gs.ID)
	fetched.Status = "running"
	models.UpdateGameserver(svc.DB, fetched)

	// SendCommand — fake worker's Exec returns (0, "", "", nil)
	output, err := svc.ConsoleSvc.SendCommand(ctx, gs.ID, "say hello")
	require.NoError(t, err)
	assert.Empty(t, output) // fake worker returns empty stdout
}

func TestConsole_StreamLogs_NotRunning(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)
	testutil.RegisterFakeWorker(t, svc, "worker-1")
	ctx := testutil.TestContext()

	gs := testutil.CreateTestGameserver(t, svc)

	_, err := svc.ConsoleSvc.StreamLogs(ctx, gs.ID, 100)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no container")
}

func TestConsole_ListLogSessions_EmptyVolume(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)
	testutil.RegisterFakeWorker(t, svc, "worker-1")
	ctx := testutil.TestContext()

	gs := testutil.CreateTestGameserver(t, svc)

	// No log files exist yet — should return empty/nil
	sessions, err := svc.ConsoleSvc.ListLogSessions(ctx, gs.ID)
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestConsole_ListLogSessions_WithLogs(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)
	fw := testutil.RegisterFakeWorker(t, svc, "worker-1")
	ctx := testutil.TestContext()

	gs := testutil.CreateTestGameserver(t, svc)

	// Write fake log files to the volume
	require.NoError(t, fw.CreateDirectory(ctx, gs.VolumeName, "/.gamejanitor/logs"))
	require.NoError(t, fw.WriteFile(ctx, gs.VolumeName, "/.gamejanitor/logs/console.log", []byte("current session\n"), 0644))
	require.NoError(t, fw.WriteFile(ctx, gs.VolumeName, "/.gamejanitor/logs/console.log.0", []byte("previous session\n"), 0644))

	sessions, err := svc.ConsoleSvc.ListLogSessions(ctx, gs.ID)
	require.NoError(t, err)
	assert.Len(t, sessions, 2)
	assert.Equal(t, 0, sessions[0].Index) // console.log = index 0
	assert.Equal(t, 1, sessions[1].Index) // console.log.0 = index 1
}

func TestConsole_ReadHistoricalLogs(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)
	fw := testutil.RegisterFakeWorker(t, svc, "worker-1")
	ctx := testutil.TestContext()

	gs := testutil.CreateTestGameserver(t, svc)

	require.NoError(t, fw.CreateDirectory(ctx, gs.VolumeName, "/.gamejanitor/logs"))
	require.NoError(t, fw.WriteFile(ctx, gs.VolumeName, "/.gamejanitor/logs/console.log", []byte("line 1\nline 2\nline 3\n"), 0644))

	lines, err := svc.ConsoleSvc.ReadHistoricalLogs(ctx, gs.ID, 0, 0)
	require.NoError(t, err)
	assert.Len(t, lines, 3)
	assert.Equal(t, "line 1", lines[0])
}

func TestConsole_ReadHistoricalLogs_Tail(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)
	fw := testutil.RegisterFakeWorker(t, svc, "worker-1")
	ctx := testutil.TestContext()

	gs := testutil.CreateTestGameserver(t, svc)

	require.NoError(t, fw.CreateDirectory(ctx, gs.VolumeName, "/.gamejanitor/logs"))
	require.NoError(t, fw.WriteFile(ctx, gs.VolumeName, "/.gamejanitor/logs/console.log", []byte("line 1\nline 2\nline 3\nline 4\nline 5\n"), 0644))

	lines, err := svc.ConsoleSvc.ReadHistoricalLogs(ctx, gs.ID, 0, 2)
	require.NoError(t, err)
	assert.Len(t, lines, 2)
	assert.Equal(t, "line 4", lines[0])
	assert.Equal(t, "line 5", lines[1])
}

func TestConsole_ReadHistoricalLogs_GameserverNotFound(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)
	testutil.RegisterFakeWorker(t, svc, "worker-1")
	ctx := testutil.TestContext()

	_, err := svc.ConsoleSvc.ReadHistoricalLogs(ctx, "nonexistent", 0, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

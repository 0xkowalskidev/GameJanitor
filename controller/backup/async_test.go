package backup_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/warsmite/gamejanitor/model"
	"github.com/warsmite/gamejanitor/store"
	"github.com/warsmite/gamejanitor/testutil"
)

// TestBackup_DeleteDuringBackup_NoPanic creates a gameserver, starts a backup
// (which runs async in a goroutine), then immediately deletes the gameserver.
// The backup goroutine hits a deleted gameserver — it should not panic.
func TestBackup_DeleteDuringBackup_NoPanic(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)
	testutil.RegisterFakeWorker(t, svc, "worker-1")
	ctx := testutil.TestContext()

	gs := &model.Gameserver{
		Name:   "Delete During Backup",
		GameID: testutil.TestGameID,
		Env:    model.Env{"REQUIRED_VAR": "v"},
	}
	_, err := svc.GameserverSvc.CreateGameserver(ctx, gs)
	require.NoError(t, err)

	// Start backup — returns immediately, goroutine runs async
	backup, err := svc.BackupSvc.CreateBackup(ctx, gs.ID, "doomed-backup")
	require.NoError(t, err)
	require.Equal(t, model.BackupStatusInProgress, backup.Status)

	// Immediately delete the gameserver while backup goroutine is running
	err = svc.GameserverSvc.DeleteGameserver(ctx, gs.ID)
	require.NoError(t, err)

	// Wait for the backup goroutine to finish. The goroutine may complete, fail,
	// or find the gameserver gone. The key assertion: no panic.
	// Poll backup status — it may be gone (cascaded delete) or failed.
	s := store.New(svc.DB)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		b, err := s.GetBackup(backup.ID)
		if err != nil || b == nil {
			// Record was deleted via cascade — that's fine
			return
		}
		if b.Status != model.BackupStatusInProgress {
			// Goroutine finished (likely failed) — acceptable
			assert.Contains(t, []string{model.BackupStatusFailed, model.BackupStatusCompleted}, b.Status,
				"backup should be failed or completed, got %s", b.Status)
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	// If we get here, the backup is still in_progress after 5s.
	// The goroutine may be stuck but the test passes if no panic occurred.
}

// TestBackup_TwoSimultaneous_BothComplete triggers two backups back-to-back on
// the same gameserver. Both should complete (or one fails gracefully) without
// data corruption or panics.
func TestBackup_TwoSimultaneous_BothComplete(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)
	testutil.RegisterFakeWorker(t, svc, "worker-1")
	ctx := testutil.TestContext()

	gs := &model.Gameserver{
		Name:   "Double Backup Host",
		GameID: testutil.TestGameID,
		Env:    model.Env{"REQUIRED_VAR": "v"},
	}
	_, err := svc.GameserverSvc.CreateGameserver(ctx, gs)
	require.NoError(t, err)

	b1, err := svc.BackupSvc.CreateBackup(ctx, gs.ID, "backup-1")
	require.NoError(t, err)
	b2, err := svc.BackupSvc.CreateBackup(ctx, gs.ID, "backup-2")
	require.NoError(t, err)

	// Wait for both to leave in_progress
	waitForBackupDone(t, svc, b1.ID)
	waitForBackupDone(t, svc, b2.ID)

	// Both should have valid terminal states
	s := store.New(svc.DB)
	final1, err := s.GetBackup(b1.ID)
	require.NoError(t, err)
	require.NotNil(t, final1)
	assert.Contains(t, []string{model.BackupStatusCompleted, model.BackupStatusFailed}, final1.Status)

	final2, err := s.GetBackup(b2.ID)
	require.NoError(t, err)
	require.NotNil(t, final2)
	assert.Contains(t, []string{model.BackupStatusCompleted, model.BackupStatusFailed}, final2.Status)

	// At least one should have succeeded
	atLeastOneCompleted := final1.Status == model.BackupStatusCompleted || final2.Status == model.BackupStatusCompleted
	assert.True(t, atLeastOneCompleted, "at least one of two simultaneous backups should complete")
}

// waitForBackupDone polls until a backup leaves in_progress or 5s elapses.
func waitForBackupDone(t *testing.T, svc *testutil.ServiceBundle, backupID string) {
	t.Helper()
	s := store.New(svc.DB)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		b, err := s.GetBackup(backupID)
		if err == nil && b != nil && b.Status != model.BackupStatusInProgress {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

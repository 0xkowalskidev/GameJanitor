package service_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/warsmite/gamejanitor/models"
	"github.com/warsmite/gamejanitor/testutil"
)

func TestReady_InstallMarkerDetected(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServicesWithSubscribers(t)
	testutil.RegisterFakeWorker(t, svc, "worker-1")
	ctx := testutil.TestContext()

	gs := testutil.CreateTestGameserver(t, svc)

	// Gameserver starts as not installed
	assert.False(t, gs.Installed)

	// Start — the fake worker writes "[gamejanitor:installed]" to the log buffer,
	// which the ReadyWatcher should detect and set installed=true
	require.NoError(t, svc.GameserverSvc.Start(ctx, gs.ID))

	pollUntil(t, func() bool {
		fetched, _ := svc.GameserverSvc.GetGameserver(gs.ID)
		return fetched != nil && fetched.Installed
	}, "installed flag should be set after install marker detected")
}

func TestReady_ReadyPatternPromotesToRunning(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServicesWithSubscribers(t)
	testutil.RegisterFakeWorker(t, svc, "worker-1")
	ctx := testutil.TestContext()

	gs := testutil.CreateTestGameserver(t, svc)

	// The fake worker has ReadyPattern = "Server is ready" which matches
	// the test-game's ready_pattern. After start, the ReadyWatcher should
	// detect the pattern and publish a GameserverReady event, which the
	// StatusSubscriber promotes to "running".
	require.NoError(t, svc.GameserverSvc.Start(ctx, gs.ID))

	pollUntil(t, func() bool {
		fetched, _ := svc.GameserverSvc.GetGameserver(gs.ID)
		return fetched != nil && fetched.Status == "running"
	}, "status should be promoted to running after ready pattern match")
}

func TestReady_StopCancelsWatcher(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServicesWithSubscribers(t)
	testutil.RegisterFakeWorker(t, svc, "worker-1")
	ctx := testutil.TestContext()

	gs := testutil.CreateTestGameserver(t, svc)
	require.NoError(t, svc.GameserverSvc.Start(ctx, gs.ID))

	// Wait for running
	pollUntil(t, func() bool {
		f, _ := svc.GameserverSvc.GetGameserver(gs.ID)
		return f != nil && f.Status == "running"
	}, "should reach running")

	// Stop — should cancel the watcher without error
	require.NoError(t, svc.GameserverSvc.Stop(ctx, gs.ID))

	pollUntil(t, func() bool {
		f, _ := svc.GameserverSvc.GetGameserver(gs.ID)
		return f != nil && f.Status == "stopped"
	}, "should return to stopped")
}

func TestReady_AlreadyInstalled_SkipsInstallMarker(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServicesWithSubscribers(t)
	testutil.RegisterFakeWorker(t, svc, "worker-1")
	ctx := testutil.TestContext()

	gs := testutil.CreateTestGameserver(t, svc)

	// Manually mark as installed before starting
	fetched, _ := svc.GameserverSvc.GetGameserver(gs.ID)
	fetched.Installed = true
	models.UpdateGameserver(svc.DB, fetched)

	require.NoError(t, svc.GameserverSvc.Start(ctx, gs.ID))

	// Should still reach running via ready pattern (install marker is irrelevant)
	pollUntil(t, func() bool {
		f, _ := svc.GameserverSvc.GetGameserver(gs.ID)
		return f != nil && f.Status == "running"
	}, "should still promote to running even when already installed")

	// Verify installed flag is still true
	final, _ := svc.GameserverSvc.GetGameserver(gs.ID)
	assert.True(t, final.Installed)
}

func TestReady_BothInstallAndReadyDetectedOnFirstStart(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServicesWithSubscribers(t)
	testutil.RegisterFakeWorker(t, svc, "worker-1")
	ctx := testutil.TestContext()

	gs := testutil.CreateTestGameserver(t, svc)
	assert.False(t, gs.Installed)

	require.NoError(t, svc.GameserverSvc.Start(ctx, gs.ID))

	// Both should happen: installed=true AND status=running
	pollUntil(t, func() bool {
		f, _ := svc.GameserverSvc.GetGameserver(gs.ID)
		return f != nil && f.Installed && f.Status == "running"
	}, "both installed flag and running status should be set on first start")
}

// pollUntilReady is defined in pipeline_test.go as pollUntil — reuse it here
// by relying on it being in the same test package.

// Increase timeout for CI environments where goroutine scheduling is slower
func init() {
	// pollUntil already has a 3s timeout, which is sufficient
	_ = time.Second
}

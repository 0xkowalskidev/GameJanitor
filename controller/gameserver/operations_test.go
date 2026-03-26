package gameserver_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/warsmite/gamejanitor/model"
	"github.com/warsmite/gamejanitor/store"
	"github.com/warsmite/gamejanitor/testutil"
)

func TestOperation_StartCreatesRecord(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)
	testutil.RegisterFakeWorker(t, svc, "worker-1")
	gs := testutil.CreateTestGameserver(t, svc)

	require.NoError(t, svc.GameserverSvc.Start(testutil.TestContext(), gs.ID))

	s := store.New(svc.DB)
	ops, err := s.ListOperations(model.OperationFilter{GameserverID: &gs.ID})
	require.NoError(t, err)
	require.NotEmpty(t, ops, "start should create an operation record")

	op := ops[0]
	assert.Equal(t, model.OpStart, op.Type)
	assert.Equal(t, model.OperationStatusCompleted, op.Status)
	assert.Equal(t, "worker-1", op.WorkerID)
	assert.NotNil(t, op.CompletedAt)
}

func TestOperation_MutexRejectsConcurrent(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)
	testutil.RegisterFakeWorker(t, svc, "worker-1")
	gs := testutil.CreateTestGameserver(t, svc)

	// Insert a fake running operation
	s := store.New(svc.DB)
	require.NoError(t, s.CreateOperation(&model.Operation{
		ID:           "op-running",
		GameserverID: gs.ID,
		WorkerID:     "worker-1",
		Type:         model.OpBackup,
		Status:       model.OperationStatusRunning,
		Metadata:     []byte("{}"),
		StartedAt:    time.Now(),
	}))

	// Start should be rejected — there's already a running operation
	err := svc.GameserverSvc.Start(testutil.TestContext(), gs.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already has an operation in progress")
}

func TestOperation_StopBypassesMutex(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)
	testutil.RegisterFakeWorker(t, svc, "worker-1")
	gs := testutil.CreateTestGameserver(t, svc)

	// Start the gameserver first
	require.NoError(t, svc.GameserverSvc.Start(testutil.TestContext(), gs.ID))

	// Insert a fake running operation
	s := store.New(svc.DB)
	require.NoError(t, s.CreateOperation(&model.Operation{
		ID:           "op-running-2",
		GameserverID: gs.ID,
		WorkerID:     "worker-1",
		Type:         model.OpBackup,
		Status:       model.OperationStatusRunning,
		Metadata:     []byte("{}"),
		StartedAt:    time.Now(),
	}))

	// Stop should still work despite the running operation
	err := svc.GameserverSvc.Stop(testutil.TestContext(), gs.ID)
	require.NoError(t, err)
}

func TestOperation_RestartCreatesOneRecord(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)
	testutil.RegisterFakeWorker(t, svc, "worker-1")
	gs := testutil.CreateTestGameserver(t, svc)

	// Start first so restart has something to stop
	require.NoError(t, svc.GameserverSvc.Start(testutil.TestContext(), gs.ID))

	// Clear operations from the start
	svc.DB.Exec("DELETE FROM operations")

	require.NoError(t, svc.GameserverSvc.Restart(testutil.TestContext(), gs.ID))

	s := store.New(svc.DB)
	ops, err := s.ListOperations(model.OperationFilter{GameserverID: &gs.ID})
	require.NoError(t, err)

	// Should have exactly one operation (restart), not three (restart + stop + start)
	assert.Len(t, ops, 1, "restart should create a single operation, not nested ones")
	assert.Equal(t, model.OpRestart, ops[0].Type)
	assert.Equal(t, model.OperationStatusCompleted, ops[0].Status)
}

func TestOperation_AbandonOnStartup(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)
	testutil.RegisterFakeWorker(t, svc, "worker-1")
	gs := testutil.CreateTestGameserver(t, svc)
	s := store.New(svc.DB)

	// Insert a "running" operation as if the controller crashed
	require.NoError(t, s.CreateOperation(&model.Operation{
		ID:           "op-stale",
		GameserverID: gs.ID,
		WorkerID:     "worker-1",
		Type:         model.OpBackup,
		Status:       model.OperationStatusRunning,
		Metadata:     []byte("{}"),
		StartedAt:    time.Now(),
	}))

	abandoned, err := s.AbandonRunningOperations()
	require.NoError(t, err)
	assert.Equal(t, 1, abandoned)

	op, err := s.GetOperation("op-stale")
	require.NoError(t, err)
	assert.Equal(t, model.OperationStatusAbandoned, op.Status)
	assert.Equal(t, "controller restarted", op.Error)
	assert.NotNil(t, op.CompletedAt)
}

package store_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/warsmite/gamejanitor/model"
	"github.com/warsmite/gamejanitor/store"
	"github.com/warsmite/gamejanitor/testutil"
)

func TestWorkerNode_UpsertAndGet(t *testing.T) {
	t.Parallel()
	db := store.New(testutil.NewTestDB(t))

	node := &model.WorkerNode{
		ID:          "worker-1",
		GRPCAddress: "127.0.0.1:9090",
		LanIP:       "192.168.1.10",
		ExternalIP:  "1.2.3.4",
	}
	require.NoError(t, db.UpsertWorkerNode(node))

	got, err := db.GetWorkerNode("worker-1")
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, "worker-1", got.ID)
	assert.Equal(t, "127.0.0.1:9090", got.GRPCAddress)
	assert.Equal(t, "192.168.1.10", got.LanIP)
	assert.Equal(t, "1.2.3.4", got.ExternalIP)
	assert.Nil(t, got.MaxMemoryMB)
	assert.Nil(t, got.MaxCPU)
	assert.Nil(t, got.MaxStorageMB)
	assert.False(t, got.Cordoned)
	assert.NotNil(t, got.LastSeen)
}

func TestWorkerNode_GetNotFound(t *testing.T) {
	t.Parallel()
	db := store.New(testutil.NewTestDB(t))

	got, err := db.GetWorkerNode("nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestWorkerNode_UpsertUpdatesExisting(t *testing.T) {
	t.Parallel()
	db := store.New(testutil.NewTestDB(t))

	node := &model.WorkerNode{
		ID:          "worker-ups",
		GRPCAddress: "127.0.0.1:9090",
		LanIP:       "10.0.0.1",
		ExternalIP:  "1.1.1.1",
	}
	require.NoError(t, db.UpsertWorkerNode(node))

	node.LanIP = "10.0.0.2"
	node.ExternalIP = "2.2.2.2"
	require.NoError(t, db.UpsertWorkerNode(node))

	got, err := db.GetWorkerNode("worker-ups")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "10.0.0.2", got.LanIP)
	assert.Equal(t, "2.2.2.2", got.ExternalIP)
}

func TestWorkerNode_UpsertEmptyGRPCAddress_PreservesExisting(t *testing.T) {
	t.Parallel()
	db := store.New(testutil.NewTestDB(t))

	node := &model.WorkerNode{
		ID:          "worker-grpc",
		GRPCAddress: "127.0.0.1:9090",
		LanIP:       "10.0.0.1",
	}
	require.NoError(t, db.UpsertWorkerNode(node))

	// Upsert with empty gRPC address should preserve the original.
	node.GRPCAddress = ""
	require.NoError(t, db.UpsertWorkerNode(node))

	got, err := db.GetWorkerNode("worker-grpc")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "127.0.0.1:9090", got.GRPCAddress)
}

func TestWorkerNode_List(t *testing.T) {
	t.Parallel()
	db := store.New(testutil.NewTestDB(t))

	node1 := &model.WorkerNode{ID: "worker-a", GRPCAddress: "a:9090"}
	node2 := &model.WorkerNode{ID: "worker-b", GRPCAddress: "b:9090"}
	require.NoError(t, db.UpsertWorkerNode(node1))
	require.NoError(t, db.UpsertWorkerNode(node2))

	list, err := db.ListWorkerNodes()
	require.NoError(t, err)
	assert.Len(t, list, 2)
	// Ordered by ID
	assert.Equal(t, "worker-a", list[0].ID)
	assert.Equal(t, "worker-b", list[1].ID)
}

func TestWorkerNode_SetSFTPPort(t *testing.T) {
	t.Parallel()
	db := store.New(testutil.NewTestDB(t))

	node := &model.WorkerNode{ID: "worker-sftp", GRPCAddress: "127.0.0.1:9090"}
	require.NoError(t, db.UpsertWorkerNode(node))

	require.NoError(t, db.SetWorkerNodeSFTPPort("worker-sftp", 2222))

	got, err := db.GetWorkerNode("worker-sftp")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 2222, got.SFTPPort)
}

func TestWorkerNode_SetCordoned(t *testing.T) {
	t.Parallel()
	db := store.New(testutil.NewTestDB(t))

	node := &model.WorkerNode{ID: "worker-cord", GRPCAddress: "127.0.0.1:9090"}
	require.NoError(t, db.UpsertWorkerNode(node))

	require.NoError(t, db.SetWorkerNodeCordoned("worker-cord", true))
	got, err := db.GetWorkerNode("worker-cord")
	require.NoError(t, err)
	assert.True(t, got.Cordoned)

	require.NoError(t, db.SetWorkerNodeCordoned("worker-cord", false))
	got, err = db.GetWorkerNode("worker-cord")
	require.NoError(t, err)
	assert.False(t, got.Cordoned)
}

func TestWorkerNode_SetTags(t *testing.T) {
	t.Parallel()
	db := store.New(testutil.NewTestDB(t))

	node := &model.WorkerNode{ID: "worker-tags", GRPCAddress: "127.0.0.1:9090"}
	require.NoError(t, db.UpsertWorkerNode(node))

	require.NoError(t, db.SetWorkerNodeTags("worker-tags", model.Labels{"hardware": "gpu", "storage": "ssd"}))

	got, err := db.GetWorkerNode("worker-tags")
	require.NoError(t, err)
	assert.Equal(t, model.Labels{"hardware": "gpu", "storage": "ssd"}, got.Tags)
}

func TestWorkerNode_SetLimits(t *testing.T) {
	t.Parallel()
	db := store.New(testutil.NewTestDB(t))

	node := &model.WorkerNode{ID: "worker-lim", GRPCAddress: "127.0.0.1:9090"}
	require.NoError(t, db.UpsertWorkerNode(node))

	mem := 16384
	cpu := 8.0
	storage := 500000
	require.NoError(t, db.SetWorkerNodeLimits("worker-lim", &mem, &cpu, &storage))

	got, err := db.GetWorkerNode("worker-lim")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.MaxMemoryMB)
	require.NotNil(t, got.MaxCPU)
	require.NotNil(t, got.MaxStorageMB)
	assert.Equal(t, 16384, *got.MaxMemoryMB)
	assert.InDelta(t, 8.0, *got.MaxCPU, 0.001)
	assert.Equal(t, 500000, *got.MaxStorageMB)
}

func TestWorkerNode_SetLimits_ClearWithNil(t *testing.T) {
	t.Parallel()
	db := store.New(testutil.NewTestDB(t))

	node := &model.WorkerNode{ID: "worker-limc", GRPCAddress: "127.0.0.1:9090"}
	require.NoError(t, db.UpsertWorkerNode(node))

	mem := 1024
	require.NoError(t, db.SetWorkerNodeLimits("worker-limc", &mem, nil, nil))

	got, err := db.GetWorkerNode("worker-limc")
	require.NoError(t, err)
	require.NotNil(t, got.MaxMemoryMB)
	assert.Equal(t, 1024, *got.MaxMemoryMB)
	assert.Nil(t, got.MaxCPU)
	assert.Nil(t, got.MaxStorageMB)

	// Clear all
	require.NoError(t, db.SetWorkerNodeLimits("worker-limc", nil, nil, nil))
	got, err = db.GetWorkerNode("worker-limc")
	require.NoError(t, err)
	assert.Nil(t, got.MaxMemoryMB)
}

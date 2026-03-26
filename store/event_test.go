package store_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/warsmite/gamejanitor/model"
	"github.com/warsmite/gamejanitor/store"
	"github.com/warsmite/gamejanitor/testutil"
)

func TestEvent_CreateAndList(t *testing.T) {
	t.Parallel()
	db := store.New(testutil.NewTestDB(t))

	now := time.Now()
	e := &model.Event{
		ID:           "evt-1",
		EventType:    "gameserver.started",
		GameserverID: "gs-1",
		Actor:        json.RawMessage(`{"type":"system"}`),
		Data:         json.RawMessage(`{"name":"My Server"}`),
		CreatedAt:    now,
	}
	require.NoError(t, db.CreateEvent(e))

	list, err := db.ListEvents(model.EventFilter{})
	require.NoError(t, err)
	require.Len(t, list, 1)

	assert.Equal(t, "evt-1", list[0].ID)
	assert.Equal(t, "gameserver.started", list[0].EventType)
	assert.Equal(t, "gs-1", list[0].GameserverID)

	var actor map[string]string
	require.NoError(t, json.Unmarshal(list[0].Actor, &actor))
	assert.Equal(t, "system", actor["type"])
}

func TestEvent_ListFilterByEventType(t *testing.T) {
	t.Parallel()
	db := store.New(testutil.NewTestDB(t))

	now := time.Now()
	events := []model.Event{
		{ID: "evt-1", EventType: "gameserver.started", Actor: json.RawMessage(`{}`), Data: json.RawMessage(`{}`), CreatedAt: now},
		{ID: "evt-2", EventType: "gameserver.stopped", Actor: json.RawMessage(`{}`), Data: json.RawMessage(`{}`), CreatedAt: now},
		{ID: "evt-3", EventType: "backup.completed", Actor: json.RawMessage(`{}`), Data: json.RawMessage(`{}`), CreatedAt: now},
	}
	for i := range events {
		require.NoError(t, db.CreateEvent(&events[i]))
	}

	// GLOB pattern matching
	list, err := db.ListEvents(model.EventFilter{EventType: "gameserver.*"})
	require.NoError(t, err)
	assert.Len(t, list, 2)
	for _, e := range list {
		assert.Contains(t, e.EventType, "gameserver.")
	}
}

func TestEvent_ListFilterByGameserverID(t *testing.T) {
	t.Parallel()
	db := store.New(testutil.NewTestDB(t))

	now := time.Now()
	e1 := &model.Event{ID: "evt-1", EventType: "gameserver.started", GameserverID: "gs-1", Actor: json.RawMessage(`{}`), Data: json.RawMessage(`{}`), CreatedAt: now}
	e2 := &model.Event{ID: "evt-2", EventType: "gameserver.started", GameserverID: "gs-2", Actor: json.RawMessage(`{}`), Data: json.RawMessage(`{}`), CreatedAt: now}
	require.NoError(t, db.CreateEvent(e1))
	require.NoError(t, db.CreateEvent(e2))

	list, err := db.ListEvents(model.EventFilter{GameserverID: "gs-1"})
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, "gs-1", list[0].GameserverID)
}

func TestEvent_ListWithPagination(t *testing.T) {
	t.Parallel()
	db := store.New(testutil.NewTestDB(t))

	now := time.Now()
	for i := 0; i < 10; i++ {
		e := &model.Event{
			ID:        fmt.Sprintf("evt-%02d", i),
			EventType: "test.event",
			Actor:     json.RawMessage(`{}`),
			Data:      json.RawMessage(`{}`),
			CreatedAt: now.Add(time.Duration(i) * time.Second),
		}
		require.NoError(t, db.CreateEvent(e))
	}

	list, err := db.ListEvents(model.EventFilter{
		Pagination: model.Pagination{Limit: 3},
	})
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

func TestEvent_ListDefaultLimit(t *testing.T) {
	t.Parallel()
	db := store.New(testutil.NewTestDB(t))

	now := time.Now()
	// ListEvents sets Limit=50 if Limit<=0, then ApplyToQuery caps at maxLimit=200.
	// Creating 5 events; with default limit we should get all 5.
	for i := 0; i < 5; i++ {
		e := &model.Event{
			ID:        fmt.Sprintf("evt-%d", i),
			EventType: "test",
			Actor:     json.RawMessage(`{}`),
			Data:      json.RawMessage(`{}`),
			CreatedAt: now.Add(time.Duration(i) * time.Second),
		}
		require.NoError(t, db.CreateEvent(e))
	}

	list, err := db.ListEvents(model.EventFilter{})
	require.NoError(t, err)
	assert.Len(t, list, 5)
}

func TestEvent_PruneEvents(t *testing.T) {
	t.Parallel()
	db := store.New(testutil.NewTestDB(t))

	now := time.Now()
	// Old event (31 days ago)
	old := &model.Event{
		ID:        "evt-old",
		EventType: "test.old",
		Actor:     json.RawMessage(`{}`),
		Data:      json.RawMessage(`{}`),
		CreatedAt: now.Add(-31 * 24 * time.Hour),
	}
	require.NoError(t, db.CreateEvent(old))

	// Recent event
	recent := &model.Event{
		ID:        "evt-recent",
		EventType: "test.recent",
		Actor:     json.RawMessage(`{}`),
		Data:      json.RawMessage(`{}`),
		CreatedAt: now,
	}
	require.NoError(t, db.CreateEvent(recent))

	pruned, err := db.PruneEvents(30)
	require.NoError(t, err)
	assert.Equal(t, int64(1), pruned)

	list, err := db.ListEvents(model.EventFilter{})
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, "evt-recent", list[0].ID)
}

func TestEvent_PruneEvents_NothingToPrune(t *testing.T) {
	t.Parallel()
	db := store.New(testutil.NewTestDB(t))

	now := time.Now()
	e := &model.Event{
		ID:        "evt-fresh",
		EventType: "test",
		Actor:     json.RawMessage(`{}`),
		Data:      json.RawMessage(`{}`),
		CreatedAt: now,
	}
	require.NoError(t, db.CreateEvent(e))

	pruned, err := db.PruneEvents(30)
	require.NoError(t, err)
	assert.Equal(t, int64(0), pruned)
}

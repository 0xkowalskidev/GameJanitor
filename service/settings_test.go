package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/warsmite/gamejanitor/service"
	"github.com/warsmite/gamejanitor/testutil"
)

func TestSettings_GetDefaults(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)

	// auth_enabled defaults to false
	assert.False(t, svc.SettingsSvc.GetBool(service.SettingAuthEnabled))

	// port_range_start defaults to 27000
	assert.Equal(t, 27000, svc.SettingsSvc.GetInt(service.SettingPortRangeStart))

	// max_backups defaults to 10
	assert.Equal(t, 10, svc.SettingsSvc.GetInt(service.SettingMaxBackups))
}

func TestSettings_SetAndGet_Bool(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)

	assert.False(t, svc.SettingsSvc.GetBool(service.SettingAuthEnabled))

	err := svc.SettingsSvc.Set(service.SettingAuthEnabled, true)
	require.NoError(t, err)

	assert.True(t, svc.SettingsSvc.GetBool(service.SettingAuthEnabled))
}

func TestSettings_SetAndGet_Int(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)

	err := svc.SettingsSvc.Set(service.SettingPortRangeStart, 30000)
	require.NoError(t, err)

	assert.Equal(t, 30000, svc.SettingsSvc.GetInt(service.SettingPortRangeStart))
}

func TestSettings_SetAndGet_String(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)

	err := svc.SettingsSvc.Set(service.SettingConnectionAddress, "192.168.1.100")
	require.NoError(t, err)

	assert.Equal(t, "192.168.1.100", svc.SettingsSvc.GetString(service.SettingConnectionAddress))
}

func TestSettings_Persistence(t *testing.T) {
	t.Parallel()
	db := testutil.NewTestDB(t)
	log := testutil.TestLogger()

	// Set a value with one service instance
	svc1 := service.NewSettingsService(db, log)
	err := svc1.Set(service.SettingAuthEnabled, true)
	require.NoError(t, err)

	// Create a new service instance on the same DB — value should persist
	svc2 := service.NewSettingsService(db, log)
	assert.True(t, svc2.GetBool(service.SettingAuthEnabled))
}

func TestSettings_ApplyConfig(t *testing.T) {
	t.Parallel()
	svc := testutil.NewTestServices(t)

	svc.SettingsSvc.ApplyConfig(map[string]any{
		"auth_enabled":     true,
		"port_range_start": 30000,
	})

	assert.True(t, svc.SettingsSvc.GetBool(service.SettingAuthEnabled))
	assert.Equal(t, 30000, svc.SettingsSvc.GetInt(service.SettingPortRangeStart))
}

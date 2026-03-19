package sftp

import (
	"crypto/subtle"
	"database/sql"
	"fmt"

	"github.com/0xkowalskidev/gamejanitor/internal/models"
)

// LocalAuth validates SFTP credentials directly against the database.
// Used on standalone and controller nodes.
type LocalAuth struct {
	db *sql.DB
}

func NewLocalAuth(db *sql.DB) *LocalAuth {
	return &LocalAuth{db: db}
}

func (a *LocalAuth) ValidateLogin(username, password string) (string, string, error) {
	gs, err := models.GetGameserverBySFTPUsername(a.db, username)
	if err != nil || gs == nil {
		return "", "", fmt.Errorf("unknown sftp user %s", username)
	}
	if subtle.ConstantTimeCompare([]byte(gs.SFTPPassword), []byte(password)) != 1 {
		return "", "", fmt.Errorf("invalid credentials")
	}
	return gs.ID, gs.VolumeName, nil
}

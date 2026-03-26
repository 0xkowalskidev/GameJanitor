package sftp

import (
	"fmt"

	"github.com/warsmite/gamejanitor/model"
	"golang.org/x/crypto/bcrypt"
)

// GameserverLookup abstracts the DB query needed for SFTP auth.
type GameserverLookup interface {
	GetGameserverBySFTPUsername(username string) (*model.Gameserver, error)
}

// LocalAuth validates SFTP credentials directly against the database.
// Used on standalone and controller nodes.
type LocalAuth struct {
	store GameserverLookup
}

func NewLocalAuth(store GameserverLookup) *LocalAuth {
	return &LocalAuth{store: store}
}

func (a *LocalAuth) ValidateLogin(username, password string) (string, string, error) {
	gs, err := a.store.GetGameserverBySFTPUsername(username)
	if err != nil || gs == nil {
		return "", "", fmt.Errorf("unknown sftp user %s", username)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(gs.HashedSFTPPassword), []byte(password)); err != nil {
		return "", "", fmt.Errorf("invalid credentials")
	}
	return gs.ID, gs.VolumeName, nil
}

package security

import (
	"shopMe/internal/pkg/security/auth"
)

const Version = "1.0.0"

type User = auth.User
type AuthConfig = auth.Config
type AuthEngine = auth.Engine

func NewAuth(store auth.UserStore, cfg auth.Config) (*auth.Engine, error) {
	return auth.New(store, cfg)
}

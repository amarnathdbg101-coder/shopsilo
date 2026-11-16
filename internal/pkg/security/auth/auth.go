package auth

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Engine struct {
	store UserStore
	cfg   Config
	otp   *OTPManager
}

func New(store UserStore, cfg Config) (*Engine, error) {
	if store == nil {
		return nil, errors.New("auth: store must not be nil")
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	otpMgr := NewOTPManager(cfg.OTPTTL, cfg.OTPMaxAttempts)

	return &Engine{
		store: store,
		cfg:   cfg,
		otp:   otpMgr,
	}, nil
}

func (e *Engine) Routes() http.Handler {
	r := chi.NewRouter()

	r.Post("/register", e.HandleRegister)
	r.Post("/login", e.HandleLogin)
	r.Post("/refresh", e.HandleRefresh)
	r.Post("/forgot-password", e.HandleForgotPassword)
	r.Post("/reset-password", e.HandleResetPassword)

	r.With(e.RequireAuth).Get("/me", e.HandleMe)

	return r
}

func (e *Engine) Mount(r chi.Router) {
	r.Post("/register", e.HandleRegister)
	r.Post("/login", e.HandleLogin)
	r.Post("/refresh", e.HandleRefresh)
	r.Post("/forgot-password", e.HandleForgotPassword)
	r.Post("/reset-password", e.HandleResetPassword)
	r.With(e.RequireAuth).Get("/me", e.HandleMe)
}

func (e *Engine) Store() UserStore {
	return e.store
}

func (e *Engine) Config() Config {
	return e.cfg
}

func (e *Engine) OTP() *OTPManager {
	return e.otp
}

func (e *Engine) Close() {
	if e.otp != nil {
		e.otp.Close()
	}
}

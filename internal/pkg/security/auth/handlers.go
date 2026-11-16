package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
)

func (e *Engine) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Password = strings.TrimSpace(req.Password)

	if req.Email == "" || !isValidEmail(req.Email) {
		writeJSONError(w, http.StatusBadRequest, ErrInvalidEmail.Error())
		return
	}

	if len(req.Password) < 8 {
		writeJSONError(w, http.StatusBadRequest, ErrWeakPassword.Error())
		return
	}

	existing, err := e.store.GetByEmail(r.Context(), req.Email)
	if err == nil && existing != nil {
		writeJSONError(w, http.StatusConflict, ErrEmailAlreadyExists.Error())
		return
	}

	hash, err := HashPassword(req.Password, e.cfg.BcryptCost)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	role := strings.TrimSpace(req.Role)
	if role == "" {
		role = e.cfg.DefaultRole
	}

	newUser := &User{
		Email:        req.Email,
		PasswordHash: hash,
		Role:         role,
	}

	if err := e.store.Create(r.Context(), newUser); err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			writeJSONError(w, http.StatusConflict, ErrEmailAlreadyExists.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "Failed to create user account")
		return
	}

	authResp, err := e.GenerateTokenPair(newUser)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to generate authentication tokens")
		return
	}

	writeJSON(w, http.StatusCreated, authResp)
}

func (e *Engine) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Password = strings.TrimSpace(req.Password)

	if req.Email == "" || req.Password == "" {
		writeJSONError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	user, err := e.store.GetByEmail(r.Context(), req.Email)
	if err != nil || user == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrInvalidCredentials.Error())
		return
	}

	if !CheckPassword(req.Password, user.PasswordHash) {
		writeJSONError(w, http.StatusUnauthorized, ErrInvalidCredentials.Error())
		return
	}

	authResp, err := e.GenerateTokenPair(user)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to generate authentication tokens")
		return
	}

	writeJSON(w, http.StatusOK, authResp)
}

func (e *Engine) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	req.RefreshToken = strings.TrimSpace(req.RefreshToken)
	if req.RefreshToken == "" {
		writeJSONError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	claims, err := e.ValidateToken(req.RefreshToken, "refresh")
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}

	user, err := e.store.GetByID(r.Context(), claims.UserID)
	if err != nil || user == nil {
		writeJSONError(w, http.StatusUnauthorized, "Associated user account no longer exists")
		return
	}

	authResp, err := e.GenerateTokenPair(user)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to rotate authentication tokens")
		return
	}

	writeJSON(w, http.StatusOK, authResp)
}

func (e *Engine) HandleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || !isValidEmail(req.Email) {
		writeJSONError(w, http.StatusBadRequest, ErrInvalidEmail.Error())
		return
	}

	user, err := e.store.GetByEmail(r.Context(), req.Email)
	if err != nil || user == nil {
		writeJSON(w, http.StatusOK, MessageResponse{
			Success: true,
			Message: "If your email is registered, you will receive an OTP shortly.",
		})
		return
	}

	otp, err := e.otp.Create(req.Email)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to generate OTP")
		return
	}

	if e.cfg.OnSendOTP != nil {
		if err := e.cfg.OnSendOTP(r.Context(), req.Email, otp); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to dispatch OTP notification")
			return
		}
	}

	writeJSON(w, http.StatusOK, MessageResponse{
		Success: true,
		Message: "OTP sent successfully. Please check your email.",
	})
}

func (e *Engine) HandleResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.OTP = strings.TrimSpace(req.OTP)
	req.NewPassword = strings.TrimSpace(req.NewPassword)

	if req.Email == "" || req.OTP == "" {
		writeJSONError(w, http.StatusBadRequest, "Email and OTP are required")
		return
	}

	if len(req.NewPassword) < 8 {
		writeJSONError(w, http.StatusBadRequest, ErrWeakPassword.Error())
		return
	}

	if err := e.otp.Verify(req.Email, req.OTP); err != nil {
		if errors.Is(err, ErrOTPExpired) {
			writeJSONError(w, http.StatusBadRequest, ErrOTPExpired.Error())
			return
		}
		if errors.Is(err, ErrMaxAttemptsExceeded) {
			writeJSONError(w, http.StatusTooManyRequests, ErrMaxAttemptsExceeded.Error())
			return
		}
		writeJSONError(w, http.StatusBadRequest, ErrInvalidOTP.Error())
		return
	}

	user, err := e.store.GetByEmail(r.Context(), req.Email)
	if err != nil || user == nil {
		writeJSONError(w, http.StatusNotFound, ErrUserNotFound.Error())
		return
	}

	hash, err := HashPassword(req.NewPassword, e.cfg.BcryptCost)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to hash new password")
		return
	}

	if err := e.store.UpdatePassword(r.Context(), user.ID, hash); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	writeJSON(w, http.StatusOK, MessageResponse{
		Success: true,
		Message: "Password updated successfully. You can now login with your new password.",
	})
}

func (e *Engine) HandleMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok || userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := e.store.GetByID(r.Context(), userID)
	if err != nil || user == nil {
		writeJSONError(w, http.StatusNotFound, ErrUserNotFound.Error())
		return
	}

	writeJSON(w, http.StatusOK, user.ToResponse())
}

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

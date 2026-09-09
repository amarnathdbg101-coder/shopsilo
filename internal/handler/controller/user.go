// Package controller handler http work.
package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/repository"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type UserController struct {
	service *services.UserService
}

func NewUserController(service *services.UserService) *UserController {
	return &UserController{
		service: service,
	}
}

func (c *UserController) Register(w http.ResponseWriter, r *http.Request) {
	var input dto.UserRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := c.service.Register(r.Context(), input)
	if err != nil {
		if errors.Is(err, services.ErrEmailTaken) {
			reuse.Error(w, http.StatusConflict, err.Error())
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Created(w, "Registered successfully", res)
}

func (c *UserController) Login(w http.ResponseWriter, r *http.Request) {
	var input dto.UserLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := c.service.Login(r.Context(), input)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			reuse.Error(w, http.StatusUnauthorized, err.Error())
			return
		}
		if errors.Is(err, services.ErrAccountInactive) {
			reuse.Error(w, http.StatusForbidden, err.Error())
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Login successful", res)
}

func (c *UserController) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var input dto.ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := c.service.ForgotPassword(r.Context(), input)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, res.Message, res)
}

func (c *UserController) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var input dto.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	err := c.service.ResetPassword(r.Context(), input)
	if err != nil {
		if errors.Is(err, services.ErrInvalidResetToken) {
			reuse.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Password reset successfully. You can now log in with your new password.", nil)
}

func (c *UserController) GetProfile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	res, err := c.service.GetProfile(r.Context(), claims.UserID)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "User profile retrieved successfully", res)
}

func (c *UserController) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.UpdateUserProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := c.service.UpdateProfile(r.Context(), claims.UserID, input)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Profile updated successfully", res)
}

// AdminListUsers lists all users with role filter and search (Admin)
func (c *UserController) AdminListUsers(w http.ResponseWriter, r *http.Request) {
	role := r.URL.Query().Get("role")
	search := r.URL.Query().Get("q")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	users, total, err := c.service.ListUsersForAdmin(r.Context(), role, search, page, limit)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Admin users retrieved successfully", map[string]interface{}{
		"users": users,
		"total": total,
	})
}

// AdminUpdateUserStatus activates, suspends or changes role of a user (Admin)
func (c *UserController) AdminUpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		reuse.Error(w, http.StatusBadRequest, "user id is required")
		return
	}

	var input dto.AdminUpdateUserStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := c.service.UpdateUserStatusForAdmin(r.Context(), userID, input.IsActive, input.Role)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			reuse.Error(w, http.StatusNotFound, err.Error())
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "User updated successfully", user)
}




// Package controller handler http work.
package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"shopMe/internal/reuse"
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


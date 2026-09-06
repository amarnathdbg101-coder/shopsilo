package controller

import (
	"encoding/json"
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
	var input dto.RegisterInput
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput, "invalid input")
		return
	}
	res, err := c.service.Register(r.Context(), input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, reuse.ErrDBFailure, err.Error())
		return
	}
	reuse.Success(w, "Registerd successfully", res)
}

func (c *UserController) Login(w http.ResponseWriter, r *http.Request) {
	var input dto.LoginInput
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		reuse.Error(w,http.StatusBadRequest,reuse.ErrInvalidInput,"invalid input")
		return
	}
	res,err := c.service.Login(r.Context(),&input)
	if err != nil {
		reuse.Error(w,http.StatusInternalServerError,reuse.ErrInternal,err.Error())
		return
	}
	reuse.Success(w,"Login Successfully",res)
}

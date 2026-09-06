package services

import (
	"context"
	"errors"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"shopMe/internal/reuse"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo *repository.UserRepo
}

func NewUserService(repo *repository.UserRepo) *UserService {
	return &UserService{
		repo: repo,
	}
}

func (s *UserService) Register(ctx context.Context, input dto.RegisterInput) (*dto.RegisterResponse, error) {
	hashedPassword, err := reuse.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	user := model.User{
		Name:     input.Name,
		Email:    input.Email,
		Password: hashedPassword,
	}

	err = s.repo.Register(ctx, user)
	if err != nil {
		return nil, err
	}
	return &dto.RegisterResponse{
		Name:  input.Name,
		Email: input.Email,
	}, nil
}

func (s *UserService) Login(ctx context.Context, input *dto.LoginInput) (*dto.LoginResponse, error) {
	user, err := s.repo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password))
	if err != nil {
		return nil, errors.New("Wrong password")

	}
	token,err := reuse.GenerateJwt(user.Id,user.Email)
	if err != nil {
		return nil,errors.New("Failed Token generate")
	}

	return &dto.LoginResponse{
		Token: token,
	}, nil
}

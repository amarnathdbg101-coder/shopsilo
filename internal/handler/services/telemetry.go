package services

import (
	"context"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/repository"
)

type TelemetryService struct {
	repo *repository.TelemetryRepo
}

func NewTelemetryService(repo *repository.TelemetryRepo) *TelemetryService {
	return &TelemetryService{repo: repo}
}

func (s *TelemetryService) RecordError(ctx context.Context, req dto.TelemetryErrorRequest) error {
	return s.repo.LogError(ctx, req)
}

func (s *TelemetryService) Get24HourErrors(ctx context.Context) ([]*dto.SystemErrorRecord, error) {
	return s.repo.GetActive24HourErrors(ctx)
}

func (s *TelemetryService) ClearErrors(ctx context.Context) error {
	return s.repo.ClearAllErrors(ctx)
}

package service

import (
	"context"

	"booking-service/internal/domain"
	"booking-service/internal/repository"

	"github.com/google/uuid"
)

type RoomService struct {
	roomRepo repository.RoomRepository
}

func NewRoomService(roomRepo repository.RoomRepository) *RoomService {
	return &RoomService{
		roomRepo: roomRepo,
	}
}

func (s *RoomService) Create(ctx context.Context, name string, description *string, capacity *int) (*domain.Room, error) {
	room := &domain.Room{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		Capacity:    capacity,
	}

	if err := s.roomRepo.Create(ctx, room); err != nil {
		return nil, err
	}

	return room, nil
}

func (s *RoomService) List(ctx context.Context) ([]domain.Room, error) {
	return s.roomRepo.List(ctx)
}

func (s *RoomService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Room, error) {
	return s.roomRepo.GetByID(ctx, id)
}

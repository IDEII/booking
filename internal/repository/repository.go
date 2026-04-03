package repository

import (
	"context"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

type RoomRepository interface {
	Create(ctx context.Context, room *domain.Room) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Room, error)
	List(ctx context.Context) ([]domain.Room, error)
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type ScheduleRepository interface {
	Create(ctx context.Context, schedule *domain.Schedule) error
	GetByRoomID(ctx context.Context, roomID uuid.UUID) (*domain.Schedule, error)
	Exists(ctx context.Context, roomID uuid.UUID) (bool, error)
}

type SlotRepository interface {
	Create(ctx context.Context, slot *domain.Slot) error
	CreateBatch(ctx context.Context, slots []domain.Slot) error
	GetByRoomAndDate(ctx context.Context, roomID uuid.UUID, date time.Time) ([]domain.Slot, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Slot, error)
	GetFutureSlotsByUser(ctx context.Context, userID uuid.UUID) ([]domain.Slot, error)
}

type BookingRepository interface {
	Create(ctx context.Context, booking *domain.Booking) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error)
	GetBySlotID(ctx context.Context, slotID uuid.UUID) (*domain.Booking, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.BookingStatus) error
	List(ctx context.Context, page, pageSize int) ([]domain.Booking, int, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Booking, error)
}

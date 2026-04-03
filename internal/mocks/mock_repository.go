package mocks

import (
	"context"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

type MockRoomRepository struct {
	mock.Mock
}

func (m *MockRoomRepository) Create(ctx context.Context, room *domain.Room) error {
	args := m.Called(ctx, room)
	return args.Error(0)
}

func (m *MockRoomRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Room, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Room), args.Error(1)
}

func (m *MockRoomRepository) List(ctx context.Context) ([]domain.Room, error) {
	args := m.Called(ctx)
	return args.Get(0).([]domain.Room), args.Error(1)
}

func (m *MockRoomRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

type MockScheduleRepository struct {
	mock.Mock
}

func (m *MockScheduleRepository) Create(ctx context.Context, schedule *domain.Schedule) error {
	args := m.Called(ctx, schedule)
	return args.Error(0)
}

func (m *MockScheduleRepository) GetByRoomID(ctx context.Context, roomID uuid.UUID) (*domain.Schedule, error) {
	args := m.Called(ctx, roomID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Schedule), args.Error(1)
}

func (m *MockScheduleRepository) Exists(ctx context.Context, roomID uuid.UUID) (bool, error) {
	args := m.Called(ctx, roomID)
	return args.Bool(0), args.Error(1)
}

type MockSlotRepository struct {
	mock.Mock
}

func (m *MockSlotRepository) Create(ctx context.Context, slot *domain.Slot) error {
	args := m.Called(ctx, slot)
	return args.Error(0)
}

func (m *MockSlotRepository) CreateBatch(ctx context.Context, slots []domain.Slot) error {
	args := m.Called(ctx, slots)
	return args.Error(0)
}

func (m *MockSlotRepository) GetByRoomAndDate(ctx context.Context, roomID uuid.UUID, date time.Time) ([]domain.Slot, error) {
	args := m.Called(ctx, roomID, date)
	return args.Get(0).([]domain.Slot), args.Error(1)
}

func (m *MockSlotRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Slot, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Slot), args.Error(1)
}

func (m *MockSlotRepository) GetFutureSlotsByUser(ctx context.Context, userID uuid.UUID) ([]domain.Slot, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]domain.Slot), args.Error(1)
}

type MockBookingRepository struct {
	mock.Mock
}

func (m *MockBookingRepository) Create(ctx context.Context, booking *domain.Booking) error {
	args := m.Called(ctx, booking)
	return args.Error(0)
}

func (m *MockBookingRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Booking), args.Error(1)
}

func (m *MockBookingRepository) GetBySlotID(ctx context.Context, slotID uuid.UUID) (*domain.Booking, error) {
	args := m.Called(ctx, slotID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Booking), args.Error(1)
}

func (m *MockBookingRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.BookingStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockBookingRepository) List(ctx context.Context, page, pageSize int) ([]domain.Booking, int, error) {
	args := m.Called(ctx, page, pageSize)
	return args.Get(0).([]domain.Booking), args.Int(1), args.Error(2)
}

func (m *MockBookingRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Booking, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]domain.Booking), args.Error(1)
}

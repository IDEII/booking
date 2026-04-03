package service

import (
	"booking-service/internal/domain"
	"booking-service/internal/mocks"

	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRoomService_Create(t *testing.T) {
	tests := []struct {
		name        string
		roomName    string
		description *string
		capacity    *int
		setupMock   func(*mocks.MockRoomRepository)
		expectedErr bool
	}{
		{
			name:        "successful room creation with all fields",
			roomName:    "Conference Room A",
			description: stringPtr("Large conference room with projector"),
			capacity:    intPtr(20),
			setupMock: func(roomRepo *mocks.MockRoomRepository) {
				roomRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Room")).Return(nil)
			},
			expectedErr: false,
		},
		{
			name:        "successful room creation with minimal fields",
			roomName:    "Small Meeting Room",
			description: nil,
			capacity:    nil,
			setupMock: func(roomRepo *mocks.MockRoomRepository) {
				roomRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Room")).Return(nil)
			},
			expectedErr: false,
		},
		{
			name:        "room creation with empty name",
			roomName:    "",
			description: nil,
			capacity:    nil,
			setupMock: func(roomRepo *mocks.MockRoomRepository) {
				roomRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Room")).Return(nil)
			},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRoomRepo := new(mocks.MockRoomRepository)
			tt.setupMock(mockRoomRepo)

			roomService := NewRoomService(mockRoomRepo)

			room, err := roomService.Create(context.Background(), tt.roomName, tt.description, tt.capacity)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, room)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, room)
				assert.Equal(t, tt.roomName, room.Name)
				assert.Equal(t, tt.description, room.Description)
				assert.Equal(t, tt.capacity, room.Capacity)
				assert.NotEmpty(t, room.ID)
			}

			mockRoomRepo.AssertExpectations(t)
		})
	}
}

func TestRoomService_List(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*mocks.MockRoomRepository)
		expectedLen int
		expectedErr bool
	}{
		{
			name: "successful list with multiple rooms",
			setupMock: func(roomRepo *mocks.MockRoomRepository) {
				rooms := []domain.Room{
					{ID: uuid.New(), Name: "Room 1", Capacity: intPtr(10)},
					{ID: uuid.New(), Name: "Room 2", Capacity: intPtr(20)},
					{ID: uuid.New(), Name: "Room 3", Capacity: intPtr(15)},
				}
				roomRepo.On("List", mock.Anything).Return(rooms, nil)
			},
			expectedLen: 3,
			expectedErr: false,
		},
		{
			name: "empty list",
			setupMock: func(roomRepo *mocks.MockRoomRepository) {
				roomRepo.On("List", mock.Anything).Return([]domain.Room{}, nil)
			},
			expectedLen: 0,
			expectedErr: false,
		},
		{
			name: "repository error",
			setupMock: func(roomRepo *mocks.MockRoomRepository) {
				roomRepo.On("List", mock.Anything).Return([]domain.Room{}, assert.AnError)
			},
			expectedLen: 0,
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRoomRepo := new(mocks.MockRoomRepository)
			tt.setupMock(mockRoomRepo)

			roomService := NewRoomService(mockRoomRepo)

			rooms, err := roomService.List(context.Background())

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, rooms, tt.expectedLen)
			}

			mockRoomRepo.AssertExpectations(t)
		})
	}
}

func TestRoomService_GetByID(t *testing.T) {
	roomID := uuid.New()
	expectedRoom := &domain.Room{
		ID:          roomID,
		Name:        "Test Room",
		Description: stringPtr("Test description"),
		Capacity:    intPtr(10),
	}

	tests := []struct {
		name        string
		roomID      uuid.UUID
		setupMock   func(*mocks.MockRoomRepository)
		expected    *domain.Room
		expectedErr bool
	}{
		{
			name:   "successful get by id",
			roomID: roomID,
			setupMock: func(roomRepo *mocks.MockRoomRepository) {
				roomRepo.On("GetByID", mock.Anything, roomID).Return(expectedRoom, nil)
			},
			expected:    expectedRoom,
			expectedErr: false,
		},
		{
			name:   "room not found",
			roomID: uuid.New(),
			setupMock: func(roomRepo *mocks.MockRoomRepository) {
				roomRepo.On("GetByID", mock.Anything, mock.Anything).Return(nil, nil)
			},
			expected:    nil,
			expectedErr: false,
		},
		{
			name:   "repository error",
			roomID: roomID,
			setupMock: func(roomRepo *mocks.MockRoomRepository) {
				roomRepo.On("GetByID", mock.Anything, roomID).Return(nil, assert.AnError)
			},
			expected:    nil,
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRoomRepo := new(mocks.MockRoomRepository)
			tt.setupMock(mockRoomRepo)

			roomService := NewRoomService(mockRoomRepo)

			room, err := roomService.GetByID(context.Background(), tt.roomID)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, room)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, room)
			}

			mockRoomRepo.AssertExpectations(t)
		})
	}
}

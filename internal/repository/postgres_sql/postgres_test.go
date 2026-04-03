package postgres_sql

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type PostgresTestSuite struct {
	suite.Suite
	db         *sql.DB
	ctx        context.Context
	testDBName string
	cleanup    []func()
	testUserID uuid.UUID
	testRoomID uuid.UUID
}

func (s *PostgresTestSuite) SetupTest() {
	err := TruncateTables(s.db)
	require.NoError(s.T(), err)

	s.testUserID = s.createTestUser()
	s.testRoomID = s.createTestRoom()
}

func (s *PostgresTestSuite) TearDownSuite() {
	for i := len(s.cleanup) - 1; i >= 0; i-- {
		s.cleanup[i]()
	}

	TeardownTestDatabase(s.T(), s.db, s.testDBName)
}

func (s *PostgresTestSuite) createTestUser() uuid.UUID {
	userID := uuid.New()
	query := `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := s.db.ExecContext(s.ctx, query,
		userID,
		fmt.Sprintf("test_%s@example.com", userID.String()[:8]),
		"hashed_password",
		domain.RoleUser,
		time.Now(),
	)
	require.NoError(s.T(), err)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM users WHERE id = $1", userID)
	})

	return userID
}

func (s *PostgresTestSuite) createTestRoom() uuid.UUID {
	roomID := uuid.New()
	query := `
		INSERT INTO rooms (id, name, description, capacity, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := s.db.ExecContext(s.ctx, query,
		roomID,
		"Test Room",
		sql.NullString{String: "Test description", Valid: true},
		sql.NullInt64{Int64: 10, Valid: true},
		time.Now(),
	)
	require.NoError(s.T(), err)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM rooms WHERE id = $1", roomID)
	})

	return roomID
}

func (s *PostgresTestSuite) createTestSchedule(roomID uuid.UUID) *domain.Schedule {
	schedule := &domain.Schedule{
		ID:         uuid.New(),
		RoomID:     roomID,
		DaysOfWeek: []int{1, 2, 3, 4, 5},
		StartTime:  "09:00",
		EndTime:    "18:00",
		CreatedAt:  timePtr(time.Now()),
	}

	repo := NewScheduleRepository(s.db)
	err := repo.Create(s.ctx, schedule)
	require.NoError(s.T(), err)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM schedules WHERE id = $1", schedule.ID)
	})

	return schedule
}

func (s *PostgresTestSuite) createTestSlot(roomID uuid.UUID, start, end time.Time) *domain.Slot {
	slot := &domain.Slot{
		ID:     uuid.New(),
		RoomID: roomID,
		Start:  start,
		End:    end,
	}

	repo := NewSlotRepository(s.db)
	err := repo.Create(s.ctx, slot)
	require.NoError(s.T(), err)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM slots WHERE id = $1", slot.ID)
	})

	return slot
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func (s *PostgresTestSuite) TestUserRepository_Create() {
	repo := NewUserRepository(s.db)

	user := &domain.User{
		ID:           uuid.New(),
		Email:        fmt.Sprintf("create_%s@example.com", uuid.New().String()[:8]),
		PasswordHash: "hashed_password",
		Role:         domain.RoleUser,
		CreatedAt:    time.Now(),
	}

	err := repo.Create(s.ctx, user)
	assert.NoError(s.T(), err)

	var count int
	query := "SELECT COUNT(*) FROM users WHERE id = $1"
	err = s.db.QueryRowContext(s.ctx, query, user.ID).Scan(&count)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 1, count)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM users WHERE id = $1", user.ID)
	})
}

func (s *PostgresTestSuite) TestUserRepository_GetByEmail() {
	repo := NewUserRepository(s.db)

	email := fmt.Sprintf("getbyemail_%s@example.com", uuid.New().String()[:8])
	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: "hashed_password",
		Role:         domain.RoleUser,
		CreatedAt:    time.Now(),
	}

	err := repo.Create(s.ctx, user)
	require.NoError(s.T(), err)

	found, err := repo.GetByEmail(s.ctx, email)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), found)
	assert.Equal(s.T(), user.ID, found.ID)
	assert.Equal(s.T(), user.Email, found.Email)
	assert.Equal(s.T(), user.Role, found.Role)

	notFound, err := repo.GetByEmail(s.ctx, "nonexistent@example.com")
	assert.Error(s.T(), err)
	assert.Nil(s.T(), notFound)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM users WHERE id = $1", user.ID)
	})
}

func (s *PostgresTestSuite) TestUserRepository_GetByID() {
	repo := NewUserRepository(s.db)

	user := &domain.User{
		ID:           uuid.New(),
		Email:        fmt.Sprintf("getbyid_%s@example.com", uuid.New().String()[:8]),
		PasswordHash: "hashed_password",
		Role:         domain.RoleUser,
		CreatedAt:    time.Now(),
	}

	err := repo.Create(s.ctx, user)
	require.NoError(s.T(), err)

	found, err := repo.GetByID(s.ctx, user.ID)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), found)
	assert.Equal(s.T(), user.ID, found.ID)
	assert.Equal(s.T(), user.Email, found.Email)

	notFound, err := repo.GetByID(s.ctx, uuid.New())
	assert.Error(s.T(), err)
	assert.Nil(s.T(), notFound)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM users WHERE id = $1", user.ID)
	})
}

func (s *PostgresTestSuite) TestRoomRepository_Create() {
	repo := NewRoomRepository(s.db)

	room := &domain.Room{
		ID:          uuid.New(),
		Name:        "Conference Room A",
		Description: stringPtr("Large conference room"),
		Capacity:    intPtr(20),
		CreatedAt:   timePtr(time.Now()),
	}

	err := repo.Create(s.ctx, room)
	assert.NoError(s.T(), err)

	var count int
	query := "SELECT COUNT(*) FROM rooms WHERE id = $1"
	err = s.db.QueryRowContext(s.ctx, query, room.ID).Scan(&count)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 1, count)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM rooms WHERE id = $1", room.ID)
	})
}

func (s *PostgresTestSuite) TestRoomRepository_GetByID() {
	repo := NewRoomRepository(s.db)

	room := &domain.Room{
		ID:          uuid.New(),
		Name:        "Test Room",
		Description: stringPtr("Test description"),
		Capacity:    intPtr(15),
		CreatedAt:   timePtr(time.Now()),
	}

	err := repo.Create(s.ctx, room)
	require.NoError(s.T(), err)

	found, err := repo.GetByID(s.ctx, room.ID)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), found)
	assert.Equal(s.T(), room.ID, found.ID)
	assert.Equal(s.T(), room.Name, found.Name)
	assert.Equal(s.T(), *room.Description, *found.Description)
	assert.Equal(s.T(), *room.Capacity, *found.Capacity)

	notFound, err := repo.GetByID(s.ctx, uuid.New())
	assert.Error(s.T(), err)
	assert.Nil(s.T(), notFound)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM rooms WHERE id = $1", room.ID)
	})
}

func (s *PostgresTestSuite) TestRoomRepository_List() {
	repo := NewRoomRepository(s.db)

	rooms := make([]*domain.Room, 3)
	for i := 0; i < 3; i++ {
		room := &domain.Room{
			ID:        uuid.New(),
			Name:      fmt.Sprintf("Room %d", i+1),
			CreatedAt: timePtr(time.Now()),
		}
		err := repo.Create(s.ctx, room)
		require.NoError(s.T(), err)
		rooms[i] = room

		s.cleanup = append(s.cleanup, func() {
			s.db.ExecContext(context.Background(), "DELETE FROM rooms WHERE id = $1", room.ID)
		})
	}

	list, err := repo.List(s.ctx)
	assert.NoError(s.T(), err)
	assert.GreaterOrEqual(s.T(), len(list), 3)
}

func (s *PostgresTestSuite) TestRoomRepository_Exists() {
	repo := NewRoomRepository(s.db)

	room := &domain.Room{
		ID:        uuid.New(),
		Name:      "Exists Test Room",
		CreatedAt: timePtr(time.Now()),
	}

	err := repo.Create(s.ctx, room)
	require.NoError(s.T(), err)

	exists, err := repo.Exists(s.ctx, room.ID)
	assert.NoError(s.T(), err)
	assert.True(s.T(), exists)

	exists, err = repo.Exists(s.ctx, uuid.New())
	assert.NoError(s.T(), err)
	assert.False(s.T(), exists)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM rooms WHERE id = $1", room.ID)
	})
}

func (s *PostgresTestSuite) TestScheduleRepository_Create() {
	repo := NewScheduleRepository(s.db)

	schedule := &domain.Schedule{
		ID:         uuid.New(),
		RoomID:     s.testRoomID,
		DaysOfWeek: []int{1, 2, 3, 4, 5},
		StartTime:  "09:00",
		EndTime:    "18:00",
		CreatedAt:  timePtr(time.Now()),
	}

	err := repo.Create(s.ctx, schedule)
	assert.NoError(s.T(), err)

	var count int
	query := "SELECT COUNT(*) FROM schedules WHERE id = $1"
	err = s.db.QueryRowContext(s.ctx, query, schedule.ID).Scan(&count)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 1, count)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM schedules WHERE id = $1", schedule.ID)
	})
}

func (s *PostgresTestSuite) TestScheduleRepository_GetByRoomID() {
	repo := NewScheduleRepository(s.db)

	schedule := s.createTestSchedule(s.testRoomID)

	found, err := repo.GetByRoomID(s.ctx, s.testRoomID)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), found)
	assert.Equal(s.T(), schedule.ID, found.ID)
	assert.Equal(s.T(), schedule.RoomID, found.RoomID)
	assert.Equal(s.T(), schedule.DaysOfWeek, found.DaysOfWeek)
	assert.Equal(s.T(), schedule.StartTime, found.StartTime)
	assert.Equal(s.T(), schedule.EndTime, found.EndTime)

	notFound, err := repo.GetByRoomID(s.ctx, uuid.New())
	assert.NoError(s.T(), err)
	assert.Nil(s.T(), notFound)
}

func (s *PostgresTestSuite) TestScheduleRepository_Exists() {
	repo := NewScheduleRepository(s.db)

	_ = s.createTestSchedule(s.testRoomID)

	exists, err := repo.Exists(s.ctx, s.testRoomID)
	assert.NoError(s.T(), err)
	assert.True(s.T(), exists)

	exists, err = repo.Exists(s.ctx, uuid.New())
	assert.NoError(s.T(), err)
	assert.False(s.T(), exists)
}

func (s *PostgresTestSuite) TestScheduleRepository_GetByRoomID_WithWeekendDays() {
	repo := NewScheduleRepository(s.db)

	schedule := &domain.Schedule{
		ID:         uuid.New(),
		RoomID:     s.testRoomID,
		DaysOfWeek: []int{6, 7},
		StartTime:  "10:00",
		EndTime:    "15:00",
		CreatedAt:  timePtr(time.Now()),
	}

	err := repo.Create(s.ctx, schedule)
	require.NoError(s.T(), err)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM schedules WHERE id = $1", schedule.ID)
	})

	found, err := repo.GetByRoomID(s.ctx, s.testRoomID)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), found)
	assert.Equal(s.T(), []int{6, 7}, found.DaysOfWeek)
	assert.Equal(s.T(), "10:00", found.StartTime)
	assert.Equal(s.T(), "15:00", found.EndTime)
}

func (s *PostgresTestSuite) TestSlotRepository_Create() {
	repo := NewSlotRepository(s.db)

	now := time.Now().UTC()
	slot := &domain.Slot{
		ID:     uuid.New(),
		RoomID: s.testRoomID,
		Start:  now.Add(24 * time.Hour),
		End:    now.Add(24*time.Hour + 30*time.Minute),
	}

	err := repo.Create(s.ctx, slot)
	assert.NoError(s.T(), err)

	var count int
	query := "SELECT COUNT(*) FROM slots WHERE id = $1"
	err = s.db.QueryRowContext(s.ctx, query, slot.ID).Scan(&count)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 1, count)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM slots WHERE id = $1", slot.ID)
	})
}

func (s *PostgresTestSuite) TestSlotRepository_CreateBatch() {
	repo := NewSlotRepository(s.db)

	now := time.Now().UTC()
	slots := make([]domain.Slot, 5)
	for i := 0; i < 5; i++ {
		slots[i] = domain.Slot{
			ID:     uuid.New(),
			RoomID: s.testRoomID,
			Start:  now.Add(time.Duration(i+1) * 24 * time.Hour),
			End:    now.Add(time.Duration(i+1)*24*time.Hour + 30*time.Minute),
		}
	}

	err := repo.CreateBatch(s.ctx, slots)
	assert.NoError(s.T(), err)

	for _, slot := range slots {
		var count int
		query := "SELECT COUNT(*) FROM slots WHERE id = $1"
		err = s.db.QueryRowContext(s.ctx, query, slot.ID).Scan(&count)
		assert.NoError(s.T(), err)
		assert.Equal(s.T(), 1, count)
	}

	for _, slot := range slots {
		s.cleanup = append(s.cleanup, func() {
			s.db.ExecContext(context.Background(), "DELETE FROM slots WHERE id = $1", slot.ID)
		})
	}
}

func (s *PostgresTestSuite) TestSlotRepository_CreateBatch_WithDuplicate() {
	repo := NewSlotRepository(s.db)

	now := time.Now().UTC()
	slotID := uuid.New()
	slot := domain.Slot{
		ID:     slotID,
		RoomID: s.testRoomID,
		Start:  now.Add(24 * time.Hour),
		End:    now.Add(24*time.Hour + 30*time.Minute),
	}

	err := repo.Create(s.ctx, &slot)
	require.NoError(s.T(), err)

	slots := []domain.Slot{slot}
	err = repo.CreateBatch(s.ctx, slots)
	assert.NoError(s.T(), err)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM slots WHERE id = $1", slotID)
	})
}

func (s *PostgresTestSuite) TestSlotRepository_GetByRoomAndDate() {
	repo := NewSlotRepository(s.db)

	now := time.Now().UTC()
	testDate := now.Add(24 * time.Hour).Truncate(24 * time.Hour)

	slots := make([]*domain.Slot, 3)
	for i := 0; i < 3; i++ {
		slot := s.createTestSlot(s.testRoomID,
			testDate.Add(time.Duration(i)*30*time.Minute),
			testDate.Add(time.Duration(i+1)*30*time.Minute),
		)
		slots[i] = slot
	}

	found, err := repo.GetByRoomAndDate(s.ctx, s.testRoomID, testDate)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), found, 3)

	for i, slot := range found {
		assert.Equal(s.T(), s.testRoomID, slot.RoomID)
		assert.Equal(s.T(), slots[i].Start, slot.Start)
		assert.Equal(s.T(), slots[i].End, slot.End)
	}

	emptyDate := now.Add(100 * 24 * time.Hour)
	empty, err := repo.GetByRoomAndDate(s.ctx, s.testRoomID, emptyDate)
	assert.NoError(s.T(), err)
	assert.Empty(s.T(), empty)
}

func (s *PostgresTestSuite) TestSlotRepository_GetByID() {
	repo := NewSlotRepository(s.db)

	now := time.Now().UTC()
	slot := s.createTestSlot(s.testRoomID,
		now.Add(24*time.Hour),
		now.Add(24*time.Hour+30*time.Minute),
	)

	found, err := repo.GetByID(s.ctx, slot.ID)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), found)
	assert.Equal(s.T(), slot.ID, found.ID)
	assert.Equal(s.T(), slot.RoomID, found.RoomID)
	assert.Equal(s.T(), slot.Start.Unix(), found.Start.Unix())
	assert.Equal(s.T(), slot.End.Unix(), found.End.Unix())

	notFound, err := repo.GetByID(s.ctx, uuid.New())
	assert.Error(s.T(), err)
	assert.Nil(s.T(), notFound)
}

func (s *PostgresTestSuite) TestSlotRepository_GetFutureSlotsByUser() {
	repo := NewSlotRepository(s.db)
	bookingRepo := NewBookingRepository(s.db)

	now := time.Now().UTC()

	futureSlot := s.createTestSlot(s.testRoomID,
		now.Add(24*time.Hour),
		now.Add(24*time.Hour+30*time.Minute),
	)

	pastSlot := s.createTestSlot(s.testRoomID,
		now.Add(-48*time.Hour),
		now.Add(-48*time.Hour+30*time.Minute),
	)

	booking1 := &domain.Booking{
		ID:        uuid.New(),
		SlotID:    futureSlot.ID,
		UserID:    s.testUserID,
		Status:    domain.BookingStatusActive,
		CreatedAt: timePtr(now),
	}
	err := bookingRepo.Create(s.ctx, booking1)
	require.NoError(s.T(), err)

	booking2 := &domain.Booking{
		ID:        uuid.New(),
		SlotID:    pastSlot.ID,
		UserID:    s.testUserID,
		Status:    domain.BookingStatusActive,
		CreatedAt: timePtr(now),
	}
	err = bookingRepo.Create(s.ctx, booking2)
	require.NoError(s.T(), err)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM bookings WHERE id = $1", booking1.ID)
		s.db.ExecContext(context.Background(), "DELETE FROM bookings WHERE id = $1", booking2.ID)
	})

	slots, err := repo.GetFutureSlotsByUser(s.ctx, s.testUserID)
	assert.NoError(s.T(), err)

	assert.Len(s.T(), slots, 1)
	assert.Equal(s.T(), futureSlot.ID, slots[0].ID)
	assert.Equal(s.T(), s.testRoomID, slots[0].RoomID)
}

func (s *PostgresTestSuite) TestBookingRepository_Create() {
	repo := NewBookingRepository(s.db)
	slotRepo := NewSlotRepository(s.db)

	now := time.Now().UTC()
	slot := &domain.Slot{
		ID:     uuid.New(),
		RoomID: s.testRoomID,
		Start:  now.Add(24 * time.Hour),
		End:    now.Add(24*time.Hour + 30*time.Minute),
	}
	err := slotRepo.Create(s.ctx, slot)
	require.NoError(s.T(), err)

	booking := &domain.Booking{
		ID:        uuid.New(),
		SlotID:    slot.ID,
		UserID:    s.testUserID,
		Status:    domain.BookingStatusActive,
		CreatedAt: timePtr(time.Now()),
	}

	err = repo.Create(s.ctx, booking)
	assert.NoError(s.T(), err)

	var count int
	query := "SELECT COUNT(*) FROM bookings WHERE id = $1"
	err = s.db.QueryRowContext(s.ctx, query, booking.ID).Scan(&count)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 1, count)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM bookings WHERE id = $1", booking.ID)
	})
}

func (s *PostgresTestSuite) TestBookingRepository_Create_WithConferenceLink() {
	repo := NewBookingRepository(s.db)
	slotRepo := NewSlotRepository(s.db)

	now := time.Now().UTC()
	slot := &domain.Slot{
		ID:     uuid.New(),
		RoomID: s.testRoomID,
		Start:  now.Add(24 * time.Hour),
		End:    now.Add(24*time.Hour + 30*time.Minute),
	}
	err := slotRepo.Create(s.ctx, slot)
	require.NoError(s.T(), err)

	conferenceLink := "https://meet.example.com/test-conference"
	booking := &domain.Booking{
		ID:             uuid.New(),
		SlotID:         slot.ID,
		UserID:         s.testUserID,
		Status:         domain.BookingStatusActive,
		ConferenceLink: &conferenceLink,
		CreatedAt:      timePtr(time.Now()),
	}

	err = repo.Create(s.ctx, booking)
	assert.NoError(s.T(), err)

	var savedLink sql.NullString
	query := "SELECT conference_link FROM bookings WHERE id = $1"
	err = s.db.QueryRowContext(s.ctx, query, booking.ID).Scan(&savedLink)
	assert.NoError(s.T(), err)
	assert.True(s.T(), savedLink.Valid)
	assert.Equal(s.T(), conferenceLink, savedLink.String)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM bookings WHERE id = $1", booking.ID)
	})
}

func (s *PostgresTestSuite) TestBookingRepository_GetByID() {
	repo := NewBookingRepository(s.db)
	slotRepo := NewSlotRepository(s.db)

	now := time.Now().UTC()
	slot := &domain.Slot{
		ID:     uuid.New(),
		RoomID: s.testRoomID,
		Start:  now.Add(24 * time.Hour),
		End:    now.Add(24*time.Hour + 30*time.Minute),
	}
	err := slotRepo.Create(s.ctx, slot)
	require.NoError(s.T(), err)

	booking := &domain.Booking{
		ID:        uuid.New(),
		SlotID:    slot.ID,
		UserID:    s.testUserID,
		Status:    domain.BookingStatusActive,
		CreatedAt: timePtr(time.Now()),
	}

	err = repo.Create(s.ctx, booking)
	require.NoError(s.T(), err)

	found, err := repo.GetByID(s.ctx, booking.ID)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), found)
	assert.Equal(s.T(), booking.ID, found.ID)
	assert.Equal(s.T(), booking.SlotID, found.SlotID)
	assert.Equal(s.T(), booking.UserID, found.UserID)
	assert.Equal(s.T(), booking.Status, found.Status)

	notFound, err := repo.GetByID(s.ctx, uuid.New())
	assert.Error(s.T(), err)
	assert.Nil(s.T(), notFound)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM bookings WHERE id = $1", booking.ID)
	})
}
func (s *PostgresTestSuite) TestBookingRepository_GetBySlotID() {
	repo := NewBookingRepository(s.db)
	slotRepo := NewSlotRepository(s.db)

	now := time.Now().UTC()
	slot := &domain.Slot{
		ID:     uuid.New(),
		RoomID: s.testRoomID,
		Start:  now.Add(24 * time.Hour),
		End:    now.Add(24*time.Hour + 30*time.Minute),
	}
	err := slotRepo.Create(s.ctx, slot)
	require.NoError(s.T(), err)

	booking := &domain.Booking{
		ID:        uuid.New(),
		SlotID:    slot.ID,
		UserID:    s.testUserID,
		Status:    domain.BookingStatusActive,
		CreatedAt: timePtr(time.Now()),
	}

	err = repo.Create(s.ctx, booking)
	require.NoError(s.T(), err)

	found, err := repo.GetBySlotID(s.ctx, slot.ID)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), found)
	assert.Equal(s.T(), booking.ID, found.ID)
	assert.Equal(s.T(), slot.ID, found.SlotID)

	cancelledSlot := &domain.Slot{
		ID:     uuid.New(),
		RoomID: s.testRoomID,
		Start:  now.Add(48 * time.Hour),
		End:    now.Add(48*time.Hour + 30*time.Minute),
	}
	err = slotRepo.Create(s.ctx, cancelledSlot)
	require.NoError(s.T(), err)

	cancelledBooking := &domain.Booking{
		ID:        uuid.New(),
		SlotID:    cancelledSlot.ID,
		UserID:    s.testUserID,
		Status:    domain.BookingStatusCancelled,
		CreatedAt: timePtr(time.Now()),
	}
	err = repo.Create(s.ctx, cancelledBooking)
	require.NoError(s.T(), err)

	found, err = repo.GetBySlotID(s.ctx, cancelledSlot.ID)
	assert.NoError(s.T(), err)
	assert.Nil(s.T(), found)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM bookings WHERE id = $1", booking.ID)
		s.db.ExecContext(context.Background(), "DELETE FROM bookings WHERE id = $1", cancelledBooking.ID)
	})
}

func (s *PostgresTestSuite) TestBookingRepository_UpdateStatus() {
	repo := NewBookingRepository(s.db)
	slotRepo := NewSlotRepository(s.db)

	now := time.Now().UTC()
	slot := &domain.Slot{
		ID:     uuid.New(),
		RoomID: s.testRoomID,
		Start:  now.Add(24 * time.Hour),
		End:    now.Add(24*time.Hour + 30*time.Minute),
	}
	err := slotRepo.Create(s.ctx, slot)
	require.NoError(s.T(), err)

	booking := &domain.Booking{
		ID:        uuid.New(),
		SlotID:    slot.ID,
		UserID:    s.testUserID,
		Status:    domain.BookingStatusActive,
		CreatedAt: timePtr(time.Now()),
	}

	err = repo.Create(s.ctx, booking)
	require.NoError(s.T(), err)

	err = repo.UpdateStatus(s.ctx, booking.ID, domain.BookingStatusCancelled)
	assert.NoError(s.T(), err)

	found, err := repo.GetByID(s.ctx, booking.ID)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), domain.BookingStatusCancelled, found.Status)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM bookings WHERE id = $1", booking.ID)
	})
}

func (s *PostgresTestSuite) TestBookingRepository_List() {
	repo := NewBookingRepository(s.db)
	slotRepo := NewSlotRepository(s.db)

	now := time.Now().UTC()
	bookings := make([]*domain.Booking, 5)

	for i := 0; i < 5; i++ {
		slot := &domain.Slot{
			ID:     uuid.New(),
			RoomID: s.testRoomID,
			Start:  now.Add(time.Duration(i+1) * 24 * time.Hour),
			End:    now.Add(time.Duration(i+1)*24*time.Hour + 30*time.Minute),
		}
		err := slotRepo.Create(s.ctx, slot)
		require.NoError(s.T(), err)

		booking := &domain.Booking{
			ID:        uuid.New(),
			SlotID:    slot.ID,
			UserID:    s.testUserID,
			Status:    domain.BookingStatusActive,
			CreatedAt: timePtr(time.Now().Add(-time.Duration(i) * time.Hour)),
		}
		err = repo.Create(s.ctx, booking)
		require.NoError(s.T(), err)
		bookings[i] = booking

		s.cleanup = append(s.cleanup, func() {
			s.db.ExecContext(context.Background(), "DELETE FROM bookings WHERE id = $1", booking.ID)
		})
	}

	list, total, err := repo.List(s.ctx, 1, 3)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), list, 3)
	assert.Equal(s.T(), 5, total)

	list2, total2, err := repo.List(s.ctx, 2, 3)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), list2, 2)
	assert.Equal(s.T(), 5, total2)
}

func (s *PostgresTestSuite) TestBookingRepository_GetByUserID() {
	repo := NewBookingRepository(s.db)
	slotRepo := NewSlotRepository(s.db)

	now := time.Now().UTC()
	bookings := make([]*domain.Booking, 3)

	for i := 0; i < 3; i++ {
		slot := &domain.Slot{
			ID:     uuid.New(),
			RoomID: s.testRoomID,
			Start:  now.Add(time.Duration(i+1) * 24 * time.Hour),
			End:    now.Add(time.Duration(i+1)*24*time.Hour + 30*time.Minute),
		}
		err := slotRepo.Create(s.ctx, slot)
		require.NoError(s.T(), err)

		booking := &domain.Booking{
			ID:        uuid.New(),
			SlotID:    slot.ID,
			UserID:    s.testUserID,
			Status:    domain.BookingStatusActive,
			CreatedAt: timePtr(time.Now()),
		}
		err = repo.Create(s.ctx, booking)
		require.NoError(s.T(), err)
		bookings[i] = booking

		s.cleanup = append(s.cleanup, func() {
			s.db.ExecContext(context.Background(), "DELETE FROM bookings WHERE id = $1", booking.ID)
		})
	}

	list, err := repo.GetByUserID(s.ctx, s.testUserID)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), list, 3)

	cancelledSlot := &domain.Slot{
		ID:     uuid.New(),
		RoomID: s.testRoomID,
		Start:  now.Add(100 * 24 * time.Hour),
		End:    now.Add(100*24*time.Hour + 30*time.Minute),
	}
	err = slotRepo.Create(s.ctx, cancelledSlot)
	require.NoError(s.T(), err)

	cancelledBooking := &domain.Booking{
		ID:        uuid.New(),
		SlotID:    cancelledSlot.ID,
		UserID:    s.testUserID,
		Status:    domain.BookingStatusCancelled,
		CreatedAt: timePtr(time.Now()),
	}
	err = repo.Create(s.ctx, cancelledBooking)
	require.NoError(s.T(), err)

	list2, err := repo.GetByUserID(s.ctx, s.testUserID)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), list2, 3)

	newUserID := s.createTestUser()
	list3, err := repo.GetByUserID(s.ctx, newUserID)
	assert.NoError(s.T(), err)
	assert.Empty(s.T(), list3)

	s.cleanup = append(s.cleanup, func() {
		s.db.ExecContext(context.Background(), "DELETE FROM bookings WHERE id = $1", cancelledBooking.ID)
	})
}

func (s *PostgresTestSuite) TestSlotRepository_CreateBatch_WithTransaction() {
	repo := NewSlotRepository(s.db)

	now := time.Now().UTC()
	slots := make([]domain.Slot, 10)
	for i := 0; i < 10; i++ {
		slots[i] = domain.Slot{
			ID:     uuid.New(),
			RoomID: s.testRoomID,
			Start:  now.Add(time.Duration(i+1) * 24 * time.Hour),
			End:    now.Add(time.Duration(i+1)*24*time.Hour + 30*time.Minute),
		}
	}

	err := repo.CreateBatch(s.ctx, slots)
	assert.NoError(s.T(), err)

	for _, slot := range slots {
		var count int
		query := "SELECT COUNT(*) FROM slots WHERE id = $1"
		err = s.db.QueryRowContext(s.ctx, query, slot.ID).Scan(&count)
		assert.NoError(s.T(), err)
		assert.Equal(s.T(), 1, count)
	}

	for _, slot := range slots {
		s.cleanup = append(s.cleanup, func() {
			s.db.ExecContext(context.Background(), "DELETE FROM slots WHERE id = $1", slot.ID)
		})
	}
}

func (s *PostgresTestSuite) TestBookingRepository_Create_WithForeignKeyViolation() {
	repo := NewBookingRepository(s.db)

	now := time.Now()
	booking := &domain.Booking{
		ID:        uuid.New(),
		SlotID:    uuid.New(),
		UserID:    s.testUserID,
		Status:    domain.BookingStatusActive,
		CreatedAt: &now,
	}

	err := repo.Create(s.ctx, booking)
	assert.Error(s.T(), err)
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func TestPostgresTestSuite(t *testing.T) {
	adminDB, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		t.Skipf("Skipping PostgreSQL tests: cannot connect to PostgreSQL: %v", err)
	}
	defer adminDB.Close()

	if err := adminDB.Ping(); err != nil {
		t.Skipf("Skipping PostgreSQL tests: cannot ping PostgreSQL: %v", err)
	}

	t.Log("✅ PostgreSQL is available")

	var exists bool
	err = adminDB.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = 'test-postgres')").Scan(&exists)
	if err != nil {
		t.Logf("Cannot check if test database exists: %v", err)
		exists = false
	}

	if !exists {
		t.Log("Creating database 'test-postgres' for dummyLogin...")
		_, err = adminDB.Exec("CREATE DATABASE \"test-postgres\"")
		if err != nil {
			t.Logf("Warning: Cannot create test-postgres database: %v", err)
		}
	}

	testDB := SetupTestDatabase(t)
	if testDB == nil {
		t.Skip("Failed to setup test database")
	}
	defer TeardownTestDatabase(t, testDB, "booking_service_test")

	testSuite := &PostgresTestSuite{
		db:  testDB,
		ctx: context.Background(),
	}

	suite.Run(t, testSuite)
}

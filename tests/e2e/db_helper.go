package e2e

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type DBHelper struct {
	db *sql.DB
}

func NewDBHelper() (*DBHelper, error) {
	connStr := "host=localhost port=5432 user=postgres password=postgres dbname=booking_service sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	return &DBHelper{db: db}, nil
}

func (h *DBHelper) Close() error {
	return h.db.Close()
}

func (h *DBHelper) GetUserByID(userID uuid.UUID) (*domain.User, error) {
	query := `SELECT id, email, password_hash, role, created_at FROM users WHERE id = $1`

	var user domain.User
	err := h.db.QueryRow(query, userID).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (h *DBHelper) GetUserByEmail(email string) (*domain.User, error) {
	query := `SELECT id, email, password_hash, role, created_at FROM users WHERE email = $1`

	var user domain.User
	err := h.db.QueryRow(query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (h *DBHelper) GetRoomByID(roomID uuid.UUID) (*domain.Room, error) {
	query := `SELECT id, name, description, capacity, created_at FROM rooms WHERE id = $1`

	var room domain.Room
	err := h.db.QueryRow(query, roomID).Scan(
		&room.ID, &room.Name, &room.Description, &room.Capacity, &room.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (h *DBHelper) GetScheduleByRoomID(roomID uuid.UUID) (*domain.Schedule, error) {
	query := `SELECT id, room_id, days_of_week, start_time, end_time, created_at FROM schedules WHERE room_id = $1`

	var schedule domain.Schedule
	var daysOfWeekStr string

	err := h.db.QueryRow(query, roomID).Scan(
		&schedule.ID,
		&schedule.RoomID,
		&daysOfWeekStr,
		&schedule.StartTime,
		&schedule.EndTime,
		&schedule.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	daysOfWeekStr = strings.Trim(daysOfWeekStr, "{}")
	if daysOfWeekStr != "" {
		parts := strings.Split(daysOfWeekStr, ",")
		schedule.DaysOfWeek = make([]int, len(parts))
		for i, part := range parts {
			fmt.Sscanf(part, "%d", &schedule.DaysOfWeek[i])
		}
	}

	return &schedule, nil
}

func (h *DBHelper) GetSlotByID(slotID uuid.UUID) (*domain.Slot, error) {
	query := `SELECT id, room_id, start_at, end_at FROM slots WHERE id = $1`

	var slot domain.Slot
	var startAt, endAt time.Time

	err := h.db.QueryRow(query, slotID).Scan(
		&slot.ID, &slot.RoomID, &startAt, &endAt,
	)
	if err != nil {
		return nil, err
	}

	slot.Start = startAt
	slot.End = endAt
	return &slot, nil
}

func (h *DBHelper) GetSlotsByRoomAndDate(roomID uuid.UUID, date time.Time) ([]domain.Slot, error) {
	startDate := date.Truncate(24 * time.Hour)
	endDate := startDate.Add(24 * time.Hour)

	query := `
		SELECT id, room_id, start_at, end_at 
		FROM slots 
		WHERE room_id = $1 AND start_at >= $2 AND start_at < $3
		ORDER BY start_at
	`

	rows, err := h.db.Query(query, roomID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []domain.Slot
	for rows.Next() {
		var slot domain.Slot
		var startAt, endAt time.Time

		if err := rows.Scan(&slot.ID, &slot.RoomID, &startAt, &endAt); err != nil {
			return nil, err
		}

		slot.Start = startAt
		slot.End = endAt
		slots = append(slots, slot)
	}
	return slots, nil
}

func (h *DBHelper) GetBookingByID(bookingID uuid.UUID) (*domain.Booking, error) {
	query := `SELECT id, slot_id, user_id, status, conference_link, created_at FROM bookings WHERE id = $1`

	var booking domain.Booking
	var createdAt time.Time

	err := h.db.QueryRow(query, bookingID).Scan(
		&booking.ID, &booking.SlotID, &booking.UserID, &booking.Status,
		&booking.ConferenceLink, &createdAt,
	)
	if err != nil {
		return nil, err
	}

	booking.CreatedAt = &createdAt
	return &booking, nil
}

func (h *DBHelper) GetActiveBookingsBySlotID(slotID uuid.UUID) (*domain.Booking, error) {
	query := `
		SELECT id, slot_id, user_id, status, conference_link, created_at 
		FROM bookings 
		WHERE slot_id = $1 AND status = 'active'
	`

	var booking domain.Booking
	var createdAt time.Time

	err := h.db.QueryRow(query, slotID).Scan(
		&booking.ID, &booking.SlotID, &booking.UserID, &booking.Status,
		&booking.ConferenceLink, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	booking.CreatedAt = &createdAt
	return &booking, nil
}

func (h *DBHelper) CountBookingsByUserID(userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM bookings WHERE user_id = $1`

	var count int
	err := h.db.QueryRow(query, userID).Scan(&count)
	return count, err
}

func (h *DBHelper) GetBookingsByUserID(userID uuid.UUID) ([]domain.Booking, error) {
	query := `
		SELECT id, slot_id, user_id, status, conference_link, created_at 
		FROM bookings 
		WHERE user_id = $1 
		ORDER BY created_at DESC
	`

	rows, err := h.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []domain.Booking
	for rows.Next() {
		var booking domain.Booking
		var createdAt time.Time

		if err := rows.Scan(&booking.ID, &booking.SlotID, &booking.UserID,
			&booking.Status, &booking.ConferenceLink, &createdAt); err != nil {
			return nil, err
		}

		booking.CreatedAt = &createdAt
		bookings = append(bookings, booking)
	}
	return bookings, nil
}

func (h *DBHelper) CleanupUserBookings(userID uuid.UUID) error {
	query := `DELETE FROM bookings WHERE user_id = $1`
	_, err := h.db.Exec(query, userID)
	return err
}

func (h *DBHelper) CleanupUser(userID uuid.UUID) error {
	if err := h.CleanupUserBookings(userID); err != nil {
		return err
	}

	query := `DELETE FROM users WHERE id = $1`
	_, err := h.db.Exec(query, userID)
	return err
}

func (h *DBHelper) CleanupRoom(roomID uuid.UUID) error {
	query := `
		DELETE FROM bookings 
		WHERE slot_id IN (SELECT id FROM slots WHERE room_id = $1)
	`
	if _, err := h.db.Exec(query, roomID); err != nil {
		return err
	}

	if _, err := h.db.Exec("DELETE FROM slots WHERE room_id = $1", roomID); err != nil {
		return err
	}

	if _, err := h.db.Exec("DELETE FROM schedules WHERE room_id = $1", roomID); err != nil {
		return err
	}

	if _, err := h.db.Exec("DELETE FROM rooms WHERE id = $1", roomID); err != nil {
		return err
	}

	return nil
}

func (h *DBHelper) PrintTableStats() {
	fmt.Println("\n=== Database Statistics ===")

	var usersCount int
	h.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&usersCount)
	fmt.Printf("Users: %d\n", usersCount)

	var roomsCount int
	h.db.QueryRow("SELECT COUNT(*) FROM rooms").Scan(&roomsCount)
	fmt.Printf("Rooms: %d\n", roomsCount)

	var schedulesCount int
	h.db.QueryRow("SELECT COUNT(*) FROM schedules").Scan(&schedulesCount)
	fmt.Printf("Schedules: %d\n", schedulesCount)

	var slotsCount int
	h.db.QueryRow("SELECT COUNT(*) FROM slots").Scan(&slotsCount)
	fmt.Printf("Slots: %d\n", slotsCount)

	var bookingsCount int
	h.db.QueryRow("SELECT COUNT(*) FROM bookings").Scan(&bookingsCount)
	fmt.Printf("Bookings: %d\n", bookingsCount)

	fmt.Println("===========================")
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

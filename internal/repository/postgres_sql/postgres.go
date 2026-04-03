package postgres_sql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

//
// booking.go
//

type BookingRepository struct {
	db *sql.DB
}

func NewBookingRepository(db *sql.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

func (r *BookingRepository) Create(ctx context.Context, booking *domain.Booking) error {
	query := `
        INSERT INTO bookings (id, slot_id, user_id, status, conference_link, created_at)
        VALUES ($1, $2, $3, $4, $5, $6)
    `
	_, err := r.db.ExecContext(ctx, query,
		booking.ID, booking.SlotID, booking.UserID, booking.Status,
		booking.ConferenceLink, booking.CreatedAt,
	)
	return err
}

func (r *BookingRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	query := `
        SELECT id, slot_id, user_id, status, conference_link, created_at
        FROM bookings
        WHERE id = $1
    `
	var booking domain.Booking
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&booking.ID, &booking.SlotID, &booking.UserID, &booking.Status,
		&booking.ConferenceLink, &booking.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

func (r *BookingRepository) GetBySlotID(ctx context.Context, slotID uuid.UUID) (*domain.Booking, error) {
	query := `
        SELECT id, slot_id, user_id, status, conference_link, created_at
        FROM bookings
        WHERE slot_id = $1 AND status = 'active'
    `
	var booking domain.Booking
	err := r.db.QueryRowContext(ctx, query, slotID).Scan(
		&booking.ID, &booking.SlotID, &booking.UserID, &booking.Status,
		&booking.ConferenceLink, &booking.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

func (r *BookingRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.BookingStatus) error {
	query := `UPDATE bookings SET status = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

func (r *BookingRepository) List(ctx context.Context, page, pageSize int) ([]domain.Booking, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM bookings`
	err := r.db.QueryRowContext(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query := `
        SELECT id, slot_id, user_id, status, conference_link, created_at
        FROM bookings
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2
    `
	rows, err := r.db.QueryContext(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var bookings []domain.Booking
	for rows.Next() {
		var booking domain.Booking
		err := rows.Scan(
			&booking.ID, &booking.SlotID, &booking.UserID, &booking.Status,
			&booking.ConferenceLink, &booking.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		bookings = append(bookings, booking)
	}
	return bookings, total, nil
}

func (r *BookingRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Booking, error) {
	query := `
        SELECT id, slot_id, user_id, status, conference_link, created_at
        FROM bookings
        WHERE user_id = $1 AND status = 'active'
        ORDER BY created_at DESC
    `
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return []domain.Booking{}, err
	}
	defer rows.Close()

	var bookings []domain.Booking
	for rows.Next() {
		var booking domain.Booking
		err := rows.Scan(
			&booking.ID, &booking.SlotID, &booking.UserID, &booking.Status,
			&booking.ConferenceLink, &booking.CreatedAt,
		)
		if err != nil {
			return []domain.Booking{}, err
		}
		bookings = append(bookings, booking)
	}
	if bookings == nil {
		return []domain.Booking{}, nil
	}
	return bookings, nil
}

//
// room.go
//

type RoomRepository struct {
	db *sql.DB
}

func NewRoomRepository(db *sql.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

func (r *RoomRepository) Create(ctx context.Context, room *domain.Room) error {
	query := `
        INSERT INTO rooms (id, name, description, capacity)
        VALUES ($1, $2, $3, $4)
    `
	_, err := r.db.ExecContext(ctx, query, room.ID, room.Name, room.Description, room.Capacity)
	return err
}

func (r *RoomRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Room, error) {
	query := `
        SELECT id, name, description, capacity, created_at
        FROM rooms
        WHERE id = $1
    `
	var room domain.Room
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&room.ID, &room.Name, &room.Description, &room.Capacity, &room.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *RoomRepository) List(ctx context.Context) ([]domain.Room, error) {
	query := `
        SELECT id, name, description, capacity, created_at
        FROM rooms
        ORDER BY created_at DESC
    `
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []domain.Room
	for rows.Next() {
		var room domain.Room
		err := rows.Scan(&room.ID, &room.Name, &room.Description, &room.Capacity, &room.CreatedAt)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, nil
}

func (r *RoomRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM rooms WHERE id = $1)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	return exists, err
}

//
// schedules.go
//

type ScheduleRepository struct {
	db *sql.DB
}

func NewScheduleRepository(db *sql.DB) *ScheduleRepository {
	return &ScheduleRepository{db: db}
}

func (r *ScheduleRepository) Create(ctx context.Context, schedule *domain.Schedule) error {
	query := `
        INSERT INTO schedules (id, room_id, days_of_week, start_time, end_time, created_at)
        VALUES ($1, $2, $3, $4::time, $5::time, $6)
    `

	daysOfWeek := make([]int64, len(schedule.DaysOfWeek))
	for i, v := range schedule.DaysOfWeek {
		daysOfWeek[i] = int64(v)
	}

	_, err := r.db.ExecContext(ctx, query,
		schedule.ID,
		schedule.RoomID,
		pq.Array(daysOfWeek),
		schedule.StartTime,
		schedule.EndTime,
		schedule.CreatedAt,
	)
	return err
}

func (r *ScheduleRepository) GetByRoomID(ctx context.Context, roomID uuid.UUID) (*domain.Schedule, error) {
	query := `
        SELECT id, room_id, days_of_week, 
               EXTRACT(HOUR FROM start_time) as start_hour,
               EXTRACT(MINUTE FROM start_time) as start_minute,
               EXTRACT(HOUR FROM end_time) as end_hour,
               EXTRACT(MINUTE FROM end_time) as end_minute,
               created_at
        FROM schedules
        WHERE room_id = $1
    `
	var schedule domain.Schedule
	var daysOfWeek pq.Int64Array
	var startHour, startMinute, endHour, endMinute int

	err := r.db.QueryRowContext(ctx, query, roomID).Scan(
		&schedule.ID,
		&schedule.RoomID,
		&daysOfWeek,
		&startHour,
		&startMinute,
		&endHour,
		&endMinute,
		&schedule.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	schedule.DaysOfWeek = make([]int, len(daysOfWeek))
	for i, v := range daysOfWeek {
		schedule.DaysOfWeek[i] = int(v)
	}

	schedule.StartTime = fmt.Sprintf("%02d:%02d", startHour, startMinute)
	schedule.EndTime = fmt.Sprintf("%02d:%02d", endHour, endMinute)

	return &schedule, nil
}

func formatTimeString(timeStr string) string {
	parts := strings.Split(timeStr, ".")
	if len(parts) > 0 {
		timeStr = parts[0]
	}
	if len(timeStr) >= 5 {
		return timeStr[:5]
	}
	return timeStr
}
func (r *ScheduleRepository) Exists(ctx context.Context, roomID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM schedules WHERE room_id = $1)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, roomID).Scan(&exists)
	return exists, err
}

//
// slots.go
//

type SlotRepository struct {
	db *sql.DB
}

func NewSlotRepository(db *sql.DB) *SlotRepository {
	return &SlotRepository{db: db}
}

func (r *SlotRepository) Create(ctx context.Context, slot *domain.Slot) error {
	query := `
        INSERT INTO slots (id, room_id, start_at, end_at)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT DO NOTHING
    `
	_, err := r.db.ExecContext(ctx, query, slot.ID, slot.RoomID, slot.Start, slot.End)
	return err
}

func (r *SlotRepository) CreateBatch(ctx context.Context, slots []domain.Slot) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
        INSERT INTO slots (id, room_id, start_at, end_at)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT DO NOTHING
    `
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, slot := range slots {
		_, err := stmt.ExecContext(ctx, slot.ID, slot.RoomID, slot.Start, slot.End)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SlotRepository) GetByRoomAndDate(ctx context.Context, roomID uuid.UUID, date time.Time) ([]domain.Slot, error) {
	query := `
        SELECT s.id, s.room_id, s.start_at, s.end_at
        FROM slots s
        LEFT JOIN bookings b ON b.slot_id = s.id AND b.status = 'active'
        WHERE s.room_id = $1
          AND s.start_at >= $2
          AND s.start_at < $3
          AND b.id IS NULL
        ORDER BY s.start_at
    `
	startDate := date.Truncate(24 * time.Hour)
	endDate := startDate.Add(24 * time.Hour)

	rows, err := r.db.QueryContext(ctx, query, roomID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []domain.Slot
	for rows.Next() {
		var slot domain.Slot
		err := rows.Scan(&slot.ID, &slot.RoomID, &slot.Start, &slot.End)
		if err != nil {
			return nil, err
		}
		slots = append(slots, slot)
	}
	return slots, nil
}

func (r *SlotRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Slot, error) {
	query := `SELECT id, room_id, start_at, end_at FROM slots WHERE id = $1`
	var slot domain.Slot
	err := r.db.QueryRowContext(ctx, query, id).Scan(&slot.ID, &slot.RoomID, &slot.Start, &slot.End)
	if err != nil {
		return nil, err
	}
	return &slot, nil
}

func (r *SlotRepository) GetFutureSlotsByUser(ctx context.Context, userID uuid.UUID) ([]domain.Slot, error) {
	query := `
        SELECT s.id, s.room_id, s.start_at, s.end_at
        FROM slots s
        INNER JOIN bookings b ON b.slot_id = s.id
        WHERE b.user_id = $1
          AND b.status = 'active'
          AND s.start_at >= NOW()
        ORDER BY s.start_at
    `
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []domain.Slot
	for rows.Next() {
		var slot domain.Slot
		err := rows.Scan(&slot.ID, &slot.RoomID, &slot.Start, &slot.End)
		if err != nil {
			return nil, err
		}
		slots = append(slots, slot)
	}
	return slots, nil
}

//
// users.go
//

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
        INSERT INTO users (id, email, password_hash, role, created_at)
        VALUES ($1, $2, $3, $4, $5)
    `
	_, err := r.db.ExecContext(ctx, query, user.ID, user.Email, user.PasswordHash, user.Role, user.CreatedAt)
	return err
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
        SELECT id, email, password_hash, role, created_at
        FROM users
        WHERE email = $1
    `
	var user domain.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
        SELECT id, email, password_hash, role, created_at
        FROM users
        WHERE id = $1
    `
	var user domain.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

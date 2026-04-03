package domain

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type BookingStatus string

const (
	BookingStatusActive    BookingStatus = "active"
	BookingStatusCancelled BookingStatus = "cancelled"
)

type Booking struct {
	ID             uuid.UUID     `json:"id"`
	SlotID         uuid.UUID     `json:"slotId"`
	UserID         uuid.UUID     `json:"userId"`
	Status         BookingStatus `json:"status"`
	ConferenceLink *string       `json:"conferenceLink,omitempty"`
	CreatedAt      *time.Time    `json:"createdAt,omitempty"`
}

type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

var (
	ErrRoomNotFound      = &AppError{Code: "ROOM_NOT_FOUND", Message: "room not found"}
	ErrSlotNotFound      = &AppError{Code: "SLOT_NOT_FOUND", Message: "slot not found"}
	ErrSlotInPast        = &AppError{Code: "INVALID_REQUEST", Message: "slot is in the past"}
	ErrSlotAlreadyBooked = &AppError{Code: "SLOT_ALREADY_BOOKED", Message: "slot is already booked"}
	ErrBookingNotFound   = &AppError{Code: "BOOKING_NOT_FOUND", Message: "booking not found"}
	ErrForbidden         = &AppError{Code: "FORBIDDEN", Message: "access denied"}
)

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AppError) Error() string {
	return e.Message
}

type ErrorResponse struct {
	Error AppError `json:"error"`
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: AppError{
			Code:    code,
			Message: message,
		},
	})
}

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

type Room struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	Capacity    *int       `json:"capacity,omitempty"`
	CreatedAt   *time.Time `json:"createdAt,omitempty"`
}

type Schedule struct {
	ID         uuid.UUID  `json:"id"`
	RoomID     uuid.UUID  `json:"roomId"`
	DaysOfWeek []int      `json:"daysOfWeek"`
	StartTime  string     `json:"startTime"`
	EndTime    string     `json:"endTime"`
	CreatedAt  *time.Time `json:"createdAt,omitempty"`
}

type Slot struct {
	ID     uuid.UUID `json:"id"`
	RoomID uuid.UUID `json:"roomId"`
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
}

type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         UserRole  `json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
}

var (
	TestAdminID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	TestUserID  = uuid.MustParse("00000000-0000-0000-0000-000000000002")
)

package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"booking-service/internal/domain"
	"booking-service/internal/middleware"
	"booking-service/internal/service"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type BookingHandler struct {
	bookingService *service.BookingService
}

func NewBookingHandler(bookingService *service.BookingService) *BookingHandler {
	return &BookingHandler{
		bookingService: bookingService,
	}
}

type CreateBookingRequest struct {
	SlotID               uuid.UUID `json:"slotId"`
	CreateConferenceLink bool      `json:"createConferenceLink"`
}

func (h *BookingHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		domain.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	var req CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		domain.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	booking, err := h.bookingService.Create(r.Context(), claims.UserID, req.SlotID, req.CreateConferenceLink)
	if err != nil {
		switch err {
		case domain.ErrSlotNotFound:
			domain.WriteError(w, http.StatusNotFound, "SLOT_NOT_FOUND", err.Error())
		case domain.ErrSlotInPast:
			domain.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		case domain.ErrSlotAlreadyBooked:
			domain.WriteError(w, http.StatusConflict, "SLOT_ALREADY_BOOKED", err.Error())
		default:
			domain.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}

	domain.WriteJSON(w, http.StatusCreated, map[string]interface{}{"booking": booking})
}

func (h *BookingHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))

	bookings, pagination, err := h.bookingService.ListAll(r.Context(), page, pageSize)
	if err != nil {
		domain.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	domain.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"bookings":   bookings,
		"pagination": pagination,
	})
}

func (h *BookingHandler) ListMy(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		domain.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	bookings, err := h.bookingService.ListUserBookings(r.Context(), claims.UserID)
	if err != nil {
		domain.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	domain.WriteJSON(w, http.StatusOK, map[string]interface{}{"bookings": bookings})
}

func (h *BookingHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bookingIDStr := vars["bookingId"]
	bookingID, err := uuid.Parse(bookingIDStr)
	if err != nil {
		domain.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid booking id")
		return
	}

	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		domain.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	booking, err := h.bookingService.Cancel(r.Context(), bookingID, claims.UserID)
	if err != nil {
		switch err {
		case domain.ErrBookingNotFound:
			domain.WriteError(w, http.StatusNotFound, "BOOKING_NOT_FOUND", err.Error())
		case domain.ErrForbidden:
			domain.WriteError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		default:
			domain.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}

	domain.WriteJSON(w, http.StatusOK, map[string]interface{}{"booking": booking})
}

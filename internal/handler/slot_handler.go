package handler

import (
	"net/http"
	"time"

	"booking-service/internal/domain"
	"booking-service/internal/service"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type SlotHandler struct {
	slotService *service.SlotService
}

func NewSlotHandler(slotService *service.SlotService) *SlotHandler {
	return &SlotHandler{
		slotService: slotService,
	}
}

func (h *SlotHandler) ListAvailable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomIDStr := vars["roomId"]
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		domain.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid room id")
		return
	}

	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		domain.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "date parameter is required")
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		domain.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid date format, expected YYYY-MM-DD")
		return
	}

	slots, err := h.slotService.GetAvailableSlots(r.Context(), roomID, date)
	if err != nil {
		if err == domain.ErrRoomNotFound {
			domain.WriteError(w, http.StatusNotFound, "ROOM_NOT_FOUND", err.Error())
			return
		}
		domain.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	domain.WriteJSON(w, http.StatusOK, map[string]interface{}{"slots": slots})
}

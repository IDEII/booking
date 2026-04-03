package handler

import (
	"encoding/json"
	"net/http"

	"booking-service/internal/domain"
	"booking-service/internal/service"
)

//
// room_handler.go
//

type RoomHandler struct {
	roomService *service.RoomService
}

func NewRoomHandler(roomService *service.RoomService) *RoomHandler {
	return &RoomHandler{
		roomService: roomService,
	}
}

type CreateRoomRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Capacity    *int    `json:"capacity,omitempty"`
}

func (h *RoomHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		domain.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	room, err := h.roomService.Create(r.Context(), req.Name, req.Description, req.Capacity)
	if err != nil {
		domain.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	domain.WriteJSON(w, http.StatusCreated, map[string]interface{}{"room": room})
}

func (h *RoomHandler) List(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.roomService.List(r.Context())
	if err != nil {
		domain.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	domain.WriteJSON(w, http.StatusOK, map[string]interface{}{"rooms": rooms})
}

package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"booking-service/internal/domain"
	"booking-service/internal/service"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

//
// schedule_handler.go
//

type ScheduleHandler struct {
	scheduleService *service.ScheduleService
}

func NewScheduleHandler(scheduleService *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{
		scheduleService: scheduleService,
	}
}

type CreateScheduleRequest struct {
	DaysOfWeek []int  `json:"daysOfWeek"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
}

func (h *ScheduleHandler) Create(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomIDStr := vars["roomId"]
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		domain.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid room id")
		return
	}

	var req CreateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		domain.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	if len(req.DaysOfWeek) == 0 {
		domain.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "days of week cannot be empty")
		return
	}

	for _, day := range req.DaysOfWeek {
		if day < 1 || day > 7 {
			domain.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", fmt.Sprintf("invalid day of week: %d", day))
			return
		}
	}

	startTime, err := time.Parse("15:04", req.StartTime)
	if err != nil {
		domain.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid start time format")
		return
	}

	endTime, err := time.Parse("15:04", req.EndTime)
	if err != nil {
		domain.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid end time format")
		return
	}

	if startTime.After(endTime) || startTime.Equal(endTime) {
		domain.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "start time must be before end time")
		return
	}

	schedule, err := h.scheduleService.Create(r.Context(), roomID, req.DaysOfWeek, req.StartTime, req.EndTime)
	if err != nil {
		if err.Error() == "room not found" {
			domain.WriteError(w, http.StatusNotFound, "ROOM_NOT_FOUND", err.Error())
			return
		}
		if err.Error() == "schedule already exists" {
			domain.WriteError(w, http.StatusConflict, "SCHEDULE_EXISTS", err.Error())
			return
		}
		domain.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	domain.WriteJSON(w, http.StatusCreated, map[string]interface{}{"schedule": schedule})
}

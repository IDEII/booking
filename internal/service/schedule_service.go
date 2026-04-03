package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"booking-service/internal/domain"
	"booking-service/internal/repository"

	"github.com/google/uuid"
)

type ScheduleService struct {
	scheduleRepo repository.ScheduleRepository
	roomRepo     repository.RoomRepository
	slotRepo     repository.SlotRepository
	slotDuration time.Duration
	slotService  *SlotService
}

func NewScheduleService(
	scheduleRepo repository.ScheduleRepository,
	roomRepo repository.RoomRepository,
	slotRepo repository.SlotRepository,
	slotDuration time.Duration,
	slotService *SlotService,
) *ScheduleService {
	return &ScheduleService{
		scheduleRepo: scheduleRepo,
		roomRepo:     roomRepo,
		slotRepo:     slotRepo,
		slotDuration: slotDuration,
		slotService:  slotService,
	}
}

func (s *ScheduleService) Create(ctx context.Context, roomID uuid.UUID, daysOfWeek []int, startTime, endTime string) (*domain.Schedule, error) {
	for _, day := range daysOfWeek {
		if day < 1 || day > 7 {
			return nil, fmt.Errorf("invalid day of week: %d", day)
		}
	}

	if _, err := time.Parse("15:04", startTime); err != nil {
		return nil, fmt.Errorf("invalid start time format: %w", err)
	}
	if _, err := time.Parse("15:04", endTime); err != nil {
		return nil, fmt.Errorf("invalid end time format: %w", err)
	}

	start, _ := time.Parse("15:04", startTime)
	end, _ := time.Parse("15:04", endTime)
	if start.After(end) || start.Equal(end) {
		return nil, errors.New("start time must be before end time")
	}

	exists, err := s.roomRepo.Exists(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("room not found")
	}

	scheduleExists, err := s.scheduleRepo.Exists(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if scheduleExists {
		return nil, errors.New("schedule already exists")
	}

	now := time.Now()
	schedule := &domain.Schedule{
		ID:         uuid.New(),
		RoomID:     roomID,
		DaysOfWeek: daysOfWeek,
		StartTime:  startTime,
		EndTime:    endTime,
		CreatedAt:  &now,
	}

	if err := s.scheduleRepo.Create(ctx, schedule); err != nil {
		return nil, err
	}

	if s.slotService != nil {
		go func() {
			genCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := s.slotService.PreGenerateSlotsForRange(genCtx, roomID, 7); err != nil {
				fmt.Printf("[SCHEDULE-SERVICE] Failed to pre-generate slots for room %s: %v\n", roomID, err)
			}
		}()
	}

	return schedule, nil
}

func (s *ScheduleService) GenerateSlots(ctx context.Context, roomID uuid.UUID, startDate, endDate time.Time) error {
	schedule, err := s.scheduleRepo.GetByRoomID(ctx, roomID)
	if err != nil {
		return err
	}
	if schedule == nil {
		return nil
	}

	var allSlots []domain.Slot
	currentDate := startDate

	for currentDate.Before(endDate) {
		weekday := int(currentDate.Weekday())
		mappedDay := weekday
		if mappedDay == 0 {
			mappedDay = 7
		}

		if contains(schedule.DaysOfWeek, mappedDay) {
			slots, err := s.generateSlotsForDay(currentDate, schedule.StartTime, schedule.EndTime, roomID)
			if err != nil {
				return err
			}
			allSlots = append(allSlots, slots...)
		}

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	if len(allSlots) > 0 {
		return s.slotRepo.CreateBatch(ctx, allSlots)
	}

	return nil
}

func (s *ScheduleService) generateSlotsForDay(date time.Time, startTimeStr, endTimeStr string, roomID uuid.UUID) ([]domain.Slot, error) {
	startTime, err := time.Parse("15:04", startTimeStr)
	if err != nil {
		return nil, fmt.Errorf("parsing start time: %w", err)
	}
	endTime, err := time.Parse("15:04", endTimeStr)
	if err != nil {
		return nil, fmt.Errorf("parsing end time: %w", err)
	}

	slotStart := time.Date(date.Year(), date.Month(), date.Day(),
		startTime.Hour(), startTime.Minute(), 0, 0, time.UTC)

	slotEnd := time.Date(date.Year(), date.Month(), date.Day(),
		endTime.Hour(), endTime.Minute(), 0, 0, time.UTC)

	if slotStart.After(slotEnd) || slotStart.Equal(slotEnd) {
		return nil, fmt.Errorf("start time must be before end time")
	}

	var slots []domain.Slot
	for slotStart.Before(slotEnd) {
		slot := domain.Slot{
			ID:     uuid.New(),
			RoomID: roomID,
			Start:  slotStart,
			End:    slotStart.Add(s.slotDuration),
		}

		if slot.End.After(slotEnd) {
			break
		}

		slots = append(slots, slot)
		slotStart = slotStart.Add(s.slotDuration)
	}

	return slots, nil
}

func contains(slice []int, item int) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

package service

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"booking-service/internal/domain"
	"booking-service/internal/repository"

	"github.com/google/uuid"
)

type SlotService struct {
	slotRepo     repository.SlotRepository
	roomRepo     repository.RoomRepository
	scheduleRepo repository.ScheduleRepository
	slotDuration time.Duration

	generatedDatesCache sync.Map
	cacheTTL            time.Duration

	generationQueue chan generationTask
	workerCount     int
}

type generationTask struct {
	ctx      context.Context
	roomID   uuid.UUID
	date     time.Time
	schedule *domain.Schedule
}

func NewSlotService(
	slotRepo repository.SlotRepository,
	roomRepo repository.RoomRepository,
	scheduleRepo repository.ScheduleRepository,
) *SlotService {
	svc := &SlotService{
		slotRepo:            slotRepo,
		roomRepo:            roomRepo,
		scheduleRepo:        scheduleRepo,
		slotDuration:        30 * time.Minute,
		generationQueue:     make(chan generationTask, 1000),
		workerCount:         5,
		generatedDatesCache: sync.Map{},
		cacheTTL:            1 * time.Hour,
	}

	for i := 0; i < svc.workerCount; i++ {
		go svc.generationWorker()
	}

	go svc.cacheCleaner()

	return svc
}

func (s *SlotService) generationWorker() {
	for task := range s.generationQueue {
		select {
		case <-task.ctx.Done():
			continue
		default:
		}

		slots, err := s.generateSlotsForDayInternal(task.roomID, task.date, task.schedule)
		if err != nil {
			fmt.Printf("[SLOT-SERVICE] Background generation failed for room %s, date %s: %v\n",
				task.roomID, task.date.Format("2006-01-02"), err)
			continue
		}

		if len(slots) > 0 {
			if err := s.slotRepo.CreateBatch(task.ctx, slots); err != nil {
				fmt.Printf("[SLOT-SERVICE] Failed to save generated slots: %v\n", err)
			} else {
				fmt.Printf("[SLOT-SERVICE] Background generated %d slots for room %s, date %s\n",
					len(slots), task.roomID, task.date.Format("2006-01-02"))
			}
		}

		cacheKey := s.getCacheKey(task.roomID, task.date)
		s.generatedDatesCache.Store(cacheKey, time.Now())
	}
}

func (s *SlotService) cacheCleaner() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.generatedDatesCache.Range(func(key, value interface{}) bool {
			if timestamp, ok := value.(time.Time); ok {
				if time.Since(timestamp) > s.cacheTTL {
					s.generatedDatesCache.Delete(key)
				}
			}
			return true
		})
	}
}

func (s *SlotService) getCacheKey(roomID uuid.UUID, date time.Time) string {
	return fmt.Sprintf("%s_%s", roomID.String(), date.Format("2006-01-02"))
}

func (s *SlotService) GetAvailableSlots(ctx context.Context, roomID uuid.UUID, date time.Time) ([]domain.Slot, error) {
	exists, err := s.roomRepo.Exists(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to check room existence: %w", err)
	}
	if !exists {
		return nil, domain.ErrRoomNotFound
	}

	schedule, err := s.scheduleRepo.GetByRoomID(ctx, roomID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	if schedule == nil {
		return []domain.Slot{}, nil
	}

	weekday := int(date.Weekday())
	mappedDay := weekday
	if mappedDay == 0 {
		mappedDay = 7
	}

	fmt.Printf("[SLOT-SERVICE] Date: %s, weekday: %d, mappedDay: %d, schedule days: %v\n",
		date.Format("2006-01-02"), weekday, mappedDay, schedule.DaysOfWeek)

	if !contains(schedule.DaysOfWeek, mappedDay) {
		fmt.Printf("[SLOT-SERVICE] Day %d not in schedule days %v\n", mappedDay, schedule.DaysOfWeek)
		return []domain.Slot{}, nil
	}

	slots, err := s.slotRepo.GetByRoomAndDate(ctx, roomID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get slots from DB: %w", err)
	}

	if len(slots) > 0 {
		return slots, nil
	}

	if date.Before(time.Now().UTC().Truncate(24 * time.Hour)) {
		return []domain.Slot{}, nil
	}

	cacheKey := s.getCacheKey(roomID, date)
	if _, exists := s.generatedDatesCache.Load(cacheKey); exists {
		return []domain.Slot{}, nil
	}

	select {
	case s.generationQueue <- generationTask{
		ctx:      context.Background(),
		roomID:   roomID,
		date:     date,
		schedule: schedule,
	}:
		fmt.Printf("[SLOT-SERVICE] Queued background generation for room %s, date %s\n",
			roomID, date.Format("2006-01-02"))
	default:
		fmt.Printf("[SLOT-SERVICE] Generation queue full for room %s, date %s\n",
			roomID, date.Format("2006-01-02"))
	}

	return []domain.Slot{}, nil
}

func (s *SlotService) generateSlotsForDayInternal(roomID uuid.UUID, date time.Time, schedule *domain.Schedule) ([]domain.Slot, error) {
	startHour, startMinute, err := parseTimeString(schedule.StartTime)
	if err != nil {
		return nil, fmt.Errorf("parsing start time: %w", err)
	}

	endHour, endMinute, err := parseTimeString(schedule.EndTime)
	if err != nil {
		return nil, fmt.Errorf("parsing end time: %w", err)
	}

	slotStart := time.Date(date.Year(), date.Month(), date.Day(),
		startHour, startMinute, 0, 0, time.UTC)

	slotEnd := time.Date(date.Year(), date.Month(), date.Day(),
		endHour, endMinute, 0, 0, time.UTC)

	if slotStart.After(slotEnd) || slotStart.Equal(slotEnd) {
		return nil, fmt.Errorf("start time must be before end time")
	}

	var slots []domain.Slot
	currentStart := slotStart
	now := time.Now().UTC()

	for currentStart.Before(slotEnd) {
		slotEndTime := currentStart.Add(s.slotDuration)

		if slotEndTime.After(slotEnd) {
			break
		}

		if currentStart.After(now) {
			slot := domain.Slot{
				ID:     uuid.New(),
				RoomID: roomID,
				Start:  currentStart,
				End:    slotEndTime,
			}
			slots = append(slots, slot)
		}

		currentStart = slotEndTime
	}

	return slots, nil
}

func (s *SlotService) PreGenerateSlotsForRange(ctx context.Context, roomID uuid.UUID, daysAhead int) error {
	schedule, err := s.scheduleRepo.GetByRoomID(ctx, roomID)
	if err != nil {
		return err
	}
	if schedule == nil {
		return nil
	}

	now := time.Now().UTC()
	endDate := now.AddDate(0, 0, daysAhead)

	var tasks []generationTask
	currentDate := now.Truncate(24 * time.Hour)

	for currentDate.Before(endDate) {
		weekday := int(currentDate.Weekday())
		mappedDay := weekday
		if mappedDay == 0 {
			mappedDay = 7
		}

		if contains(schedule.DaysOfWeek, mappedDay) {
			cacheKey := s.getCacheKey(roomID, currentDate)
			if _, exists := s.generatedDatesCache.Load(cacheKey); !exists {
				tasks = append(tasks, generationTask{
					ctx:      ctx,
					roomID:   roomID,
					date:     currentDate,
					schedule: schedule,
				})
			}
		}
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	go func() {
		for _, task := range tasks {
			select {
			case s.generationQueue <- task:
			default:
				fmt.Printf("[SLOT-SERVICE] Queue full, skipping date %s\n", task.date.Format("2006-01-02"))
			}
		}
	}()

	return nil
}

func (s *SlotService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Slot, error) {
	return s.slotRepo.GetByID(ctx, id)
}

func parseTimeString(timeStr string) (hour, minute int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in parseTimeString: %v", r)
		}
	}()

	timeStr = strings.TrimSpace(timeStr)

	if timeStr == "" {
		return 0, 0, fmt.Errorf("empty time string")
	}

	if len(timeStr) < 5 {
		return 0, 0, fmt.Errorf("time string too short: %s", timeStr)
	}

	parts := strings.Split(timeStr, ":")
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("invalid time format (no colon): %s", timeStr)
	}

	hour, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid hour: %s", parts[0])
	}

	minute, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minute: %s", parts[1])
	}

	if hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("hour out of range: %d", hour)
	}

	if minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("minute out of range: %d", minute)
	}

	return hour, minute, nil
}

func (s *SlotService) IsSlotInPast(ctx context.Context, slotID uuid.UUID) (bool, error) {
	slot, err := s.slotRepo.GetByID(ctx, slotID)
	if err != nil {
		return false, err
	}
	if slot == nil {
		return false, nil
	}
	return slot.Start.Before(time.Now().Truncate(time.Second)), nil
}

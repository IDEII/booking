package service

import (
	"context"
	"time"

	"booking-service/internal/domain"
	"booking-service/internal/repository"

	"github.com/google/uuid"
)

type BookingService struct {
	bookingRepo     repository.BookingRepository
	slotRepo        repository.SlotRepository
	conferenceSvc   ConferenceService
	maxPageSize     int
	defaultPageSize int
}

func NewBookingService(
	bookingRepo repository.BookingRepository,
	slotRepo repository.SlotRepository,
	conferenceSvc ConferenceService,
	maxPageSize, defaultPageSize int,
) *BookingService {
	return &BookingService{
		bookingRepo:     bookingRepo,
		slotRepo:        slotRepo,
		conferenceSvc:   conferenceSvc,
		maxPageSize:     maxPageSize,
		defaultPageSize: defaultPageSize,
	}
}

func (s *BookingService) Create(ctx context.Context, userID uuid.UUID, slotID uuid.UUID, createConferenceLink bool) (*domain.Booking, error) {
	slot, err := s.slotRepo.GetByID(ctx, slotID)
	if err != nil {
		return nil, err
	}
	if slot == nil {
		return nil, domain.ErrSlotNotFound
	}

	if slot.Start.Before(time.Now()) {
		return nil, domain.ErrSlotInPast
	}

	existingBooking, err := s.bookingRepo.GetBySlotID(ctx, slotID)
	if err != nil {
		return nil, err
	}
	if existingBooking != nil {
		return nil, domain.ErrSlotAlreadyBooked
	}

	now := time.Now()
	booking := &domain.Booking{
		ID:        uuid.New(),
		SlotID:    slotID,
		UserID:    userID,
		Status:    domain.BookingStatusActive,
		CreatedAt: &now,
	}

	if createConferenceLink {
		link, err := s.conferenceSvc.CreateConference(ctx, slot)
		if err != nil {
			return nil, err
		}
		booking.ConferenceLink = &link
	}

	if err := s.bookingRepo.Create(ctx, booking); err != nil {
		return nil, err
	}

	return booking, nil
}

func (s *BookingService) Cancel(ctx context.Context, bookingID uuid.UUID, userID uuid.UUID) (*domain.Booking, error) {
	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if booking == nil {
		return nil, domain.ErrBookingNotFound
	}

	if booking.UserID != userID {
		return nil, domain.ErrForbidden
	}

	if booking.Status == domain.BookingStatusCancelled {
		return booking, nil
	}

	if err := s.bookingRepo.UpdateStatus(ctx, bookingID, domain.BookingStatusCancelled); err != nil {
		return nil, err
	}

	booking.Status = domain.BookingStatusCancelled
	return booking, nil
}

func (s *BookingService) ListAll(ctx context.Context, page, pageSize int) ([]domain.Booking, *domain.Pagination, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = s.defaultPageSize
	}
	if pageSize > s.maxPageSize {
		pageSize = s.maxPageSize
	}

	bookings, total, err := s.bookingRepo.List(ctx, page, pageSize)
	if err != nil {
		return nil, nil, err
	}

	pagination := &domain.Pagination{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}

	return bookings, pagination, nil
}

func (s *BookingService) ListUserBookings(ctx context.Context, userID uuid.UUID) ([]domain.Booking, error) {
	bookings, err := s.bookingRepo.GetByUserID(ctx, userID)
	if err != nil {
		return []domain.Booking{}, err
	}

	var futureBookings []domain.Booking
	for _, booking := range bookings {
		slot, err := s.slotRepo.GetByID(ctx, booking.SlotID)
		if err != nil {
			continue
		}
		if slot != nil && slot.Start.After(time.Now()) {
			futureBookings = append(futureBookings, booking)
		}
	}

	if futureBookings == nil {
		return []domain.Booking{}, nil
	}
	return futureBookings, nil
}

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (c *TestClient) Register(email, password, role string) (*domain.User, error) {
	reqBody := map[string]interface{}{
		"email":    email,
		"password": password,
		"role":     role,
	}

	resp, body, err := c.doRequest("POST", "/register", reqBody)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("register failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		User domain.User `json:"user"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result.User, nil
}

func (c *TestClient) Login(email, password string) (string, error) {
	reqBody := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	resp, body, err := c.doRequest("POST", "/login", reqBody)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	c.token = result.Token
	return result.Token, nil
}

func TestInfo(t *testing.T) {
	client := NewTestClient(baseURL)
	err := client.Info()
	assert.NoError(t, err)
}

func TestDummyLogin(t *testing.T) {
	client := NewTestClient(baseURL)

	token, err := client.LoginAs("admin")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	token, err = client.LoginAs("user")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	_, err = client.LoginAs("invalid")
	assert.Error(t, err)
}

func TestRegisterAndLogin(t *testing.T) {
	client := NewTestClient(baseURL)
	email := fmt.Sprintf("test_%d@example.com", time.Now().UnixNano())

	user, err := client.Register(email, "password123", "user")
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, email, user.Email)
	assert.Equal(t, domain.RoleUser, user.Role)

	token, err := client.Login(email, "password123")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	_, err = client.Login(email, "wrongpassword")
	assert.Error(t, err)

	_, err = client.Register(email, "password123", "user")
	assert.Error(t, err)
}

func TestRooms(t *testing.T) {
	adminClient := NewTestClient(baseURL)
	_, err := adminClient.LoginAs("admin")
	require.NoError(t, err)

	room, err := adminClient.CreateRoom("Test Room", stringPtr("Description"), intPtr(10))
	assert.NoError(t, err)
	assert.NotNil(t, room)
	assert.Equal(t, "Test Room", room.Name)

	rooms, err := adminClient.ListRooms()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(rooms), 1)

	userClient := NewTestClient(baseURL)
	_, err = userClient.LoginAs("user")
	require.NoError(t, err)

	_, err = userClient.CreateRoom("Should Fail", nil, nil)
	assert.Error(t, err)

	rooms, err = userClient.ListRooms()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(rooms), 1)
}

func TestSchedules(t *testing.T) {
	adminClient := NewTestClient(baseURL)
	_, err := adminClient.LoginAs("admin")
	require.NoError(t, err)

	room, err := adminClient.CreateRoom("Schedule Test Room", nil, nil)
	require.NoError(t, err)

	schedule, err := adminClient.CreateSchedule(room.ID, []int{1, 2, 3, 4, 5}, "09:00", "18:00")
	assert.NoError(t, err)
	assert.NotNil(t, schedule)
	assert.Equal(t, room.ID, schedule.RoomID)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, schedule.DaysOfWeek)

	_, err = adminClient.CreateSchedule(room.ID, []int{1, 2, 3}, "10:00", "17:00")
	assert.Error(t, err)

	_, err = adminClient.CreateSchedule(room.ID, []int{0, 8}, "09:00", "18:00")
	assert.Error(t, err)

	_, err = adminClient.CreateSchedule(room.ID, []int{1, 2, 3}, "18:00", "09:00")
	assert.Error(t, err)

	_, err = adminClient.CreateSchedule(uuid.New(), []int{1, 2, 3}, "09:00", "18:00")
	assert.Error(t, err)
}

func TestSlots(t *testing.T) {
	adminClient := NewTestClient(baseURL)
	_, err := adminClient.LoginAs("admin")
	require.NoError(t, err)

	room, err := adminClient.CreateRoom("Slots Test Room", nil, nil)
	require.NoError(t, err)

	_, err = adminClient.CreateSchedule(room.ID, []int{1, 2, 3, 4, 5}, "09:00", "18:00")
	require.NoError(t, err)

	userClient := NewTestClient(baseURL)
	_, err = userClient.LoginAs("user")
	require.NoError(t, err)

	now := time.Now().UTC()
	nextWeekday := now.AddDate(0, 0, 1)
	for nextWeekday.Weekday() == time.Saturday || nextWeekday.Weekday() == time.Sunday {
		nextWeekday = nextWeekday.AddDate(0, 0, 1)
	}
	dateStr := nextWeekday.Format("2006-01-02")

	slots, err := userClient.WaitForSlots(room.ID, dateStr, maxRetries, retryDelay)
	assert.NoError(t, err)
	assert.NotEmpty(t, slots, "Should have at least one slot")

	for _, slot := range slots {
		assert.NotEqual(t, uuid.Nil, slot.ID)
		assert.Equal(t, room.ID, slot.RoomID)
		assert.False(t, slot.Start.IsZero())
		assert.False(t, slot.End.IsZero())
	}

	_, err = userClient.ListAvailableSlots(room.ID, "invalid-date")
	assert.NoError(t, err)

	slots, err = userClient.ListAvailableSlots(uuid.New(), dateStr)
	assert.NoError(t, err)
	assert.Empty(t, slots)
}

func TestBookingsCreate(t *testing.T) {
	adminClient := NewTestClient(baseURL)
	_, err := adminClient.LoginAs("admin")
	require.NoError(t, err)

	room, err := adminClient.CreateRoom("Booking Test Room", nil, nil)
	require.NoError(t, err)

	_, err = adminClient.CreateSchedule(room.ID, []int{1, 2, 3, 4, 5}, "09:00", "18:00")
	require.NoError(t, err)

	userClient := NewTestClient(baseURL)
	_, err = userClient.LoginAs("user")
	require.NoError(t, err)

	now := time.Now().UTC()
	nextWeekday := now.AddDate(0, 0, 1)
	for nextWeekday.Weekday() == time.Saturday || nextWeekday.Weekday() == time.Sunday {
		nextWeekday = nextWeekday.AddDate(0, 0, 1)
	}
	dateStr := nextWeekday.Format("2006-01-02")

	slots, err := userClient.WaitForSlots(room.ID, dateStr, maxRetries, retryDelay)
	require.NoError(t, err)
	require.NotEmpty(t, slots, "Should have at least one slot")

	booking, err := userClient.CreateBooking(slots[0].ID, false)
	assert.NoError(t, err)
	assert.NotNil(t, booking)
	assert.Equal(t, slots[0].ID, booking.SlotID)
	assert.Equal(t, domain.BookingStatusActive, booking.Status)

	_, err = userClient.CreateBooking(slots[0].ID, false)
	assert.Error(t, err)

	_, err = adminClient.CreateBooking(slots[0].ID, false)
	assert.Error(t, err)

	_, err = userClient.CreateBooking(uuid.New(), false)
	assert.Error(t, err)
}

func TestBookingsWithConferenceLink(t *testing.T) {
	adminClient := NewTestClient(baseURL)
	_, err := adminClient.LoginAs("admin")
	require.NoError(t, err)

	room, err := adminClient.CreateRoom("Conference Link Test", nil, nil)
	require.NoError(t, err)

	_, err = adminClient.CreateSchedule(room.ID, []int{1, 2, 3, 4, 5}, "09:00", "18:00")
	require.NoError(t, err)

	userClient := NewTestClient(baseURL)
	_, err = userClient.LoginAs("user")
	require.NoError(t, err)

	now := time.Now().UTC()
	nextWeekday := now.AddDate(0, 0, 1)
	for nextWeekday.Weekday() == time.Saturday || nextWeekday.Weekday() == time.Sunday {
		nextWeekday = nextWeekday.AddDate(0, 0, 1)
	}
	dateStr := nextWeekday.Format("2006-01-02")

	slots, err := userClient.WaitForSlots(room.ID, dateStr, maxRetries, retryDelay)
	require.NoError(t, err)
	require.NotEmpty(t, slots, "Should have at least one slot")

	booking, err := userClient.CreateBooking(slots[0].ID, true)
	assert.NoError(t, err)
	assert.NotNil(t, booking)
	assert.NotNil(t, booking.ConferenceLink)
	assert.NotEmpty(t, *booking.ConferenceLink)
}

func TestBookingsList(t *testing.T) {
	adminClient := NewTestClient(baseURL)
	_, err := adminClient.LoginAs("admin")
	require.NoError(t, err)

	room, err := adminClient.CreateRoom("List Test Room", nil, nil)
	require.NoError(t, err)

	_, err = adminClient.CreateSchedule(room.ID, []int{1, 2, 3, 4, 5}, "09:00", "18:00")
	require.NoError(t, err)

	userClient := NewTestClient(baseURL)
	_, err = userClient.LoginAs("user")
	require.NoError(t, err)

	now := time.Now().UTC()
	nextWeekday := now.AddDate(0, 0, 1)
	for nextWeekday.Weekday() == time.Saturday || nextWeekday.Weekday() == time.Sunday {
		nextWeekday = nextWeekday.AddDate(0, 0, 1)
	}
	dateStr := nextWeekday.Format("2006-01-02")

	slots, err := userClient.WaitForSlots(room.ID, dateStr, maxRetries, retryDelay)
	require.NoError(t, err)
	require.NotEmpty(t, slots, "Should have at least one slot")

	for i := 0; i < 3 && i < len(slots); i++ {
		_, err = userClient.CreateBooking(slots[i].ID, false)
		require.NoError(t, err)
	}

	bookings, pagination, err := adminClient.ListAllBookings(1, 2)
	assert.NoError(t, err)
	assert.NotEmpty(t, bookings)
	assert.Equal(t, 1, pagination.Page)
	assert.Equal(t, 2, pagination.PageSize)

	_, _, err = userClient.ListAllBookings(1, 10)
	assert.Error(t, err)
}

func TestMyBookings(t *testing.T) {
	adminClient := NewTestClient(baseURL)
	_, err := adminClient.LoginAs("admin")
	require.NoError(t, err)

	room, err := adminClient.CreateRoom("My Bookings Test", nil, nil)
	require.NoError(t, err)

	_, err = adminClient.CreateSchedule(room.ID, []int{1, 2, 3, 4, 5}, "09:00", "18:00")
	require.NoError(t, err)

	userClient := NewTestClient(baseURL)
	_, err = userClient.LoginAs("user")
	require.NoError(t, err)

	now := time.Now().UTC()
	nextWeekday := now.AddDate(0, 0, 1)
	for nextWeekday.Weekday() == time.Saturday || nextWeekday.Weekday() == time.Sunday {
		nextWeekday = nextWeekday.AddDate(0, 0, 1)
	}
	dateStr := nextWeekday.Format("2006-01-02")

	slots, err := userClient.WaitForSlots(room.ID, dateStr, maxRetries, retryDelay)
	require.NoError(t, err)
	require.NotEmpty(t, slots, "Should have at least one slot")

	booking, err := userClient.CreateBooking(slots[0].ID, false)
	require.NoError(t, err)

	myBookings, err := userClient.ListMyBookings()
	assert.NoError(t, err)

	found := false
	for _, b := range myBookings {
		if b.ID == booking.ID {
			found = true
			break
		}
	}
	assert.True(t, found)

	_, err = adminClient.ListMyBookings()
	assert.Error(t, err)
}

func TestCancelBooking(t *testing.T) {
	adminClient := NewTestClient(baseURL)
	_, err := adminClient.LoginAs("admin")
	require.NoError(t, err)

	room, err := adminClient.CreateRoom("Cancel Test Room", nil, nil)
	require.NoError(t, err)

	_, err = adminClient.CreateSchedule(room.ID, []int{1, 2, 3, 4, 5}, "09:00", "18:00")
	require.NoError(t, err)

	userClient := NewTestClient(baseURL)
	_, err = userClient.LoginAs("user")
	require.NoError(t, err)

	now := time.Now().UTC()
	nextWeekday := now.AddDate(0, 0, 1)
	for nextWeekday.Weekday() == time.Saturday || nextWeekday.Weekday() == time.Sunday {
		nextWeekday = nextWeekday.AddDate(0, 0, 1)
	}
	dateStr := nextWeekday.Format("2006-01-02")

	slots, err := userClient.WaitForSlots(room.ID, dateStr, maxRetries, retryDelay)
	require.NoError(t, err)
	require.NotEmpty(t, slots, "Should have at least one slot")

	booking, err := userClient.CreateBooking(slots[0].ID, false)
	require.NoError(t, err)

	cancelled, err := userClient.CancelBooking(booking.ID)
	assert.NoError(t, err)
	assert.Equal(t, domain.BookingStatusCancelled, cancelled.Status)

	cancelledAgain, err := userClient.CancelBooking(booking.ID)
	assert.NoError(t, err)
	assert.Equal(t, domain.BookingStatusCancelled, cancelledAgain.Status)

	_, err = userClient.CancelBooking(uuid.New())
	assert.Error(t, err)
}

func TestCannotCancelOtherUsersBooking(t *testing.T) {
	adminClient := NewTestClient(baseURL)
	_, err := adminClient.LoginAs("admin")
	require.NoError(t, err)

	room, err := adminClient.CreateRoom("Cancel Other User Room", nil, nil)
	require.NoError(t, err)

	_, err = adminClient.CreateSchedule(room.ID, []int{1, 2, 3, 4, 5}, "09:00", "18:00")
	require.NoError(t, err)

	user1Client := NewTestClient(baseURL)
	user1Email := fmt.Sprintf("user1_%d@example.com", time.Now().UnixNano())
	_, err = user1Client.Register(user1Email, "password123", "user")
	require.NoError(t, err)

	_, err = user1Client.Login(user1Email, "password123")
	require.NoError(t, err)

	now := time.Now().UTC()
	nextWeekday := now.AddDate(0, 0, 1)
	for nextWeekday.Weekday() == time.Saturday || nextWeekday.Weekday() == time.Sunday {
		nextWeekday = nextWeekday.AddDate(0, 0, 1)
	}
	dateStr := nextWeekday.Format("2006-01-02")

	slots, err := user1Client.WaitForSlots(room.ID, dateStr, maxRetries, retryDelay)
	require.NoError(t, err)
	require.NotEmpty(t, slots, "Should have at least one slot")

	booking, err := user1Client.CreateBooking(slots[0].ID, false)
	require.NoError(t, err)
	t.Logf("User1 created booking: %s", booking.ID)

	user2Client := NewTestClient(baseURL)
	user2Email := fmt.Sprintf("user2_%d@example.com", time.Now().UnixNano())
	_, err = user2Client.Register(user2Email, "password123", "user")
	require.NoError(t, err)

	_, err = user2Client.Login(user2Email, "password123")
	require.NoError(t, err)
	t.Logf("User2 logged in with different ID")

	_, err = user2Client.CancelBooking(booking.ID)
	assert.Error(t, err, "User2 should not be able to cancel User1's booking")

	if err != nil {
		assert.Contains(t, err.Error(), "403", "Should return 403 Forbidden")
	}

	_, err = user1Client.CancelBooking(booking.ID)
	assert.NoError(t, err, "User1 should be able to cancel own booking")
}

func TestBookingsMyOnlyFuture(t *testing.T) {
	adminClient := NewTestClient(baseURL)
	_, err := adminClient.LoginAs("admin")
	require.NoError(t, err)

	room, err := adminClient.CreateRoom("Future Only Room", nil, nil)
	require.NoError(t, err)

	_, err = adminClient.CreateSchedule(room.ID, []int{1, 2, 3, 4, 5}, "09:00", "18:00")
	require.NoError(t, err)

	userClient := NewTestClient(baseURL)
	_, err = userClient.LoginAs("user")
	require.NoError(t, err)

	now := time.Now().UTC()
	futureDate := now.AddDate(0, 0, 7)
	dateStr := futureDate.Format("2006-01-02")

	slots, err := userClient.WaitForSlots(room.ID, dateStr, maxRetries, retryDelay)
	require.NoError(t, err)
	require.NotEmpty(t, slots, "Should have at least one slot")

	booking, err := userClient.CreateBooking(slots[0].ID, false)
	require.NoError(t, err)

	myBookings, err := userClient.ListMyBookings()
	assert.NoError(t, err)

	found := false
	for _, b := range myBookings {
		if b.ID == booking.ID {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestDatabaseIntegrity(t *testing.T) {
	dbHelper, err := NewDBHelper()
	require.NoError(t, err)
	defer dbHelper.Close()

	adminID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	admin, err := dbHelper.GetUserByID(adminID)
	if err != nil {
		t.Logf("Admin test user not found in DB: %v", err)
	} else {
		assert.Equal(t, "admin", string(admin.Role))
		t.Logf("Admin user exists: %s", admin.ID)
	}

	user, err := dbHelper.GetUserByID(userID)
	if err != nil {
		t.Logf("Regular test user not found in DB: %v", err)
	} else {
		assert.Equal(t, "user", string(user.Role))
		t.Logf("Regular user exists: %s", user.ID)
	}

	indexes := []string{
		"idx_slots_room_date",
		"idx_bookings_active_slot",
		"idx_bookings_user",
	}

	for _, idx := range indexes {
		var exists bool
		query := `
			SELECT EXISTS (
				SELECT 1 FROM pg_indexes 
				WHERE indexname = $1
			)
		`
		err := dbHelper.db.QueryRow(query, idx).Scan(&exists)
		assert.NoError(t, err)
		assert.True(t, exists, "Index %s should exist", idx)
		t.Logf("Index %s exists", idx)
	}

	constraints := []string{
		"bookings_slot_id_fkey",
		"bookings_user_id_fkey",
		"schedules_room_id_fkey",
		"slots_room_id_fkey",
	}

	for _, constraint := range constraints {
		var exists bool
		query := `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.table_constraints 
				WHERE constraint_name = $1
			)
		`
		err := dbHelper.db.QueryRow(query, constraint).Scan(&exists)
		assert.NoError(t, err)
		assert.True(t, exists, "Constraint %s should exist", constraint)
		t.Logf("Constraint %s exists", constraint)
	}

	dbHelper.PrintTableStats()
}

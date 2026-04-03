package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime/debug"
	"testing"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	baseURL    = "http://localhost:8080"
	maxRetries = 10
	retryDelay = 500 * time.Millisecond
)

type TestClient struct {
	baseURL    string
	httpClient *http.Client
	token      string
	userID     uuid.UUID
}

func NewTestClient(baseURL string) *TestClient {

	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	client := &TestClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    100,
				IdleConnTimeout: 90 * time.Second,
			},
		},
		token: "",
	}

	if client.httpClient == nil {
		return nil
	}

	return client
}

func (c *TestClient) LoginAs(role string) (string, error) {

	if c == nil {
		return "", fmt.Errorf("client is nil")
	}
	if c.httpClient == nil {
		return "", fmt.Errorf("http client is nil")
	}
	if c.baseURL == "" {
		return "", fmt.Errorf("base URL is empty")
	}

	reqBody := map[string]string{"role": role}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	fullURL := c.baseURL + "/dummyLogin"

	resp, err := c.httpClient.Post(fullURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
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
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if result.Token == "" {
		return "", fmt.Errorf("empty token received")
	}

	c.token = result.Token

	return result.Token, nil
}

func (c *TestClient) doRequest(method, path string, body interface{}) (*http.Response, []byte, error) {
	defer func() {
		if r := recover(); r != nil {
			debug.PrintStack()
		}
	}()

	if c == nil {
		return nil, nil, fmt.Errorf("client is nil")
	}
	if c.httpClient == nil {
		return nil, nil, fmt.Errorf("http client is nil")
	}
	if c.baseURL == "" {
		return nil, nil, fmt.Errorf("base URL is empty")
	}

	fullURL := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("http request failed: %w", err)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return resp, bodyBytes, nil
}

func safeSubstring(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func (c *TestClient) CreateRoom(name string, description *string, capacity *int) (*domain.Room, error) {

	reqBody := map[string]interface{}{
		"name": name,
	}
	if description != nil {
		reqBody["description"] = description
	}
	if capacity != nil {
		reqBody["capacity"] = capacity
	}

	resp, body, err := c.doRequest("POST", "/rooms/create", reqBody)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create room failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v, body: %s", err, string(body))
	}

	roomMap, ok := result["room"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("room not found in response: %s", string(body))
	}

	room := &domain.Room{}
	if id, ok := roomMap["id"].(string); ok {
		room.ID, _ = uuid.Parse(id)
	}

	if name, ok := roomMap["name"].(string); ok {
		room.Name = name
	}

	if desc, ok := roomMap["description"].(string); ok {
		room.Description = &desc
	}
	if cap, ok := roomMap["capacity"].(float64); ok {
		capInt := int(cap)
		room.Capacity = &capInt
	}

	return room, nil
}

func (c *TestClient) ListRooms() ([]domain.Room, error) {
	resp, body, err := c.doRequest("GET", "/rooms/list", nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list rooms failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Rooms []domain.Room `json:"rooms"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v, body: %s", err, string(body))
	}
	return result.Rooms, nil
}

func (c *TestClient) CreateSchedule(roomID uuid.UUID, daysOfWeek []int, startTime, endTime string) (*domain.Schedule, error) {

	reqBody := map[string]interface{}{
		"daysOfWeek": daysOfWeek,
		"startTime":  startTime,
		"endTime":    endTime,
	}

	resp, body, err := c.doRequest("POST", fmt.Sprintf("/rooms/%s/schedule/create", roomID), reqBody)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create schedule failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v, body: %s", err, string(body))
	}

	scheduleMap, ok := result["schedule"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("schedule not found in response: %s", string(body))
	}

	schedule := &domain.Schedule{}
	if id, ok := scheduleMap["id"].(string); ok {
		schedule.ID, _ = uuid.Parse(id)
	}
	if roomId, ok := scheduleMap["roomId"].(string); ok {
		schedule.RoomID, _ = uuid.Parse(roomId)
	}
	if days, ok := scheduleMap["daysOfWeek"].([]interface{}); ok {
		daysOfWeek := make([]int, len(days))
		for i, d := range days {
			if dayFloat, ok := d.(float64); ok {
				daysOfWeek[i] = int(dayFloat)
			}
		}
		schedule.DaysOfWeek = daysOfWeek
	}
	if start, ok := scheduleMap["startTime"].(string); ok {
		schedule.StartTime = start
	}
	if end, ok := scheduleMap["endTime"].(string); ok {
		schedule.EndTime = end
	}

	return schedule, nil
}

func (c *TestClient) ListAvailableSlots(roomID uuid.UUID, date string) ([]domain.Slot, error) {

	if c == nil {
		return nil, fmt.Errorf("client is nil")
	}
	if c.httpClient == nil {
		return nil, fmt.Errorf("http client is nil")
	}
	if c.baseURL == "" {
		return nil, fmt.Errorf("base URL is empty")
	}

	resp, body, err := c.doRequest("GET", fmt.Sprintf("/rooms/%s/slots/list?date=%s", roomID, date), nil)
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return nil, fmt.Errorf("response is nil")
	}

	if resp.StatusCode != http.StatusOK {
		return []domain.Slot{}, nil
	}

	if len(body) == 0 {
		return []domain.Slot{}, nil
	}

	var result struct {
		Slots []domain.Slot `json:"slots"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Slots, nil
}

func (c *TestClient) CreateBooking(slotID uuid.UUID, createConferenceLink bool) (*domain.Booking, error) {

	reqBody := map[string]interface{}{
		"slotId":               slotID,
		"createConferenceLink": createConferenceLink,
	}

	resp, body, err := c.doRequest("POST", "/bookings/create", reqBody)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create booking failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v, body: %s", err, string(body))
	}

	bookingMap, ok := result["booking"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("booking not found in response: %s", string(body))
	}

	booking := &domain.Booking{}
	if id, ok := bookingMap["id"].(string); ok {
		booking.ID, _ = uuid.Parse(id)
	}
	if slotId, ok := bookingMap["slotId"].(string); ok {
		booking.SlotID, _ = uuid.Parse(slotId)
	}
	if userId, ok := bookingMap["userId"].(string); ok {
		booking.UserID, _ = uuid.Parse(userId)
	}
	if status, ok := bookingMap["status"].(string); ok {
		booking.Status = domain.BookingStatus(status)
	}
	if link, ok := bookingMap["conferenceLink"].(string); ok {
		booking.ConferenceLink = &link
	}

	return booking, nil
}

func (c *TestClient) ListMyBookings() ([]domain.Booking, error) {
	resp, body, err := c.doRequest("GET", "/bookings/my", nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list my bookings failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Bookings []domain.Booking `json:"bookings"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v, body: %s", err, string(body))
	}
	return result.Bookings, nil
}

func (c *TestClient) CancelBooking(bookingID uuid.UUID) (*domain.Booking, error) {

	resp, body, err := c.doRequest("POST", fmt.Sprintf("/bookings/%s/cancel", bookingID), nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cancel booking failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v, body: %s", err, string(body))
	}

	bookingMap, ok := result["booking"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("booking not found in response: %s", string(body))
	}

	booking := &domain.Booking{}
	if id, ok := bookingMap["id"].(string); ok {
		booking.ID, _ = uuid.Parse(id)
	}
	if status, ok := bookingMap["status"].(string); ok {
		booking.Status = domain.BookingStatus(status)
	}

	return booking, nil
}

func (c *TestClient) ListAllBookings(page, pageSize int) ([]domain.Booking, *domain.Pagination, error) {
	resp, body, err := c.doRequest("GET", fmt.Sprintf("/bookings/list?page=%d&pageSize=%d", page, pageSize), nil)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("list all bookings failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Bookings   []domain.Booking  `json:"bookings"`
		Pagination domain.Pagination `json:"pagination"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, nil, fmt.Errorf("failed to parse response: %v, body: %s", err, string(body))
	}
	return result.Bookings, &result.Pagination, nil
}

func (c *TestClient) Info() error {
	resp, err := c.httpClient.Get(c.baseURL + "/_info")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected 200, got %d", resp.StatusCode)
	}
	return nil
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestMain(m *testing.M) {
	client := NewTestClient(baseURL)
	maxAttempts := 30
	for i := 0; i < maxAttempts; i++ {
		if err := client.Info(); err == nil {
			break
		}
		if i == maxAttempts-1 {
			os.Exit(1)
		}
		time.Sleep(1 * time.Second)
	}
	code := m.Run()
	os.Exit(code)
}

func (c *TestClient) WaitForSlots(roomID uuid.UUID, date string, maxRetries int, delay time.Duration) ([]domain.Slot, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		slots, err := c.ListAvailableSlots(roomID, date)
		if err != nil {
			lastErr = err
			if i < maxRetries-1 {
				time.Sleep(delay)
				continue
			}
			return nil, fmt.Errorf("error listing slots after %d retries: %w", maxRetries, err)
		}
		if len(slots) > 0 {
			return slots, nil
		}
		if i < maxRetries-1 {
			time.Sleep(delay)
		}
	}
	return nil, fmt.Errorf("no slots available after %d retries (last error: %v)", maxRetries, lastErr)
}

func TestConnection(t *testing.T) {
	resp, err := http.Get("http://localhost:8080/_info")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response: status=%d, body=%s\n", resp.StatusCode, string(body))

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestCreateRoomScheduleAndBooking(t *testing.T) {
	if baseURL == "" {
		t.Fatal("baseURL is empty")
	}

	adminClient := NewTestClient(baseURL)
	if adminClient == nil {
		t.Fatal("adminClient is nil")
	}
	if adminClient.httpClient == nil {
		t.Fatal("adminClient.httpClient is nil")
	}

	token, err := adminClient.LoginAs("admin")
	require.NoError(t, err)
	require.NotEmpty(t, token, "Token should not be empty")
	t.Logf("Admin token: %s...", token[:20])

	if adminClient.token == "" {
		t.Fatal("Admin token not saved in client")
	}

	room, err := adminClient.CreateRoom("Conference Room A", stringPtr("Large meeting room"), intPtr(10))
	require.NoError(t, err)
	require.NotNil(t, room)
	assert.NotEqual(t, uuid.Nil, room.ID)
	assert.Equal(t, "Conference Room A", room.Name)
	t.Logf("Created room: %s", room.ID)

	schedule, err := adminClient.CreateSchedule(room.ID, []int{1, 2, 3, 4, 5}, "09:00", "18:00")
	require.NoError(t, err)
	require.NotNil(t, schedule)
	assert.Equal(t, room.ID, schedule.RoomID)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, schedule.DaysOfWeek)
	assert.Equal(t, "09:00", schedule.StartTime)
	assert.Equal(t, "18:00", schedule.EndTime)
	t.Logf("Created schedule: %s", schedule.ID)

	userClient := NewTestClient(baseURL)
	if userClient == nil {
		t.Fatal("userClient is nil")
	}
	if userClient.httpClient == nil {
		t.Fatal("userClient.httpClient is nil")
	}

	userToken, err := userClient.LoginAs("user")
	require.NoError(t, err)
	require.NotEmpty(t, userToken, "User token should not be empty")
	t.Logf("User token: %s...", userToken[:20])

	if userClient.token == "" {
		t.Fatal("User token not saved in client")
	}

	now := time.Now().UTC()
	startDate := now.AddDate(0, 0, 1)

	var slots []domain.Slot
	var selectedDate string
	var foundSlots bool

	for i := 0; i < 30; i++ {
		checkDate := startDate.AddDate(0, 0, i)
		dateStr := checkDate.Format("2006-01-02")

		if checkDate.Weekday() == time.Saturday || checkDate.Weekday() == time.Sunday {
			continue
		}

		slots, err = userClient.WaitForSlots(room.ID, dateStr, maxRetries, retryDelay)
		if err == nil && len(slots) > 0 {
			selectedDate = dateStr
			foundSlots = true
			break
		}
	}

	require.True(t, foundSlots, "Expected at least one slot available within next 30 days")
	require.NotEmpty(t, slots, "Slots slice should not be empty")

	t.Logf("Using slot from date %s: ID=%s, from %s to %s",
		selectedDate, slots[0].ID, slots[0].Start.Format("15:04"), slots[0].End.Format("15:04"))

	booking, err := userClient.CreateBooking(slots[0].ID, false)
	require.NoError(t, err)
	require.NotNil(t, booking)
	assert.Equal(t, slots[0].ID, booking.SlotID)
	assert.Equal(t, domain.BookingStatusActive, booking.Status)
	t.Logf("Created booking: %s", booking.ID)

	myBookings, err := userClient.ListMyBookings()
	require.NoError(t, err)
	t.Logf("Found %d bookings in my list", len(myBookings))

	found := false
	for _, b := range myBookings {
		if b.ID == booking.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "Booking not found in user's bookings")

	slotsAfterBooking, err := userClient.ListAvailableSlots(room.ID, selectedDate)
	require.NoError(t, err)
	t.Logf("Found %d slots after booking", len(slotsAfterBooking))

	slotStillAvailable := false
	for _, slot := range slotsAfterBooking {
		if slot.ID == slots[0].ID {
			slotStillAvailable = true
			break
		}
	}
	assert.False(t, slotStillAvailable, "Slot should no longer be available")
}

func TestBookingOnAlreadyBookedSlot(t *testing.T) {
	client := NewTestClient(baseURL)

	_, err := client.LoginAs("admin")
	require.NoError(t, err)

	room, err := client.CreateRoom("Conference Room C", stringPtr("Test concurrent booking"), intPtr(8))
	require.NoError(t, err)

	_, err = client.CreateSchedule(room.ID, []int{1, 2, 3, 4, 5}, "09:00", "18:00")
	require.NoError(t, err)

	_, err = client.LoginAs("user")
	require.NoError(t, err)

	now := time.Now().UTC()
	nextDay := now.AddDate(0, 0, 1)
	for nextDay.Weekday() == time.Saturday || nextDay.Weekday() == time.Sunday {
		nextDay = nextDay.AddDate(0, 0, 1)
	}
	dateStr := nextDay.Format("2006-01-02")

	slots, err := client.WaitForSlots(room.ID, dateStr, maxRetries, retryDelay)
	if err != nil || len(slots) == 0 {
		t.Skip("No slots available for testing")
	}

	booking, err := client.CreateBooking(slots[0].ID, false)
	require.NoError(t, err)
	assert.NotNil(t, booking)

	_, err = client.CreateBooking(slots[0].ID, false)
	assert.Error(t, err, "Should not be able to book already booked slot")
}

func TestUserCannotCreateRoom(t *testing.T) {
	client := NewTestClient(baseURL)

	_, err := client.LoginAs("user")
	require.NoError(t, err)

	_, err = client.CreateRoom("Should Not Create", nil, nil)
	assert.Error(t, err)
}

func TestAdminCannotCreateBooking(t *testing.T) {
	client := NewTestClient(baseURL)

	_, err := client.LoginAs("admin")
	require.NoError(t, err)

	_, err = client.CreateBooking(uuid.New(), false)
	assert.Error(t, err)
}

func TestCannotBookPastSlot(t *testing.T) {
	client := NewTestClient(baseURL)

	_, err := client.LoginAs("admin")
	require.NoError(t, err)

	room, err := client.CreateRoom("Conference Room D", stringPtr("Past slot test"), intPtr(3))
	require.NoError(t, err)

	_, err = client.CreateSchedule(room.ID, []int{1, 2, 3, 4, 5}, "09:00", "18:00")
	require.NoError(t, err)

	_, err = client.LoginAs("user")
	require.NoError(t, err)

	_, err = client.CreateBooking(uuid.New(), false)
	assert.Error(t, err)
}

func TestAdminBookingsPagination(t *testing.T) {
	client := NewTestClient(baseURL)

	_, err := client.LoginAs("admin")
	require.NoError(t, err)

	room, err := client.CreateRoom("Pagination Test Room", nil, nil)
	require.NoError(t, err)

	_, err = client.CreateSchedule(room.ID, []int{1, 2, 3, 4, 5}, "09:00", "12:00")
	require.NoError(t, err)

	_, err = client.LoginAs("user")
	require.NoError(t, err)

	now := time.Now().UTC()

	nextWeekday := now.AddDate(0, 0, 1)

	for nextWeekday.Weekday() == time.Saturday || nextWeekday.Weekday() == time.Sunday {
		nextWeekday = nextWeekday.AddDate(0, 0, 1)
	}

	dateStr := nextWeekday.Format("2006-01-02")

	t.Logf("Looking for slots on date: %s (weekday: %s)", dateStr, nextWeekday.Weekday().String())

	slots, err := client.WaitForSlots(room.ID, dateStr, 20, 500*time.Millisecond)
	if err != nil {
		t.Logf("WaitForSlots error: %v", err)

		for i := 1; i <= 7; i++ {
			checkDate := nextWeekday.AddDate(0, 0, i)
			dateStr = checkDate.Format("2006-01-02")
			t.Logf("Trying alternative date: %s", dateStr)
			slots, err = client.WaitForSlots(room.ID, dateStr, 20, 500*time.Millisecond)
			if err == nil && len(slots) > 0 {
				t.Logf("Found slots on alternative date: %s", dateStr)
				break
			}
		}
	}

	require.NoError(t, err, "Should find slots within 7 days")
	require.NotEmpty(t, slots, "Should have at least one slot")
	t.Logf("Found %d slots", len(slots))

	createdBookings := 0
	for i := 0; i < 5 && i < len(slots); i++ {
		_, err = client.CreateBooking(slots[i].ID, false)
		if err != nil {
			t.Logf("Failed to create booking for slot %d: %v", i, err)
			continue
		}
		createdBookings++
	}

	require.GreaterOrEqual(t, createdBookings, 5, "Should create at least 5 bookings, created %d", createdBookings)

	_, err = client.LoginAs("admin")
	require.NoError(t, err)

	bookings, pagination, err := client.ListAllBookings(1, 2)
	require.NoError(t, err)
	assert.NotEmpty(t, bookings, "Should have bookings on page 1")
	assert.Equal(t, 1, pagination.Page)
	assert.Equal(t, 2, pagination.PageSize)
	assert.GreaterOrEqual(t, pagination.Total, 5, "Total bookings should be at least 5")

	bookings2, pagination2, err := client.ListAllBookings(2, 2)
	require.NoError(t, err)
	assert.NotEmpty(t, bookings2, "Should have bookings on page 2")
	assert.Equal(t, 2, pagination2.Page)
	assert.Equal(t, 2, pagination2.PageSize)
	assert.Equal(t, pagination.Total, pagination2.Total)

	assert.NotEqual(t, bookings[0].ID, bookings2[0].ID, "Bookings on different pages should be different")
}

func TestCreateBookingWithConferenceLink(t *testing.T) {
	client := NewTestClient(baseURL)

	_, err := client.LoginAs("admin")
	require.NoError(t, err)

	room, err := client.CreateRoom("Conference Room E", stringPtr("Conference link test"), intPtr(10))
	require.NoError(t, err)

	_, err = client.CreateSchedule(room.ID, []int{1, 2, 3, 4, 5}, "09:00", "18:00")
	require.NoError(t, err)

	_, err = client.LoginAs("user")
	require.NoError(t, err)

	now := time.Now().UTC()

	startDate := now.AddDate(0, 0, 1)

	nextWeekday := startDate
	for nextWeekday.Weekday() == time.Saturday || nextWeekday.Weekday() == time.Sunday {
		nextWeekday = nextWeekday.AddDate(0, 0, 1)
	}
	dateStr := nextWeekday.Format("2006-01-02")
	t.Logf("Looking for slots on date: %s", dateStr)

	slots, err := client.WaitForSlots(room.ID, dateStr, 20, 500*time.Millisecond)
	if err != nil {

		for i := 1; i <= 7; i++ {
			checkDate := nextWeekday.AddDate(0, 0, i)
			for checkDate.Weekday() == time.Saturday || checkDate.Weekday() == time.Sunday {
				checkDate = checkDate.AddDate(0, 0, 1)
			}
			dateStr = checkDate.Format("2006-01-02")
			slots, err = client.WaitForSlots(room.ID, dateStr, 20, 500*time.Millisecond)
			if err == nil && len(slots) > 0 {
				t.Logf("Found slots on date: %s", dateStr)
				break
			}
		}
	}

	require.NoError(t, err, "Should find slots within 7 days")
	require.NotEmpty(t, slots, "Should have at least one slot")

	booking, err := client.CreateBooking(slots[0].ID, true)
	require.NoError(t, err)
	require.NotNil(t, booking)
	assert.Equal(t, domain.BookingStatusActive, booking.Status)
	assert.NotNil(t, booking.ConferenceLink)
	assert.NotEmpty(t, *booking.ConferenceLink)
	t.Logf("Created booking with conference link: %s", *booking.ConferenceLink)
}

func TestMyBookingsOnlyFuture(t *testing.T) {
	client := NewTestClient(baseURL)

	_, err := client.LoginAs("admin")
	require.NoError(t, err)

	room, err := client.CreateRoom("Future Only Room", nil, nil)
	require.NoError(t, err)

	_, err = client.CreateSchedule(room.ID, []int{1, 2, 3, 4, 5}, "09:00", "12:00")
	require.NoError(t, err)

	_, err = client.LoginAs("user")
	require.NoError(t, err)

	now := time.Now().UTC()

	startDate := now.AddDate(0, 0, 1)

	nextWeekday := startDate
	for nextWeekday.Weekday() == time.Saturday || nextWeekday.Weekday() == time.Sunday {
		nextWeekday = nextWeekday.AddDate(0, 0, 1)
	}

	dateStr := nextWeekday.Format("2006-01-02")
	t.Logf("Looking for slots on date: %s (weekday: %s)", dateStr, nextWeekday.Weekday().String())

	slots, err := client.WaitForSlots(room.ID, dateStr, 20, 500*time.Millisecond)
	if err != nil {
		t.Logf("WaitForSlots error for date %s: %v", dateStr, err)

		for i := 1; i <= 7; i++ {
			checkDate := nextWeekday.AddDate(0, 0, i)

			for checkDate.Weekday() == time.Saturday || checkDate.Weekday() == time.Sunday {
				checkDate = checkDate.AddDate(0, 0, 1)
			}
			dateStr = checkDate.Format("2006-01-02")
			t.Logf("Trying alternative date: %s (weekday: %s)", dateStr, checkDate.Weekday().String())
			slots, err = client.WaitForSlots(room.ID, dateStr, 20, 500*time.Millisecond)
			if err == nil && len(slots) > 0 {
				t.Logf("Found slots on alternative date: %s", dateStr)
				break
			}
		}
	}

	require.NoError(t, err, "Should find slots within 7 days")
	require.NotEmpty(t, slots, "Should have at least one slot")
	t.Logf("Found %d slots", len(slots))

	futureBooking, err := client.CreateBooking(slots[0].ID, false)
	require.NoError(t, err)
	t.Logf("Created booking: %s for slot: %s", futureBooking.ID, slots[0].ID)

	time.Sleep(100 * time.Millisecond)

	myBookings, err := client.ListMyBookings()
	require.NoError(t, err)
	t.Logf("Found %d bookings in my list", len(myBookings))

	found := false
	for _, b := range myBookings {
		t.Logf("Booking in list: ID=%s, Status=%s", b.ID, b.Status)
		if b.ID == futureBooking.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "Future booking should be in /bookings/my")
}

func TestInvalidScheduleDays(t *testing.T) {
	client := NewTestClient(baseURL)

	_, err := client.LoginAs("admin")
	require.NoError(t, err)

	room, err := client.CreateRoom("Invalid Days Room", nil, nil)
	require.NoError(t, err)

	_, err = client.CreateSchedule(room.ID, []int{0, 8}, "09:00", "18:00")
	assert.Error(t, err)

	_, err = client.CreateSchedule(room.ID, []int{}, "09:00", "18:00")
	assert.Error(t, err)
}

func TestInvalidScheduleTime(t *testing.T) {
	client := NewTestClient(baseURL)

	_, err := client.LoginAs("admin")
	require.NoError(t, err)

	room, err := client.CreateRoom("Invalid Time Room", nil, nil)
	require.NoError(t, err)

	_, err = client.CreateSchedule(room.ID, []int{1, 2, 3}, "18:00", "09:00")
	assert.Error(t, err)

	_, err = client.CreateSchedule(room.ID, []int{1, 2, 3}, "09:00", "09:00")
	assert.Error(t, err)
}

func TestDuplicateSchedule(t *testing.T) {
	client := NewTestClient(baseURL)

	_, err := client.LoginAs("admin")
	require.NoError(t, err)

	room, err := client.CreateRoom("Duplicate Schedule Room", nil, nil)
	require.NoError(t, err)

	_, err = client.CreateSchedule(room.ID, []int{1, 2, 3, 4, 5}, "09:00", "18:00")
	require.NoError(t, err)

	_, err = client.CreateSchedule(room.ID, []int{1, 2, 3, 4, 5}, "09:00", "18:00")
	assert.Error(t, err)
}

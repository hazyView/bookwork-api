package mockdb

import (
	"context"
	"database/sql"
	"time"

	"bookwork-api/internal/models"

	"github.com/google/uuid"
)

type mockResult struct{}

func (m *mockResult) LastInsertId() (int64, error) { return 1, nil }
func (m *mockResult) RowsAffected() (int64, error) { return 1, nil }

// MockDB implements the same methods as database.DB
type MockDB struct {
	store *MockStore
}

// NewMockDB creates a new mock database
func NewMockDB() *MockDB {
	return &MockDB{
		store: NewMockStore(),
	}
}

// Close implements the database.DB Close method
func (m *MockDB) Close() error {
	return nil
}

// Ping implements the database.DB Ping method
func (m *MockDB) Ping() error {
	return nil
}

// BeginTx implements the database.DB BeginTx method
func (m *MockDB) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return &sql.Tx{}, nil
}

// QueryRowContext provides mock database query support
func (m *MockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return &sql.Row{}
}

// QueryContext provides mock query support with context
func (m *MockDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return &sql.Rows{}, nil
}

// ExecContext provides mock exec support with context
func (m *MockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return &mockResult{}, nil
}

// DB returns the underlying sql.DB (nil for mock)
var _ *sql.DB

func (m *MockDB) GetSQLDB() *sql.DB {
	return nil
}

// MockStore provides an in-memory data store for the mock database
type MockStore struct {
	users         map[uuid.UUID]*models.User
	clubs         map[uuid.UUID]*models.Club
	clubMembers   map[uuid.UUID]*models.ClubMember
	events        map[uuid.UUID]*models.Event
	eventItems    map[uuid.UUID]*models.EventItem
	availability  map[uuid.UUID]*models.Availability
	refreshTokens map[uuid.UUID]*models.RefreshToken
}

// NewMockStore creates a new mock store with pre-populated data
func NewMockStore() *MockStore {
	store := &MockStore{
		users:         make(map[uuid.UUID]*models.User),
		clubs:         make(map[uuid.UUID]*models.Club),
		clubMembers:   make(map[uuid.UUID]*models.ClubMember),
		events:        make(map[uuid.UUID]*models.Event),
		eventItems:    make(map[uuid.UUID]*models.EventItem),
		availability:  make(map[uuid.UUID]*models.Availability),
		refreshTokens: make(map[uuid.UUID]*models.RefreshToken),
	}

	store.populateTestData()
	return store
}

// populateTestData creates comprehensive test data for the demo
func (m *MockStore) populateTestData() {
	now := time.Now().UTC()

	// Create test users (25+)
	users := []struct {
		id       uuid.UUID
		name     string
		email    string
		password string
		phone    *string
		avatar   *string
		role     string
		isActive bool
	}{
		{uuid.New(), "Admin User", "admin@bookwork.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0101"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=admin"), "admin", true},
		{uuid.New(), "Emma Thompson", "emma.thompson@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0102"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=emma"), "moderator", true},
		{uuid.New(), "James Rodriguez", "james.rodriguez@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0103"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=james"), "member", true},
		{uuid.New(), "Sophia Chen", "sophia.chen@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0104"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=sophia"), "member", true},
		{uuid.New(), "Michael Johnson", "michael.johnson@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0105"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=michael"), "member", true},
		{uuid.New(), "Isabella Davis", "isabella.davis@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0106"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=isabella"), "member", true},
		{uuid.New(), "William Brown", "william.brown@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0107"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=william"), "member", true},
		{uuid.New(), "Olivia Wilson", "olivia.wilson@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0108"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=olivia"), "member", true},
		{uuid.New(), "Benjamin Garcia", "benjamin.garcia@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0109"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=benjamin"), "member", true},
		{uuid.New(), "Charlotte Martinez", "charlotte.martinez@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0110"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=charlotte"), "member", true},
		{uuid.New(), "Alexander Lee", "alexander.lee@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0111"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=alexander"), "member", true},
		{uuid.New(), "Amelia White", "amelia.white@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0112"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=amelia"), "member", true},
		{uuid.New(), "Daniel Harris", "daniel.harris@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0113"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=daniel"), "member", true},
		{uuid.New(), "Harper Clark", "harper.clark@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0114"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=harper"), "member", true},
		{uuid.New(), "Matthew Lewis", "matthew.lewis@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0115"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=matthew"), "member", true},
		{uuid.New(), "Evelyn Robinson", "evelyn.robinson@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0116"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=evelyn"), "member", true},
		{uuid.New(), "Henry Walker", "henry.walker@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0117"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=henry"), "member", true},
		{uuid.New(), "Avery Hall", "avery.hall@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0118"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=avery"), "member", true},
		{uuid.New(), "Sebastian Allen", "sebastian.allen@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0119"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=sebastian"), "member", true},
		{uuid.New(), "Luna Young", "luna.young@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0120"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=luna"), "member", true},
		{uuid.New(), "Jackson King", "jackson.king@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0121"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=jackson"), "member", true},
		{uuid.New(), "Scarlett Scott", "scarlett.scott@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0122"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=scarlett"), "member", true},
		{uuid.New(), "Lucas Green", "lucas.green@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0123"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=lucas"), "member", true},
		{uuid.New(), "Grace Adams", "grace.adams@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0124"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=grace"), "member", true},
		{uuid.New(), "Elijah Baker", "elijah.baker@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0125"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=elijah"), "member", true},
		{uuid.New(), "Zoe Hill", "zoe.hill@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0126"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=zoe"), "member", true},
		{uuid.New(), "Owen Nelson", "owen.nelson@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0127"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=owen"), "member", true},
		{uuid.New(), "Lily Carter", "lily.carter@email.com", "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", stringPtr("+1-555-0128"), stringPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=lily"), "member", true},
	}

	// Create user records
	userList := make([]*models.User, 0, len(users))
	for _, userData := range users {
		user := &models.User{
			ID:           userData.id,
			Name:         userData.name,
			Email:        userData.email,
			PasswordHash: userData.password,
			Phone:        userData.phone,
			Avatar:       userData.avatar,
			Role:         userData.role,
			IsActive:     userData.isActive,
			LastLoginAt:  &now,
			CreatedAt:    now.Add(-time.Duration(len(userList)*24) * time.Hour),
			UpdatedAt:    now,
		}
		m.users[user.ID] = user
		userList = append(userList, user)
	}

	// Create book clubs (8 clubs)
	clubs := []struct {
		name        string
		description string
		ownerIdx    int
		isPublic    bool
		maxMembers  *int
		frequency   *string
		currentBook *string
		tags        []string
		location    *string
	}{
		{
			"Classic Literature Society",
			"A club dedicated to exploring timeless literary masterpieces from around the world",
			0, true, intPtr(25), stringPtr("Monthly"), stringPtr("Pride and Prejudice"),
			[]string{"classics", "literary fiction", "discussion"}, stringPtr("Downtown Library"),
		},
		{
			"Sci-Fi Enthusiasts",
			"Dive into the future with us as we explore science fiction novels that challenge imagination",
			1, true, intPtr(20), stringPtr("Bi-weekly"), stringPtr("Dune"),
			[]string{"science fiction", "fantasy", "futuristic"}, stringPtr("Community Center"),
		},
		{
			"Mystery & Thriller Club",
			"For those who love page-turners, plot twists, and edge-of-your-seat suspense",
			2, true, intPtr(18), stringPtr("Weekly"), stringPtr("Gone Girl"),
			[]string{"mystery", "thriller", "suspense"}, stringPtr("Coffee House Main St"),
		},
		{
			"Young Adult Reading Circle",
			"A vibrant community exploring the best in young adult literature and coming-of-age stories",
			3, true, intPtr(30), stringPtr("Twice Monthly"), stringPtr("The Seven Husbands of Evelyn Hugo"),
			[]string{"young adult", "contemporary", "romance"}, stringPtr("Youth Center"),
		},
		{
			"Historical Fiction Readers",
			"Journey through different eras and cultures via compelling historical narratives",
			4, true, intPtr(22), stringPtr("Monthly"), stringPtr("All Quiet on the Western Front"),
			[]string{"historical fiction", "history", "culture"}, stringPtr("Historical Society"),
		},
		{
			"Business & Self-Help Book Club",
			"Focus on personal and professional development through insightful non-fiction reads",
			5, false, intPtr(15), stringPtr("Monthly"), stringPtr("Atomic Habits"),
			[]string{"business", "self-help", "productivity"}, stringPtr("Business District Conference Room"),
		},
		{
			"Poetry & Literary Arts",
			"Celebrate the beauty of language through poetry, literary essays, and experimental fiction",
			6, true, intPtr(12), stringPtr("Monthly"), stringPtr("Milk and Honey"),
			[]string{"poetry", "literary arts", "experimental"}, stringPtr("Arts Center"),
		},
		{
			"International Literature Club",
			"Explore diverse voices and stories from authors around the globe",
			7, true, intPtr(20), stringPtr("Monthly"), stringPtr("One Hundred Years of Solitude"),
			[]string{"international", "translated works", "cultural"}, stringPtr("International Community Center"),
		},
	}

	clubList := make([]*models.Club, 0, len(clubs))
	for _, clubData := range clubs {
		club := &models.Club{
			ID:               uuid.New(),
			Name:             clubData.name,
			Description:      clubData.description,
			OwnerID:          userList[clubData.ownerIdx].ID,
			MemberCount:      0, // Will be calculated
			IsPublic:         clubData.isPublic,
			MaxMembers:       clubData.maxMembers,
			MeetingFrequency: clubData.frequency,
			CurrentBook:      clubData.currentBook,
			Tags:             models.StringArray(clubData.tags),
			Location:         clubData.location,
			CreatedAt:        now.Add(-time.Duration(len(clubList)*15) * 24 * time.Hour),
			UpdatedAt:        now,
		}
		m.clubs[club.ID] = club
		clubList = append(clubList, club)
	}

	// Create club memberships (distribute users across clubs)
	membershipCount := 0
	for i, user := range userList {
		// Each user joins 1-3 clubs
		numClubs := 1 + (i % 3)
		clubsJoined := 0
		for j := 0; j < len(clubList) && clubsJoined < numClubs; j++ {
			if (i+j)%4 == 0 { // Some distribution logic
				role := "member"
				if j == 0 && i < 3 { // Some users are moderators
					role = "moderator"
				}

				membership := &models.ClubMember{
					ID:         uuid.New(),
					ClubID:     clubList[j].ID,
					UserID:     user.ID,
					Role:       role,
					JoinedDate: now.Add(-time.Duration((membershipCount%30)+1) * 24 * time.Hour),
					BooksRead:  (membershipCount % 15) + 1,
					IsActive:   true,
					User:       user,
				}
				m.clubMembers[membership.ID] = membership

				// Update club member count
				clubList[j].MemberCount++

				membershipCount++
				clubsJoined++
			}
		}
	}

	// Create events for clubs (25+ events)
	eventTypes := []string{"book_discussion", "author_meetup", "book_swap", "literary_workshop", "social_gathering"}
	eventCount := 0
	for i, club := range clubList {
		// Each club has 3-4 events
		numEvents := 3 + (i % 2)
		for j := 0; j < numEvents; j++ {
			eventDate := now.Add(time.Duration((eventCount%20)+1) * 24 * time.Hour)
			event := &models.Event{
				ID:           uuid.New(),
				ClubID:       club.ID,
				Title:        getEventTitle(club.Name, eventTypes[eventCount%len(eventTypes)]),
				Description:  stringPtr(getEventDescription(eventTypes[eventCount%len(eventTypes)])),
				Date:         eventDate.Format("2006-01-02"),
				Time:         getEventTime(eventCount),
				Location:     getEventLocation(club.Location, eventCount),
				Book:         club.CurrentBook,
				Type:         eventTypes[eventCount%len(eventTypes)],
				MaxAttendees: intPtr(15 + (eventCount % 10)),
				IsPublic:     club.IsPublic,
				CreatedBy:    club.OwnerID,
				Attendees:    models.UUIDArray{},
				CreatedAt:    now.Add(-time.Duration(eventCount) * time.Hour),
				UpdatedAt:    now,
			}

			// Add some attendees
			attendeeCount := (eventCount % 8) + 2
			for k := 0; k < attendeeCount && k < len(userList); k++ {
				if (eventCount+k)%3 == 0 {
					event.Attendees = append(event.Attendees, userList[k].ID)
				}
			}

			m.events[event.ID] = event
			eventCount++
		}
	}

	// Create event items (coordination items for events)
	itemCategories := []string{"Food & Beverages", "Setup & Logistics", "Materials", "Technology", "Cleanup"}
	itemNames := map[string][]string{
		"Food & Beverages":  {"Coffee & Tea", "Snacks", "Lunch Catering", "Water Bottles", "Dessert"},
		"Setup & Logistics": {"Chairs Setup", "Table Arrangement", "Sign-in Table", "Parking Coordination", "Welcome Signage"},
		"Materials":         {"Name Tags", "Discussion Guides", "Notebooks", "Pens & Markers", "Book Copies"},
		"Technology":        {"Microphone Setup", "Projector", "WiFi Password", "Camera for Photos", "Sound System"},
		"Cleanup":           {"Trash Collection", "Chair Stacking", "Equipment Return", "Venue Cleanup", "Leftover Management"},
	}
	itemStatuses := []string{"pending", "assigned", "in_progress", "completed"}

	itemCount := 0
	for _, event := range m.events {
		// Each event has 3-5 coordination items
		numItems := 3 + (itemCount % 3)
		for j := 0; j < numItems; j++ {
			category := itemCategories[itemCount%len(itemCategories)]
			names := itemNames[category]

			item := &models.EventItem{
				ID:         uuid.New(),
				EventID:    event.ID,
				Name:       names[itemCount%len(names)],
				Category:   category,
				AssignedTo: getRandomAssignee(userList, itemCount),
				Status:     itemStatuses[itemCount%len(itemStatuses)],
				Notes:      getItemNotes(itemCount),
				CreatedBy:  event.CreatedBy,
				CreatedAt:  event.CreatedAt.Add(time.Duration(j) * time.Hour),
				UpdatedAt:  now,
			}

			m.eventItems[item.ID] = item
			itemCount++
		}
	}

	// Create availability records
	availabilityCount := 0
	availabilityStatuses := []string{"available", "maybe", "unavailable"}
	for _, event := range m.events {
		// Random users provide availability for each event
		numResponses := (availabilityCount % 8) + 3
		for j := 0; j < numResponses && j < len(userList); j++ {
			if (availabilityCount+j)%3 == 0 {
				availability := &models.Availability{
					ID:        uuid.New(),
					EventID:   event.ID,
					UserID:    userList[j].ID,
					Status:    availabilityStatuses[availabilityCount%len(availabilityStatuses)],
					Notes:     getAvailabilityNotes(availabilityStatuses[availabilityCount%len(availabilityStatuses)], availabilityCount),
					UpdatedAt: now.Add(-time.Duration(availabilityCount%48) * time.Hour),
				}

				m.availability[availability.ID] = availability
				availabilityCount++
			}
		}
	}
}

// Helper functions for generating test data

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func getEventTitle(clubName, eventType string) string {
	switch eventType {
	case "book_discussion":
		return clubName + " - Monthly Book Discussion"
	case "author_meetup":
		return clubName + " - Author Meet & Greet"
	case "book_swap":
		return clubName + " - Book Exchange Event"
	case "literary_workshop":
		return clubName + " - Writing Workshop"
	case "social_gathering":
		return clubName + " - Social Hour"
	default:
		return clubName + " - Club Meeting"
	}
}

func getEventDescription(eventType string) string {
	descriptions := map[string]string{
		"book_discussion":   "Join us for an engaging discussion of this month's selected book. Come prepared with your thoughts, questions, and favorite quotes to share!",
		"author_meetup":     "A special opportunity to meet and interact with a featured author. Enjoy insights into their writing process and ask your burning questions.",
		"book_swap":         "Bring books you've finished reading and discover new treasures from fellow book lovers. A great way to expand your library sustainably!",
		"literary_workshop": "Hands-on workshop focusing on various aspects of creative writing. Perfect for aspiring authors and seasoned writers alike.",
		"social_gathering":  "Casual meetup to connect with fellow book enthusiasts. Light refreshments will be provided as we chat about our latest reads.",
	}
	if desc, exists := descriptions[eventType]; exists {
		return desc
	}
	return "Join us for another exciting club gathering!"
}

func getEventTime(count int) string {
	times := []string{"18:00", "19:00", "14:00", "15:30", "16:00", "17:30"}
	return times[count%len(times)]
}

func getEventLocation(clubLocation *string, count int) string {
	if clubLocation != nil {
		return *clubLocation
	}
	locations := []string{
		"Main Library Conference Room",
		"Community Center Hall A",
		"Downtown Coffee Shop",
		"University Meeting Room",
		"Public Library Study Room",
		"Bookstore Event Space",
	}
	return locations[count%len(locations)]
}

func getRandomAssignee(users []*models.User, count int) *uuid.UUID {
	if count%4 == 0 { // 25% chance of no assignee
		return nil
	}
	return &users[count%len(users)].ID
}

func getItemNotes(count int) *string {
	notes := []string{
		"Please coordinate with venue management",
		"Budget approved for up to $50",
		"Contact Sarah for supplier information",
		"Needs to be completed by 2 hours before event",
		"Volunteers available to help with setup",
		"Check dietary restrictions with attendees",
		"Ensure backup plans are in place",
		"Previous volunteer has contact details",
	}
	if count%3 == 0 {
		return nil
	}
	return stringPtr(notes[count%len(notes)])
}

func getAvailabilityNotes(status string, count int) *string {
	notesByStatus := map[string][]string{
		"available": {
			"Looking forward to this event!",
			"Can help with setup if needed",
			"Will arrive 15 minutes early",
			"Bringing a friend who's interested in joining",
		},
		"maybe": {
			"Depends on work schedule",
			"Will confirm by end of week",
			"Have a potential conflict, checking calendar",
			"50/50 chance - family commitment might come up",
		},
		"unavailable": {
			"Out of town that weekend",
			"Prior family commitment",
			"Work conference conflict",
			"Will catch the next one!",
		},
	}

	if count%4 == 0 {
		return nil
	}

	if notes, exists := notesByStatus[status]; exists {
		return stringPtr(notes[count%len(notes)])
	}
	return nil
}

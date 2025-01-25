package globals

import (
	"sync"

	"github.com/gofiber/fiber/v2/middleware/session"
)

var Store = session.New() // Initialize session store
// GlobalStruct is a structure that will be accessed globally
type LoggedInUser struct {
	UserID         string
	UserName       string
	Role           string
	FevRestaurants string
	FevFoods       string
}

type SelectedAddress struct {
	AddressName string
	Longitude   float64
	Latitude    float64
}

var (
	SelectedUsersAddress = SelectedAddress{
		AddressName: "Second Address",
		Longitude:   72.979,
		Latitude:    19.013,
	}
	mu sync.Mutex
)

var (
	CurrentLoggedInUser = LoggedInUser{
		UserID:   "12345",
		UserName: "john_doe",
		Role:     "admin",
	}
)

// UpdateUser updates the global struct safely
func UpdateUser(id, name, role string) {
	mu.Lock()
	defer mu.Unlock()
	CurrentLoggedInUser.UserID = id
	CurrentLoggedInUser.UserName = name
	CurrentLoggedInUser.Role = role
}

// UpdateUser updates the global struct safely
func UpdateSelectedAddress(name string, lat, long float64) {
	mu.Lock()
	defer mu.Unlock()
	SelectedUsersAddress.AddressName = name
	SelectedUsersAddress.Latitude = lat
	SelectedUsersAddress.Longitude = long
}

// GetUser retrieves the current global struct safely
func GetUser() LoggedInUser {
	mu.Lock()
	defer mu.Unlock()
	return CurrentLoggedInUser
}

// UpdateUserID updates only the UserID field in the global struct
func UpdateUserID(newUserID string) {
	CurrentLoggedInUser.UserID = newUserID
}
func UpdateLoggedInUser(newUserID, newUserName, newRole, fevRest, fevFood string) {
	mu.Lock()
	defer mu.Unlock()
	CurrentLoggedInUser.UserID = newUserID
	CurrentLoggedInUser.UserName = newUserName
	CurrentLoggedInUser.Role = newRole
	CurrentLoggedInUser.FevRestaurants = fevRest
	CurrentLoggedInUser.FevFoods = fevFood
}
func GetLoogedInUserId() string {
	mu.Lock()
	defer mu.Unlock()
	return CurrentLoggedInUser.UserID
}

func GetSelectedAddLatLong() (float64, float64) {
	mu.Lock()
	defer mu.Unlock()
	return 72.979, 19.013
}

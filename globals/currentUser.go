package globals

import "sync"

// GlobalStruct is a structure that will be accessed globally
type LoggedInUser struct {
	UserID   string
	UserName string
	Role     string
}

var (
	CurrentLoggedInUser = LoggedInUser{
		UserID:   "12345",
		UserName: "john_doe",
		Role:     "admin",
	}
	mu sync.Mutex
)

// UpdateUser updates the global struct safely
func UpdateUser(id, name, role string) {
	mu.Lock()
	defer mu.Unlock()
	CurrentLoggedInUser.UserID = id
	CurrentLoggedInUser.UserName = name
	CurrentLoggedInUser.Role = role
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

func GetLoogedInUserId() string {
	mu.Lock()
	defer mu.Unlock()
	return CurrentLoggedInUser.UserID
}

package game

import (
	"encoding/json"
	"sync"
)

// User tracks player inventory and resources.
type User struct {
	mu sync.RWMutex

	Loaded bool

	Credits    int
	PLT        int
	Experience int
	Honor      int
	Level      int
	BootyKeys  int

	Lasers  map[string]int
	Rockets map[string]int
	Energy  map[string]int
}

// NewUser creates a new user tracker.
func NewUser() *User {
	return &User{
		Lasers: map[string]int{
			"RLX_1":    0,
			"GLX_2":    0,
			"BLX_3":    0,
			"WLX_4":    0,
			"GLX_2_AS": 0,
			"MRS_6X":   0,
		},
		Rockets: map[string]int{
			"KEP_410": 0,
			"NC_30":   0,
			"TNC_130": 0,
		},
		Energy: map[string]int{
			"EE": 0,
			"EN": 0,
			"EG": 0,
			"EM": 0,
		},
	}
}

// HandleUserInfo processes a UserInfoResponsePacket payload.
func (u *User) HandleUserInfo(payload *UserInfoResponsePayload) {
	u.mu.Lock()
	defer u.mu.Unlock()

	for _, param := range payload.Params {
		var dataInt int
		json.Unmarshal(param.Data, &dataInt)

		switch param.ID {
		case 3:
			u.Credits = dataInt
		case 4:
			u.PLT = dataInt
		case 5:
			u.Experience = dataInt
		case 6:
			u.Honor = dataInt
		case 7:
			u.Level = dataInt
		case 43:
			u.BootyKeys = dataInt
		case 8: // Lasers
			switch param.Type {
			case 1:
				u.Lasers["RLX_1"] = dataInt
			case 2:
				u.Lasers["GLX_2"] = dataInt
			case 3:
				u.Lasers["BLX_3"] = dataInt
			case 4:
				u.Lasers["WLX_4"] = dataInt
			case 5:
				u.Lasers["GLX_2_AS"] = dataInt
			case 6:
				u.Lasers["MRS_6X"] = dataInt
			}
		case 9: // Rockets
			switch param.Type {
			case 1:
				u.Rockets["KEP_410"] = dataInt
			case 2:
				u.Rockets["NC_30"] = dataInt
			case 3:
				u.Rockets["TNC_130"] = dataInt
			}
		case 10: // Energy
			switch param.Type {
			case 1:
				u.Energy["EE"] = dataInt
			case 2:
				u.Energy["EN"] = dataInt
			case 3:
				u.Energy["EG"] = dataInt
			case 4:
				u.Energy["EM"] = dataInt
			}
		}
	}

	u.Loaded = true
}

// GetCredits returns current credits.
func (u *User) GetCredits() int {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.Credits
}

// GetPLT returns current PLT.
func (u *User) GetPLT() int {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.PLT
}

// GetBootyKeys returns current booty key count.
func (u *User) GetBootyKeys() int {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.BootyKeys
}

// GetIsLoaded returns whether the user data has been received.
func (u *User) GetIsLoaded() bool {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.Loaded
}

// Snapshot returns a read-only copy of user data.
type UserSnapshot struct {
	Credits    int
	PLT        int
	Experience int
	Honor      int
	Level      int
	BootyKeys  int
	Lasers     map[string]int
	Rockets    map[string]int
}

func (u *User) GetSnapshot() UserSnapshot {
	u.mu.RLock()
	defer u.mu.RUnlock()

	lasers := make(map[string]int)
	for k, v := range u.Lasers {
		lasers[k] = v
	}
	rockets := make(map[string]int)
	for k, v := range u.Rockets {
		rockets[k] = v
	}

	return UserSnapshot{
		Credits:    u.Credits,
		PLT:        u.PLT,
		Experience: u.Experience,
		Honor:      u.Honor,
		Level:      u.Level,
		BootyKeys:  u.BootyKeys,
		Lasers:     lasers,
		Rockets:    rockets,
	}
}

package store

import "time"

type User struct {
	ID              int64
	Nick            string
	Banned          bool
	MaxPrivateRooms int
	NickChangedAt time.Time
	CreatedAt time.Time
}

type Room struct {
	ID            int64
	Name          string
	Private       bool
	OwnerID       int64
	CapBytes      int64
	RetentionDays int
	CreatedAt time.Time

	pinHash string
}

type Message struct {
	ID     int64
	RoomID int64
	UserID int64
	Nick string
	Body string
	At time.Time
}


type DM struct {
	ID       int64
	FromNick string
	ToID int64
	Body string
	At time.Time
}

func fromUnix(n int64) time.Time {
	if n == 0 {
		return time.Time{}
	}
	return time.Unix(n, 0)
}

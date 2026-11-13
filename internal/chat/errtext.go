package chat

import (
	"errors"
	"fmt"
	"math"
	"time"

	"sshchat/internal/store"
)

func (h *Hub) errText(err error) string {
	var locked *store.LockedError
	var cool *store.CooldownError
	switch {
	case errors.As(err, &locked):
		return "Too many wrong PINs. Try again in " + wait(locked.Until.Sub(h.Now())) + "."
	case errors.As(err, &cool):
		return "You can change your nickname again in " + wait(cool.Until.Sub(h.Now())) + "."
	case errors.Is(err, store.ErrBadNick):
		return "Nicknames are 3 to 20 characters: letters, digits, _ and -."
	case errors.Is(err, store.ErrNickTaken):
		return "That nickname is taken."
	case errors.Is(err, store.ErrBadRoomName):
		return "Room names are 2 to 24 characters: letters, digits and -."
	case errors.Is(err, store.ErrRoomTaken):
		return "That room name is taken."
	case errors.Is(err,store.ErrBadPIN):
		return "The PIN must be exactly 6 digits."
	case errors.Is(err, store.ErrWrongPIN):
		return "No such room, or wrong PIN."
	case errors.Is(err, store.ErrRoomLimit):
		return "You have reached your private room limit."
	case errors.Is(err, store.ErrNotFound):
		return "Not found."
	}
	h.Logf("unexpected error: %v", err)
	return "Something went wrong. Try again."
}

func wait(d time.Duration) string {
	switch {
	case d <= 0:
		return "a moment"
	case d < time.Hour:
		return plural(int(math.Ceil(d.Minutes())), "minute")
	case d<48*time.Hour:
		return plural(int(math.Ceil(d.Hours())), "hour")
	default:
		return plural(int(math.Ceil(d.Hours()/24)), "day")
	}
}

func plural(n int, unit string) string{
	if n == 1 { return "1 " + unit }
	return fmt.Sprintf("%d %ss", n, unit)
}

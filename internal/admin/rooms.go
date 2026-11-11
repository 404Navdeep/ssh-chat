### internal/admin/rooms.go
package admin

import (
	"errors"
	"fmt"
	"strings"
	"text/tabwriter"

	"sshchat/internal/store"
)

func roomErr(name string, err error) error {
	if errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("no room named #%s", store.NormalizeRoomName(name))
	}
	return err
}

func setCap(st *store.Store, args []string) (string, error) {
	if err := need(args, 2, "setcap"); err != nil { return "", err }
	mb, err := parseInt(args[1], "the size limit (MB)", 1)
	if err != nil{
		return "", err
	}
	if err := st.SetCap(args[0], int64(mb)<<20); err != nil { return "", roomErr(args[0], err) }

	return fmt.Sprintf("The size limit of #%s is now %d MB.", store.NormalizeRoomName(args[0]), mb), nil
}

func setRetention(st *store.Store, args []string) (string, error) {

	if err := need(args, 2, "setretention"); err != nil {
		return "", err
	}
	days, err := parseInt(args[1], "the retention (days)", 1)
	if err != nil { return "", err }
	if err := st.SetRetention(args[0], days); err != nil {
		return "", roomErr(args[0], err)
	}
	return fmt.Sprintf("#%s now keeps messages for %d days.", store.NormalizeRoomName(args[0]), days), nil
}

func deleteRoom(st *store.Store, args []string) (string, error) {
	if err := need(args, 1, "deleteroom"); err != nil { return "", err }
	if err := st.DeleteRoom(args[0]); err != nil{
		return "", roomErr(args[0], err)
	}
	return fmt.Sprintf("Deleted #%s.", store.NormalizeRoomName(args[0])), nil
}

func listRooms(st *store.Store, _ []string) (string, error) {
	rooms, err := st.ListRooms()
	if err != nil { return "", err }
	var b strings.Builder
	w := tabwriter.NewWriter(&b, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ROOM\tTYPE\tOWNER\tUSED\tLIMIT\tKEEPS")
	for _, r := range rooms {
		kind, limit := "public", "-"
		if r.Private {
			kind = "private"
			limit = fmt.Sprintf("%d MB", r.CapBytes>>20)
		}
		owner := "-"
		if r.OwnerID != 0 {
			if u, err := st.UserByID(r.OwnerID); err==nil{
				owner = u.Nick
			}
		}
		used,err := st.RoomUsage(r.ID)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(w, "#%s\t%s\t%s\t%.2f MB\t%s\t%d days\n",
			r.Name, kind, owner, float64(used)/(1<<20), limit, r.RetentionDays)
	}
	w.Flush()
	return strings.TrimRight(b.String(), "\n"), nil
}

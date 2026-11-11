package admin

import (
	"fmt"
	"errors"
	"strings"
	"text/tabwriter"
	"sshchat/internal/store"
)

func userErr(nick string, err error) error {
	if errors.Is(err, store.ErrNotFound) { return fmt.Errorf("no user named %s", nick) }
	return err
}

func addUser(st *store.Store, args []string) (string, error) {
	if err := need(args, 2, "adduser"); err != nil{
		return "", err
	}
	fp, authorized, err := ParseKey(strings.Join(args[1:], " "))
	if err != nil { return "", err }
	u, err := st.CreateUser(args[0], fp, authorized)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Added user %s with key %s.", u.Nick, fp), nil
}

func addKey(st *store.Store, args []string) (string, error) {
	if err := need(args, 2, "addkey"); err != nil { return "", err }
	fp, authorized, err := ParseKey(strings.Join(args[1:], " "))
	if err != nil {
		return "", err
	}
	if err := st.AddKey(args[0], fp, authorized); err != nil { return "", userErr(args[0], err) }
	return fmt.Sprintf("Added key %s to %s.", fp, args[0]), nil
}
func removeUser(st *store.Store, args []string) (string, error) {
	if err := need(args, 1, "removeuser"); err != nil{
		return "", err
	}
	if err := st.DeleteUser(args[0]); err != nil { return "", userErr(args[0], err) }
	return fmt.Sprintf("Removed user %s. The nickname is free again.", args[0]), nil
}

func ban(st *store.Store, args []string) (string, error) {

	if err := need(args, 1, "ban"); err != nil {
		return "", err
	}
	if err := st.SetBanned(args[0], true); err != nil { return "", userErr(args[0], err) }
	return fmt.Sprintf("Banned %s. Running sessions are closed within 10 seconds.", args[0]), nil
}

func unban(st *store.Store, args []string) (string, error) {
	if err := need(args, 1, "unban"); err != nil {
		return "", err
	}
	if err := st.SetBanned(args[0], false); err != nil { return "", userErr(args[0], err) }

	return fmt.Sprintf("Unbanned %s.", args[0]), nil
}


func setRoomLimit(st *store.Store, args []string) (string, error){
	if err := need(args, 2, "setroomlimit"); err != nil {
		return "", err
	}
	n, err := parseInt(args[1], "the room limit", 0)
	if err != nil { return "", err }
	if err := st.SetMaxPrivateRooms(args[0], n); err != nil {
		return "", userErr(args[0], err)
	}
	return fmt.Sprintf("%s may now own %d private rooms.", args[0], n), nil
}

func listUsers(st *store.Store, _ []string) (string, error) {

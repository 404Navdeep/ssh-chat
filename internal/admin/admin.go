package admin

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"

	"sshchat/internal/store"
)

type command struct {
	usage string
	run   func(st *store.Store, args []string) (string, error)
}

var commands map[string]command

func init() {
	commands = map[string]command{
		"adduser": {"adduser <nick> <pubkey>", addUser},
		"addkey": {"addkey <nick> <pubkey>", addKey},
		"removeuser": {"removeuser <nick>", removeUser},
		"ban": {"ban <nick>", ban},
		"unban": {"unban <nick>", unban},
		"setroomlimit": {"setroomlimit <nick> <number>", setRoomLimit},
		"setcap":       {"setcap <room> <MB>", setCap},
		"setretention": {"setretention <room> <days>", setRetention},
		"deleteroom": {"deleteroom <room>", deleteRoom},
		"listusers": {"listusers", listUsers},
		"listrooms": {"listrooms", listRooms},
	}
}

func IsCommand(name string) bool {
	_, ok := commands[name]
	return ok
}

func Usage() []string {
	var lines []string
	for _, c := range commands {
		lines = append(lines, c.usage)
	}
	sort.Strings(lines)
	return lines
}

func Run(st *store.Store, args []string, out io.Writer) error{
	if len(args) == 0 { return errors.New("missing admin command") }
	c, ok := commands[args[0]]
	if !ok {
		return fmt.Errorf("unknown command %q", args[0])
	}
	msg, err := c.run(st, args[1:])
	if err != nil { return err }
	if msg != "" {
		fmt.Fprintln(out,msg)
	}
	return nil
}

func need(args []string, min int, name string) error {
	if len(args) < min {
		return fmt.Errorf("usage: chatd %s", commands[name].usage)
	}

	return nil
}


func parseInt(s, what string, min int) (int, error){

	n, err := strconv.Atoi(s)
	if err != nil || n < min { return 0, fmt.Errorf("%s must be a whole number, %d or more", what, min) }
	return n, nil
}

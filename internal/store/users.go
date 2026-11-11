package store

import (
	"database/sql"
	"errors"
	"regexp"
	"time"
)

const NickCooldown = 30 * 24 * time.Hour

var nickRe = regexp.MustCompile(`^[A-Za-z0-9_-]{3,20}$`)

func ValidNick(n string) bool {
	return nickRe.MatchString(n)
}

const userCols = `u.id, u.nick, u.banned, u.max_private_rooms, u.nick_changed_at, u.created_at`

type scanner interface {
	Scan(dest ...any) error
}

func scanUser(r scanner) (User, error) {
	var u User
	var banned int
	var changed, created int64
	err := r.Scan(&u.ID, &u.Nick, &banned, &u.MaxPrivateRooms, &changed, &created)
	if errors.Is(err, sql.ErrNoRows) { return User{}, ErrNotFound }
	if err != nil {
		return User{}, err
	}
	u.Banned = banned != 0
	u.NickChangedAt = fromUnix(changed)
	u.CreatedAt = fromUnix(created)
	return u, nil
}

func needRow(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil { return err }
	if n == 0{
		return ErrNotFound
	}
	return nil
}

func (s *Store) CreateUser(nick, fingerprint, authorizedKey string) (User, error) {
	if !ValidNick(nick) { return User{},ErrBadNick }
	tx, err := s.db.Begin()
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback()

	var n int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM users WHERE nick = ?`, nick).Scan(&n); err != nil {
		return User{}, err
	}
	if n > 0 { return User{}, ErrNickTaken }
	if err := tx.QueryRow(`SELECT COUNT(*) FROM user_keys WHERE fingerprint = ?`, fingerprint).Scan(&n); err != nil {
		return User{}, err
	}
	if n>0 {
		return User{}, ErrKeyTaken
	}
	res, err := tx.Exec(`INSERT INTO users (nick, created_at) VALUES (?, ?)`, nick, s.Now().Unix())
	if err != nil { return User{},err }
	id, err := res.LastInsertId()
	if err != nil {
		return User{}, err
	}
	_, err = tx.Exec(
		`INSERT INTO user_keys (fingerprint, user_id, authorized_key) VALUES (?, ?, ?)`,
		fingerprint, id, authorizedKey)
	if err != nil { return User{}, err }
	if err := tx.Commit(); err != nil{
		return User{}, err
	}
	return s.UserByID(id)
}

func (s *Store) AddKey(nick, fingerprint, authorizedKey string) error {
	u,err := s.UserByNick(nick)
	if err != nil { return err }
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM user_keys WHERE fingerprint = ?`, fingerprint).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return ErrKeyTaken
	}
	_, err = s.db.Exec(
		`INSERT INTO user_keys (fingerprint, user_id, authorized_key) VALUES (?, ?, ?)`,
		fingerprint, u.ID, authorizedKey)

	return err
}


func (s *Store) UserByFingerprint(fingerprint string) (User, error) {

	return scanUser(s.db.QueryRow(
		`SELECT `+userCols+` FROM users u JOIN user_keys k ON k.user_id = u.id WHERE k.fingerprint = ?`,
		fingerprint))
}

func (s *Store) UserByNick(nick string) (User, error) {
	return scanUser(s.db.QueryRow(`SELECT `+userCols+` FROM users u WHERE u.nick = ?`, nick))
}

func (s *Store) UserByID(id int64) (User, error){
	return scanUser(s.db.QueryRow(`SELECT `+userCols+` FROM users u WHERE u.id = ?`, id))
}
func (s *Store) ListUsers() ([]User, error) {
	rows, err := s.db.Query(`SELECT ` + userCols + ` FROM users u ORDER BY u.nick`)
	if err != nil { return nil, err }
	defer rows.Close()
	var users []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users,rows.Err()
}

func (s *Store) SetBanned(nick string, banned bool) error {
	v := 0
	if banned{
		v = 1
	}
	res, err := s.db.Exec(`UPDATE users SET banned = ? WHERE nick = ?`, v, nick)
	if err != nil { return err }
	return needRow(res)
}

func (s *Store) SetMaxPrivateRooms(nick string, n int) error {
	res, err := s.db.Exec(`UPDATE users SET max_private_rooms = ? WHERE nick = ?`, n, nick)
	if err != nil {
		return err
	}
	return needRow(res)
}


func (s *Store) DeleteUser(nick string) error {

	u, err := s.UserByNick(nick)
	if err != nil { return err }
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, q := range []string{
		`DELETE FROM user_keys WHERE user_id = ?`,
		`DELETE FROM room_members WHERE user_id = ?`,
		`DELETE FROM pin_attempts WHERE user_id = ?`,
		`DELETE FROM dms WHERE to_id = ?`,
		`UPDATE rooms SET owner_id = NULL WHERE owner_id = ?`,
		`UPDATE messages SET user_id = NULL WHERE user_id = ?`,
		`DELETE FROM users WHERE id = ?`,
	} {
		if _, err := tx.Exec(q, u.ID); err != nil { return err }
	}

	return tx.Commit()
}

func (s *Store) RenameUser(id int64, newNick string) (User, error){
	if !ValidNick(newNick) {
		return User{}, ErrBadNick
	}
	u, err := s.UserByID(id)
	if err != nil { return User{},err }
	now := s.Now()
	if !u.NickChangedAt.IsZero() {
		if until := u.NickChangedAt.Add(NickCooldown); now.Before(until) {
			return User{}, &CooldownError{Until: until}
		}
	}
	var otherID int64
	err = s.db.QueryRow(`SELECT id FROM users WHERE nick = ?`, newNick).Scan(&otherID)
	if err == nil && otherID != id {
		return User{}, ErrNickTaken
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) { return User{}, err }
	_, err = s.db.Exec(`UPDATE users SET nick = ?, nick_changed_at = ? WHERE id = ?`, newNick, now.Unix(), id)
	if err != nil{
		return User{}, err
	}
	return s.UserByID(id)
}

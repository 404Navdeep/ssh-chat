### internal/store/messages.go
package store

import "time"

func (s *Store) AddMessage(roomID int64, u User, body string) (Message, error) {

	room, err := s.RoomByID(roomID)
	if err != nil { return Message{}, err }
	size := int64(len(body))
	if room.Private && room.CapBytes > 0{
		used, err := s.RoomUsage(roomID)
		if err != nil {
			return Message{}, err
		}
		if used+size>room.CapBytes { return Message{},ErrRoomFull }
	}
	now := s.Now().Unix()
	res, err := s.db.Exec(
		`INSERT INTO messages (room_id, user_id, nick, body, size, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		roomID, u.ID, u.Nick, body, size, now)
	if err != nil {
		return Message{}, err
	}
	id, err := res.LastInsertId()
	if err != nil { return Message{}, err }
	return Message{
		ID: id, RoomID: roomID, UserID: u.ID, Nick: u.Nick, Body: body, At: time.Unix(now, 0),
	},nil
}

func (s *Store) RecentMessages(roomID int64, limit int) ([]Message, error) {
	rows, err := s.db.Query(
		`SELECT id, room_id, COALESCE(user_id, 0), nick, body, created_at FROM (
			SELECT * FROM messages WHERE room_id = ? ORDER BY id DESC LIMIT ?
		 ) ORDER BY id`, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next(){
		var m Message
		var created int64
		if err := rows.Scan(&m.ID, &m.RoomID, &m.UserID, &m.Nick, &m.Body, &created); err != nil {
			return nil, err
		}
		m.At = time.Unix(created, 0)
		out = append(out, m)
	}

	return out,rows.Err()
}
func (s *Store) ClearRoom(roomID int64) error {
	_, err := s.db.Exec(`DELETE FROM messages WHERE room_id = ?`, roomID)
	return err
}

func (s *Store) PurgeMessages() (int64, error) {
	res, err := s.db.Exec(
		`DELETE FROM messages
		 WHERE created_at < ? - (SELECT retention_days FROM rooms WHERE rooms.id = messages.room_id) * 86400`,
		s.Now().Unix())
	if err != nil { return 0, err }
	return res.RowsAffected()
}

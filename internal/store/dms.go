package store

import "time"

func (s *Store) AddDM(fromNick string, toID int64, body string) error {
	_, err := s.db.Exec(
		`INSERT INTO dms (from_nick, to_id, body, created_at) VALUES (?, ?, ?, ?)`,
		fromNick,toID,body,s.Now().Unix())
	return err
}

func (s *Store) TakePendingDMs(toID int64, limit int) ([]DM, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.Query(
		`SELECT id, from_nick, to_id, body, created_at FROM dms WHERE to_id = ? ORDER BY id LIMIT ?`,
		toID, limit)
	if err != nil { return nil, err }
	var out []DM
	for rows.Next() {
		var d DM
		var created int64
		if err := rows.Scan(&d.ID, &d.FromNick, &d.ToID, &d.Body, &created); err != nil{
			rows.Close()
			return nil,err
		}
		d.At = time.Unix(created, 0)
		out = append(out, d)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for _, d := range out {
		if _, err := tx.Exec(`DELETE FROM dms WHERE id = ?`, d.ID); err != nil { return nil, err }
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *Store) PurgeDMs() (int64, error) {
	cutoff := s.Now().Add(-DMRetentionDays * 24 * time.Hour).Unix()
	res, err := s.db.Exec(`DELETE FROM dms WHERE created_at < ?`, cutoff)
	if err != nil { return 0,err }
	return res.RowsAffected()
}

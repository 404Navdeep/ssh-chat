### internal/chat/limits.go
package chat

import "time"

const (
	MaxMessageRunes = 500
	RateMessages    = 5
	RateWindow      = 10 * time.Second
	HistoryLimit    = 50
	PendingDMLimit  = 200
)

type limiter struct {
	hits []time.Time
}

func (l *limiter) allow(now time.Time) bool {
	cutoff := now.Add(-RateWindow)
	kept := l.hits[:0]
	for _, t := range l.hits {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	l.hits = kept
	if len(l.hits) >= RateMessages { return false }
	l.hits = append(l.hits, now)
	return true
}

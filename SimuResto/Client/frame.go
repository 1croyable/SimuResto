package client

import "time"

type FrameLimiter struct {
	next time.Time
	fps  int
}

// Calculate how long to wait before the next frame.
func (l *FrameLimiter) delay(now time.Time) time.Duration {
	if l.next.IsZero() {
		return 0
	}
	return max(0, l.next.Sub(now))
}

func (l *FrameLimiter) step(now time.Time, interval time.Duration) {
	if l.next.IsZero() || now.Sub(l.next) >= interval {
		l.next = now.Add(interval)
		return
	}
	l.next = l.next.Add(interval)
}

// Wait until the next scheduled frame.
func (l *FrameLimiter) Wait(fps int) {
	if fps <= 0 {
		return
	}

	// Reset the schedule when FPS changes.
	if l.fps != fps {
		l.fps = fps
		l.next = time.Time{}
	}

	interval := (time.Second + time.Duration(fps) - 1) / time.Duration(fps)

	if delay := l.delay(time.Now()); delay > 0 {
		time.Sleep(delay)
	}

	l.step(time.Now(), interval)
}

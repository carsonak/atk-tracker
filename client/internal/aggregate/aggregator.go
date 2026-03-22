package aggregate

import (
	"time"
)

type TickSummary struct {
	EndAt         time.Time
	ActiveSeconds int
}

type Aggregator struct {
	windowSeconds int
}

func New(window time.Duration) *Aggregator {
	seconds := int(window.Seconds())

	if seconds <= 0 {
		seconds = 300
	}

	return &Aggregator{windowSeconds: seconds}
}

func (a *Aggregator) Run(events <-chan struct{}, stop <-chan struct{}) <-chan TickSummary {
	out := make(chan TickSummary, 4)

	go func() {
		defer close(out)
		secTicker := time.NewTicker(1 * time.Second)

		defer secTicker.Stop()

		activeThisSecond := false
		activeSeconds := 0
		ticks := 0

		for {
			select {
			case <-stop:
				return
			case <-events:
				activeThisSecond = true
			case t := <-secTicker.C:
				if activeThisSecond {
					activeSeconds++
				}

				ticks++
				activeThisSecond = false
				if ticks >= a.windowSeconds {
					select {
					case out <- TickSummary{EndAt: t.UTC(), ActiveSeconds: activeSeconds}:
					default:
					}

					activeSeconds = 0
					ticks = 0
				}
			}
		}
	}()

	return out
}

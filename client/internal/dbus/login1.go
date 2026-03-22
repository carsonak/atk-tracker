package dbus

import (
	"context"
	"fmt"

	"github.com/godbus/dbus/v5"
)

type SessionEventType string

const (
	SessionNew      SessionEventType = "SessionNew"
	SessionRemoved  SessionEventType = "SessionRemoved"
	PrepareForSleep SessionEventType = "PrepareForSleep"
)

type SessionEvent struct {
	Type      SessionEventType
	SessionID string
	UID       uint32
	Sleeping  bool
}

type Watcher struct {
	conn *dbus.Conn
}

func NewWatcher() (*Watcher, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, fmt.Errorf("connect system bus: %w", err)
	}

	return &Watcher{conn: conn}, nil
}

func (w *Watcher) ListSessions(ctx context.Context) ([]SessionEvent, error) {
	obj := w.conn.Object("org.freedesktop.login1", "/org/freedesktop/login1")
	call := obj.CallWithContext(ctx, "org.freedesktop.login1.Manager.ListSessions", 0)

	if call.Err != nil {
		return nil, call.Err
	}

	var rows [][]interface{}

	if err := call.Store(&rows); err != nil {
		return nil, err
	}

	result := make([]SessionEvent, 0, len(rows))

	for _, row := range rows {
		if len(row) < 2 {
			continue
		}

		sid, _ := row[0].(string)
		uid, _ := row[1].(uint32)

		result = append(result, SessionEvent{Type: SessionNew, SessionID: sid, UID: uid})
	}

	return result, nil
}

func (w *Watcher) Subscribe(ctx context.Context) (<-chan SessionEvent, error) {
	if err := w.conn.AddMatchSignal(dbus.WithMatchInterface("org.freedesktop.login1.Manager")); err != nil {
		return nil, err
	}

	sigChan := make(chan *dbus.Signal, 32)

	w.conn.Signal(sigChan)
	out := make(chan SessionEvent, 32)

	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case sig := <-sigChan:
				if sig == nil {
					continue
				}

				switch sig.Name {
				case "org.freedesktop.login1.Manager.SessionNew":
					if len(sig.Body) >= 2 {
						sid, _ := sig.Body[0].(string)
						uid, _ := sig.Body[1].(uint32)

						out <- SessionEvent{Type: SessionNew, SessionID: sid, UID: uid}
					}
				case "org.freedesktop.login1.Manager.SessionRemoved":
					if len(sig.Body) >= 1 {
						sid, _ := sig.Body[0].(string)

						out <- SessionEvent{Type: SessionRemoved, SessionID: sid}
					}
				case "org.freedesktop.login1.Manager.PrepareForSleep":
					if len(sig.Body) >= 1 {
						sleep, _ := sig.Body[0].(bool)

						out <- SessionEvent{Type: PrepareForSleep, Sleeping: sleep}
					}
				}
			}
		}
	}()

	return out, nil
}

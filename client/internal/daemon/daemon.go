package daemon

import (
	"context"
	"errors"
	"log"
	"os/user"
	"strconv"
	"time"

	"atk-tracker/client/internal/aggregate"
	"atk-tracker/client/internal/buffer"
	dbuswatch "atk-tracker/client/internal/dbus"
	"atk-tracker/client/internal/input"
	"atk-tracker/client/internal/session"
	"atk-tracker/client/internal/syncer"
	"atk-tracker/shared/go/atkshared"
)

type Daemon struct {
	cfg        Config
	reader     *input.Reader
	agg        *aggregate.Aggregator
	buffer     *buffer.Store
	watcher    *dbuswatch.Watcher
	http       *syncer.HTTPClient
	session    *session.State
	stop       chan struct{}
	eventPulse chan struct{}
}

func New(cfg Config) (*Daemon, error) {
	reader, err := input.NewReader()
	if err != nil {
		return nil, err
	}

	store, err := buffer.New(cfg.BufferPath)
	if err != nil {
		return nil, err
	}

	watcher, err := dbuswatch.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Daemon{
		cfg:        cfg,
		reader:     reader,
		agg:        aggregate.New(cfg.HeartbeatWindow),
		buffer:     store,
		watcher:    watcher,
		http:       syncer.New(cfg.ServerURL, cfg.RequestTimeout),
		session:    &session.State{},
		stop:       make(chan struct{}),
		eventPulse: make(chan struct{}, 1024),
	}, nil
}

func (d *Daemon) Run(ctx context.Context) error {
	defer close(d.stop)
	defer d.buffer.Close()

	if err := d.bootstrapSession(ctx); err != nil {
		log.Printf("bootstrap warning: %v", err)
	}

	dbusEvents, err := d.watcher.Subscribe(ctx)
	if err != nil {
		return err
	}

	activity := d.reader.Start(d.stop)
	summaries := d.agg.Run(d.eventPulse, d.stop)
	flushTicker := time.NewTicker(d.cfg.OfflineFlushEvery)

	defer flushTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			d.flushAndEndSession(context.Background())
			return nil
		case ev, ok := <-activity:
			if !ok {
				return nil
			}

			_ = ev
			select {
			case d.eventPulse <- struct{}{}:
			default:
			}
		case summary, ok := <-summaries:
			if !ok {
				return nil
			}

			d.handleSummary(ctx, summary)
		case ev := <-dbusEvents:
			d.handleDBusEvent(ctx, ev)
		case <-flushTicker.C:
			d.flushBuffer(ctx)
		}
	}
}

func (d *Daemon) bootstrapSession(ctx context.Context) error {
	sessions, err := d.watcher.ListSessions(ctx)
	if err != nil {
		return err
	}

	for _, s := range sessions {
		if s.UID == 0 {
			continue
		}

		apprenticeID := uidToApprentice(s.UID)
		sessionID, err := d.http.CreateSession(ctx, apprenticeID, d.cfg.MachineID)
		if err != nil {
			return err
		}

		d.session.Set(sessionID, apprenticeID)
		return nil
	}

	return nil
}

func (d *Daemon) handleDBusEvent(ctx context.Context, ev dbuswatch.SessionEvent) {
	switch ev.Type {
	case dbuswatch.SessionNew:
		if ev.UID == 0 {
			return
		}

		apprenticeID := uidToApprentice(ev.UID)
		sessionID, err := d.http.CreateSession(ctx, apprenticeID, d.cfg.MachineID)
		if err != nil {
			log.Printf("session create failed: %v", err)
			return
		}

		d.flushAndEndSession(ctx)
		d.session.Set(sessionID, apprenticeID)
	case dbuswatch.SessionRemoved:
		d.flushAndEndSession(ctx)
		d.session.Clear()
	case dbuswatch.PrepareForSleep:
		if ev.Sleeping {
			d.flushAndEndSession(ctx)
		}
	}
}

func (d *Daemon) handleSummary(ctx context.Context, s aggregate.TickSummary) {
	sessionID, apprenticeID := d.session.Get()

	if sessionID == "" {
		return
	}

	payload := atkshared.HeartbeatPayload{
		SessionID:    sessionID,
		Timestamp:    s.EndAt.UTC(),
		Duration:     s.ActiveSeconds,
		MachineID:    d.cfg.MachineID,
		ApprenticeID: apprenticeID,
	}

	if err := atkshared.ValidateHeartbeatDuration(payload.Duration); err != nil {
		log.Printf("invalid heartbeat duration: %v", err)
		return
	}

	if err := d.http.SendHeartbeat(ctx, payload); err != nil {
		if errors.Is(err, syncer.ErrSessionInvalid) {
			if newID, recreateErr := d.http.CreateSession(ctx, apprenticeID, d.cfg.MachineID); recreateErr == nil {
				d.session.Set(newID, apprenticeID)
				payload.SessionID = newID
				if retryErr := d.http.SendHeartbeat(ctx, payload); retryErr == nil {
					return
				}
			}
		}

		if queueErr := d.buffer.Enqueue(ctx, payload); queueErr != nil {
			log.Printf("failed to queue heartbeat: %v", queueErr)
		}
	}
}

func (d *Daemon) flushBuffer(ctx context.Context) {
	queued, err := d.buffer.DequeueBatch(ctx, 100)
	if err != nil {
		log.Printf("buffer dequeue failed: %v", err)
		return
	}

	for _, item := range queued {
		if err := d.http.SendHeartbeat(ctx, item.Payload); err != nil {
			return
		}

		if err := d.buffer.DeleteByID(ctx, item.ID); err != nil {
			log.Printf("buffer cleanup failed id=%d err=%v", item.ID, err)
		}
	}
}

func (d *Daemon) flushAndEndSession(ctx context.Context) {
	d.flushBuffer(ctx)
	sid, _ := d.session.Get()

	if sid == "" {
		return
	}

	if err := d.http.EndSession(ctx, sid, time.Now().UTC()); err != nil {
		log.Printf("end session failed: %v", err)
	}
}

func uidToApprentice(uid uint32) string {
	uidStr := strconv.FormatUint(uint64(uid), 10)
	usr, err := user.LookupId(uidStr)
	if err != nil || usr == nil || usr.Username == "" {
		return "uid-" + uidStr
	}
	return usr.Username
}

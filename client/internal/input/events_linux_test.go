package input

import (
	"bytes"
	"encoding/binary"
	"testing"

	"golang.org/x/sys/unix"
)

func TestParseInputEvent_MalformedPacket(t *testing.T) {
	_, err := parseInputEvent([]byte{1, 2, 3})
	if err == nil {
		t.Fatal("expected error for packet shorter than input_event size")
	}
}

func TestParseInputEvent_EmptyPacket(t *testing.T) {
	_, err := parseInputEvent([]byte{})
	if err == nil {
		t.Fatal("expected error for empty packet")
	}
}

func TestParseInputEvent_KeyEventIsActivity(t *testing.T) {
	ev := inputEvent{Type: evKey}
	packet := encodeEvent(t, ev)
	ok, err := parseInputEvent(packet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("key event should be reported as activity")
	}
}

func TestParseInputEvent_RelEventIsActivity(t *testing.T) {
	ev := inputEvent{Type: evRel}
	packet := encodeEvent(t, ev)
	ok, err := parseInputEvent(packet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("relative mouse event should be reported as activity")
	}
}

func TestParseInputEvent_SyncEventIgnored(t *testing.T) {
	// Type 0x00 = EV_SYN — should not be counted as activity
	ev := inputEvent{Type: 0x00}
	packet := encodeEvent(t, ev)
	ok, err := parseInputEvent(packet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("sync event should not be reported as activity")
	}
}

func TestParseInputEvent_AbsEventIgnored(t *testing.T) {
	// Type 0x03 = EV_ABS — touchpad absolute events, not activity for us
	ev := inputEvent{Type: 0x03}
	packet := encodeEvent(t, ev)
	ok, err := parseInputEvent(packet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("absolute event should not be reported as activity")
	}
}

func TestParseInputEvent_WithTimestamp(t *testing.T) {
	ev := inputEvent{
		Time:  unix.Timeval{Sec: 1234567890, Usec: 0},
		Type:  evKey,
		Code:  30, // KEY_A
		Value: 1,  // key down
	}
	packet := encodeEvent(t, ev)
	ok, err := parseInputEvent(packet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("key event with timestamp should be activity")
	}
}

func encodeEvent(t *testing.T, ev inputEvent) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, ev); err != nil {
		t.Fatalf("encode input event: %v", err)
	}
	return buf.Bytes()
}

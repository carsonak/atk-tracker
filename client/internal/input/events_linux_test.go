package input

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestParseInputEventMalformedPacket(t *testing.T) {
	_, err := parseInputEvent([]byte{1, 2, 3})

	if err == nil {
		t.Fatal("expected parse error for malformed packet")
	}
}

func TestParseInputEventActivity(t *testing.T) {
	ev := inputEvent{Type: evKey}
	packet := make([]byte, inputEventSize)
	buf := bytes.NewBuffer(packet[:0])
	if err := binary.Write(buf, binary.LittleEndian, ev); err != nil {
		t.Fatalf("unexpected encode error: %v", err)
	}
	packet = buf.Bytes()
	ok, err := parseInputEvent(packet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ok {
		t.Fatal("expected activity event")
	}
}

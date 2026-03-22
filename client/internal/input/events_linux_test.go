package input

import "testing"

func TestParseInputEventMalformedPacket(t *testing.T) {
	_, err := parseInputEvent([]byte{1, 2, 3})
	if err == nil {
		t.Fatal("expected parse error for malformed packet")
	}
}

func TestParseInputEventActivity(t *testing.T) {
	packet := make([]byte, 24)
	packet[16] = 0x01
	ok, err := parseInputEvent(packet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected activity event")
	}
}

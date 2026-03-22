package input

import (
	"bytes"
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	evKey = 0x01
	evRel = 0x02
)

type ActivityEvent struct {
	Timestamp time.Time
}

type inputEvent struct {
	Time  unix.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

var inputEventSize = int(unsafe.Sizeof(inputEvent{}))

type Reader struct {
	paths []string
}

func NewReader() (*Reader, error) {
	paths, err := filterInputDevices()
	if err != nil {
		return nil, err
	}

	return &Reader{paths: paths}, nil
}

func (r *Reader) Start(stop <-chan struct{}) <-chan ActivityEvent {
	out := make(chan ActivityEvent, 1024)
	var wg sync.WaitGroup

	for _, p := range r.paths {
		p := p

		wg.Add(1)
		go func() {
			defer wg.Done()
			streamDevice(p, stop, out)
		}()
	}

	go func() {
		<-stop
		wg.Wait()
		close(out)
	}()

	return out
}

func filterInputDevices() ([]string, error) {
	devices, err := filepath.Glob("/dev/input/event*")
	if err != nil {
		return nil, fmt.Errorf("list event devices: %w", err)
	}

	filtered := make([]string, 0, len(devices))

	for _, dev := range devices {
		ok, err := supportsKeyOrRel(dev)
		if err != nil {
			continue
		}

		if ok {
			filtered = append(filtered, dev)
		}
	}

	return filtered, nil
}

func supportsKeyOrRel(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	fd := int(f.Fd())
	buf := make([]byte, 64)
	req := eviocgbitRequest(0, len(buf))

	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(req), uintptr(unsafe.Pointer(&buf[0]))); errno != 0 {
		return false, errno
	}

	return bitIsSet(buf, evKey) || bitIsSet(buf, evRel), nil
}

func bitIsSet(bitset []byte, bit int) bool {
	byteIndex := bit / 8
	bitOffset := bit % 8

	if byteIndex >= len(bitset) {
		return false
	}

	return (bitset[byteIndex] & (1 << bitOffset)) != 0
}

func eviocgbitRequest(evType, length int) uint {
	const (
		iocRead      = 2
		iocNRBits    = 8
		iocTypeBits  = 8
		iocSizeBits  = 14
		iocNRShift   = 0
		iocTypeShift = iocNRShift + iocNRBits
		iocSizeShift = iocTypeShift + iocTypeBits
		iocDirShift  = iocSizeShift + iocSizeBits
	)

	return uint((iocRead << iocDirShift) | (int('E') << iocTypeShift) | ((0x20 + evType) << iocNRShift) | (length << iocSizeShift))
}

func streamDevice(path string, stop <-chan struct{}, out chan<- ActivityEvent) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	stopRead := make(chan struct{})
	go func() {
		select {
		case <-stop:
			_ = f.Close()
		case <-stopRead:
		}
	}()
	defer close(stopRead)

	reader := bufio.NewReader(f)
	packet := make([]byte, inputEventSize)

	for {
		select {
		case <-stop:
			return
		default:
		}

		if _, err := io.ReadFull(reader, packet); err != nil {
			return
		}

		isActivity, err := parseInputEvent(packet)
		if err != nil {
			continue
		}

		if isActivity {
			select {
			case out <- ActivityEvent{Timestamp: time.Now().UTC()}:
			default:
			}
		}
	}
}

func parseInputEvent(packet []byte) (bool, error) {
	if len(packet) != inputEventSize {
		return false, fmt.Errorf("invalid input_event packet length=%d", len(packet))
	}
	var ev inputEvent
	if err := binary.Read(bytes.NewReader(packet), binary.LittleEndian, &ev); err != nil {
		return false, fmt.Errorf("parse input_event: %w", err)
	}

 	evType := ev.Type

	return evType == evKey || evType == evRel, nil
}

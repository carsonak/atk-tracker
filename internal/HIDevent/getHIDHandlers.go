package HIDevent

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	// The following constants indicate the starting positions of various bits
	// for an `ioctl()` syscall request parameter.

	ioControlNumberShift    = 0  // command identifier.
	ioControlTypeShift      = 8  // subsystem/event identifier.
	ioControlSizeShift      = 16 // size of buffer.
	ioControlDirectionShift = 30 // reading direction of the buffer.

	ioControlReading = 2            // ioctl reading direction.
	ioControlEvent   = uintptr('E') // ioctl event magic number.

	// The following constants indicate the various capabilities of
	// different I/O devices.

	evKey = 0x01 // Keys and Buttons (Keyboard, Mouse clicks)
	evRel = 0x02 // Relative axes (Mouse movement)
	evMax = 0x1F // The highest event type value in Linux.

)

const wordSize = unsafe.Sizeof(uint(0))

// ioControlEventRequestBits generates the correct `ioctl()` request code.
func ioControlEventRequestBits(eventNumber uintptr, bitsInBuffer uintptr) uintptr {
	return (ioControlReading << ioControlDirectionShift) |
		(bitsInBuffer << ioControlSizeShift) |
		(ioControlEvent << ioControlTypeShift) |
		((0x20 + eventNumber) << ioControlNumberShift)
}

func checkBit(bitmask []uint, bit uintptr) bool {
	wordIndex := bit / wordSize
	bitIndex := bit % wordSize

	if wordIndex >= uintptr(len(bitmask)) {
		return false
	}

	return (bitmask[wordIndex] & (1 << bitIndex)) != 0
}

func isActivityEvent(file *os.File) (bool, error) {
	fd := file.Fd()
	const arraySize = (evMax + wordSize - 1) / wordSize
	bitmask := [arraySize]uint{}
	request := ioControlEventRequestBits(0, uintptr(len(bitmask))*(wordSize/8))
	_, _, sysErr := unix.Syscall(unix.SYS_IOCTL, fd, request, uintptr(unsafe.Pointer(&bitmask[0])))

	if sysErr != 0 {
		return false, sysErr
	}

	supportsKeys := checkBit(bitmask[:], evKey)
	supportsMouse := checkBit(bitmask[:], evRel)

	return supportsKeys || supportsMouse, nil
}

// GetHIDHandlers returns a slice of all event handlers for input devices like
// mice and keyboard.
// It also returns a slice of errors encountered while trying to identify the
// event handlers.
func GetHIDHandlers() (validDevices []string, errs []error) {
	files, err := filepath.Glob("/dev/input/event*")
	if err != nil {
		return nil, []error{err}
	}

	checkEventType := func(filename string) (bool, error) {
		file, err := os.Open(filename)
		if err != nil {
			return false, fmt.Errorf("error opening %q: %w", filename, err)
		}
		defer file.Close()

		res, err := isActivityEvent(file)
		if err != nil {
			return false, fmt.Errorf("error reading %q: %w", filename, err)
		}

		return res, nil
	}

	for _, filename := range files {
		res, err := checkEventType(filename)
		if err != nil {
			errs = append(errs, err)
		}

		if res {
			validDevices = append(validDevices, filename)
		}
	}

	return validDevices, errs
}

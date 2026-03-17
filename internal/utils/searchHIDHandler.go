package utils

import (
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Standard Linux input event types

const (
	EV_KEY = 0x01 // Keys and Buttons (Keyboard, Mouse clicks)
	EV_REL = 0x02 // Relative axes (Mouse movement)

	EV_MAX = 0x1F // EV_MAX is the highest event type value in Linux
)

// We calculate the number of native words needed.
// EV_MAX is 31.
// On 64-bit: (31 + 63) / 64 = 1 word
// On 32-bit: (31 + 31) / 32 = 1 word
const wordSize = 32 << (^uint(0) >> 63) // Magic trick to dynamically get 32 or 64

// eventIOControlGetBit recreates the Linux C macro (EVIOCGBIT) to generate
// the correct ioctl request code.
// It uses standard Linux ioctl bit shifts:
// DIR (2 bits) | SIZE (14 bits) | TYPE (8 bits) | NR (8 bits)
func eventIOControlGetBit(ev int, size int) int {
	const (
		IOC_NRSHIFT   = 0
		IOC_TYPESHIFT = 8
		IOC_SIZESHIFT = 16
		IOC_DIRSHIFT  = 30
		IOC_READ      = 2
	)

	return (IOC_READ << IOC_DIRSHIFT) |
		(size << IOC_SIZESHIFT) |
		(int('E') << IOC_TYPESHIFT) |
		((0x20 + ev) << IOC_NRSHIFT)
}

func checkBit(bitmask []uint, bit int) bool {
	wordIndex := bit / wordSize
	bitIndex := bit % wordSize

	if wordIndex >= len(bitmask) {
		return false
	}

	// The Go compiler handles the endianness automatically here
	return (bitmask[wordIndex] & (1 << bitIndex)) != 0
}

func IsActivityDevice(file *os.File) (bool, error) {
	fd := file.Fd()
	const arraySize = (EV_MAX + wordSize - 1) / wordSize
	// Allocate a slice of native integers instead of bytes
	bitmask := [arraySize]uint{}
	// We calculate the byte length of the slice to tell the kernel
	// len(bitmask) * (wordSize / 8) gives us the total bytes
	request := eventIOControlGetBit(0, len(bitmask)*(wordSize/8))
	_, _, sysErr := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(request), uintptr(unsafe.Pointer(&bitmask[0])))

	if sysErr != 0 {
		return false, sysErr
	}

	supportsKeys := checkBit(bitmask[:], EV_KEY)
	supportsMouse := checkBit(bitmask[:], EV_REL)

	return supportsKeys || supportsMouse, nil
}

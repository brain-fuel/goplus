//go:build windows

package memory

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

func platformAllocate(size int) ([]byte, error) {
	address, err := windows.VirtualAlloc(0, uintptr(size), windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
	if err != nil {
		return nil, err
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(address)), size), nil
}

func platformRelease(storage []byte) error {
	if len(storage) == 0 {
		return nil
	}
	return windows.VirtualFree(uintptr(unsafe.Pointer(&storage[0])), 0, windows.MEM_RELEASE)
}

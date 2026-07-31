//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package memory

import "golang.org/x/sys/unix"

func platformAllocate(size int) ([]byte, error) {
	return unix.Mmap(-1, 0, size, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_PRIVATE|unix.MAP_ANON)
}

func platformRelease(storage []byte) error {
	return unix.Munmap(storage)
}

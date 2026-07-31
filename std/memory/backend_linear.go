//go:build js || wasip1 || plan9

package memory

func platformAllocate(size int) ([]byte, error) {
	return make([]byte, size), nil
}

func platformRelease([]byte) error {
	return nil
}

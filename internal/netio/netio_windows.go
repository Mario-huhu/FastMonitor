// +build windows

package netio

import (
	"fastmonitor/pkg/model"
)

// EnhancedDarwinList is a stub for Windows (not used)
// This function exists only to satisfy the compiler on Windows
// The actual implementation is in netio_darwin.go for macOS
func EnhancedDarwinList() ([]model.NetworkInterface, error) {
	// This should never be called on Windows
	// The List() function checks runtime.GOOS before calling this
	return nil, nil
}

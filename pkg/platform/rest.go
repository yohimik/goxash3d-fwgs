//go:build !386
// +build !386

// This file is used on all platforms except i386 (32-bit x86).
// It contains platform-specific code that applies to non-i386 systems.

package platform

// Delay is a placeholder to match platform-specific implementations.
//
// On non-i386 systems, a delay may not be necessary, so this
// implementation is intentionally left empty to preserve consistent behavior
// across architectures without introducing unnecessary pauses.
func Delay() {
}

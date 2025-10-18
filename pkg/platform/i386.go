//go:build 386
// +build 386

// This file is specific to the i386 (32-bit x86) architecture.
// It includes platform-dependent code that is only built and used
// when targeting systems with this architecture.

package platform

import "time"

// Delay pauses execution briefly to simulate timing behavior.
//
// This is useful in situations such as emulating delays after
// network operations like recvfrom, or preventing tight loops
// from consuming too much CPU on certain system configurations.
func Delay() {
	time.Sleep(time.Millisecond)
}

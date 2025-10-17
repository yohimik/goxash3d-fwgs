//go:build 386
// +build 386

package platform

import "time"

func Delay() {
	time.Sleep(time.Millisecond)
}

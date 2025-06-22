package goxash3d_fwgs

import "C"
import "os"

const GameDir = "valve"

// SysStart Runs Xash3D main loop with default params
func (x *Xash3D) SysStart() int {
	return x.HostMain(os.Args, GameDir, 0)
}

package main

import goxash3d_fwgs "github.com/yohimik/goxash3d-fwgs/pkg"

func main() {
	goxash3d_fwgs.DefaultXash3D.RegisterNetCallbacks()

	go runSFU()

	goxash3d_fwgs.DefaultXash3D.SysStart()
}

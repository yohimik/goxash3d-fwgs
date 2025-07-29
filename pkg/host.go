package goxash3d_fwgs

/*
#cgo LDFLAGS: -L. -lxash -lpublic -lbuild_vcs -lm -lbacktrace
#include "xash.h"
#include <stdlib.h>

int host_Main( int argc, char **argv, const char *progname, int bChangeGame ) {
	return Host_Main( argc, argv, progname, bChangeGame, Sys_ChangeGame);
}

*/
import "C"
import (
	"unsafe"
)

// HostMain Runs Xash3D main loop
func (x *Xash3D) HostMain(args []string, gameDir string, bChangeGame int) int {
	argsCount := len(args)
	argc := Int(argsCount)
	argv := make([]*C.char, argsCount+1) // +1 for NULL terminator
	for i, arg := range args {
		argv[i] = C.CString(arg)
	}
	argv[argsCount] = nil

	cGameDir := C.CString(gameDir)
	defer func() {
		for _, arg := range argv {
			if arg != nil {
				C.free(unsafe.Pointer(arg))
			}
		}
		C.free(unsafe.Pointer(cGameDir))
	}()

	return int(C.host_Main(argc, &argv[0], cGameDir, Int(bChangeGame)))
}

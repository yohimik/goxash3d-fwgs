package main

/*
#cgo LDFLAGS: -L. -lxash -lpublic -lbuild_vcs -lm -lbacktrace
#include "xash.h"
#include <stdlib.h>
*/
import "C"

import (
	"os"
	"unsafe"
)

func main() {
	args := os.Args

	argc := C.int(len(args))
	argv := make([]*C.char, len(args)+1)

	for i, s := range args {
		argv[i] = C.CString(s)
		defer C.free(unsafe.Pointer(argv[i]))
	}
	argv[len(args)] = nil

	C.Launcher_Main(argc, &argv[0])
}

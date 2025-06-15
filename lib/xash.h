#ifndef ENGINE_H
#define ENGINE_H

#ifdef __cplusplus
extern "C" {
#endif

// These function signatures must match the actual definitions in the C source.
// You need to choose the real entry points from the engine code.
int Launcher_Main( int argc, char **argv );

#ifdef __cplusplus
}
#endif

#endif // ENGINE_H
#ifndef ENGINE_H
#define ENGINE_H

#ifdef __cplusplus
extern "C" {
#endif

typedef void( *pfnChangeGame )( const char *progname );

static void Sys_ChangeGame( const char *progname ) {}

int Host_Main( int argc, char **argv, const char *progname, int bChangeGame, pfnChangeGame func );

#ifdef __cplusplus
}
#endif

#endif // ENGINE_H
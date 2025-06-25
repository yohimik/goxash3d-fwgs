# Xash3D Dedicated Server (i386) Example

This project provides an example of running [the Xash3D-FWGS engine](https://github.com/FWGS/xash3d-fwgs) dedicated server inside an i386 Docker container. Compatible with original game DLLs.

## Overview

* Uses a CGO wrapper to interface Go with the native Xash3D engine libraries.
* Runs the server in a 32-bit (i386) Debian-based Docker container for compatibility with Xash3D.
* Fully compatible with all standard Xash3D-compatible plugins, including custom game DLLs and Metamod-like extensions.
* Demonstrates basic server startup, plugin binding, and command handling from Go.

## Compiling and running

### Using Docker

```shell
docker compose up -d
```
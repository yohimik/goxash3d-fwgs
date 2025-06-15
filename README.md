# Xash3D-FWGS CGO Wrapper

This repository provides a Go (Golang) [CGO](https://pkg.go.dev/cmd/cgo) wrapper for the [Xash3D-FWGS](https://github.com/FWGS/xash3d-fwgs) dedicated server, enabling integration with Go applications. The project compiles and runs inside a Docker container for ease of use and environment consistency.

## Features

1. CGO wrapper for native Xash3D-FWGS dedicated server
2. Fully containerized build and run process using Docker
3. Simplified integration of server-side game logic with Go projects

## Prerequisites

Docker installed on your system

## Getting Started

### Build & Run Using Docker

```shell
docker compose up -d
```
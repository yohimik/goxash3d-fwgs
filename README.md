# Xash3D-FWGS CGO Wrapper

This repository provides a Go (Golang) [CGO](https://pkg.go.dev/cmd/cgo) wrapper for the [Xash3D-FWGS](https://github.com/FWGS/xash3d-fwgs) dedicated server. It enables seamless integration of game logic and networking with modern Go applications.

It features Go-idiomatic engine structures, a custom UDP network layer (designed around Go channels), and a modular interface that can be extended to support modern networking stacks such as WebRTC.

> Note: Due to the underlying Xash3D engine's use of global variables, only a single instance of the engine structure can currently be created and managed per process. Multi-instance support is not available at this time (planned).

## Features

1. 🧩 Go-compatible engine structures with bound methods for direct manipulation
2. 🔗 Custom UDP networking layer using Go channels (WebRTC-ready)
3. 📦 Fully containerized build and runtime using Docker
4. 🧠 Enables writing server-side game logic entirely in Go
5. 📁 Real-world usage examples in the examples/ directory

## Getting Started

To get started quickly, check out the [/examples](./examples) directory for ready-made Go modules.
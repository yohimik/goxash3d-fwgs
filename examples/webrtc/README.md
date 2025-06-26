# Xash3D WebRTC Dedicated Server Example

This example illustrates how to implement a dedicated Xash3D server using a custom WebRTC-based networking layer in Go. It builds on the Go CGO wrapper around [the Xash3D-FWGS engine](https://github.com/FWGS/xash3d-fwgs) and demonstrates how to replace traditional UDP networking with WebRTC data channels.

The WebRTC networking layer is based on an example from [the Pion SFU-WS project](https://github.com/pion/example-webrtc-applications/tree/master/sfu-ws), a popular Go-based Selective Forwarding Unit library for WebRTC. This ensures robust and scalable peer communication using Go idioms.

## What This Example Shows

* ✅ Initializing the Xash3D engine from Go
* ✅ Registering engine callbacks and handling game logic
* ✅ Creating a WebRTC peer connection for client communication
* ✅ Routing engine network messages over a WebRTC data channel
* ✅ Managing networking via Go channels for concurrency and isolation

## Compiling and running

### Using Docker

```shell
docker compose up -d
```
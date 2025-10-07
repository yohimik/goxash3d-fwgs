package goxash3d_fwgs

// Xash3D Represents an instance of Xash3D-FWGS engine.
type Xash3D struct {
	Net Xash3DNetwork
}

// newXash3D Constructs new Xash3D instance.
// Private due to only single instance per process limitation.
func newXash3D() *Xash3D {
	return &Xash3D{}
}

var DefaultXash3D = newXash3D()

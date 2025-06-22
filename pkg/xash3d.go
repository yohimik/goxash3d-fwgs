package goxash3d_fwgs

// Xash3D Represents an instance of Xash3D-FWGS engine.
type Xash3D struct {
	*Xash3DNetwork
}

// newXash3D Constructs new Xash3D instance.
// Private due to only single instance per process limitation.
func newXash3D() *Xash3D {
	return &Xash3D{
		Xash3DNetwork: NewXash3DNetwork(),
	}
}

var DefaultXash3D = newXash3D()

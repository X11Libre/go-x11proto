package base

// ARC: a section of an ellipse. Angle1/Angle2 are in units of 1/64 degree.
type Arc struct {
	X      INT16
	Y      INT16
	Width  CARD16
	Height CARD16
	Angle1 INT16
	Angle2 INT16
}

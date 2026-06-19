package base

// CHARINFO: per-character font metrics.
type CharInfo struct {
	LeftSideBearing  INT16
	RightSideBearing INT16
	CharacterWidth   INT16
	Ascent           INT16
	Descent          INT16
	Attributes       CARD16
}

// FONTPROP: a font property (atom name + value).
type FontProp struct {
	Name  ATOM
	Value CARD32
}

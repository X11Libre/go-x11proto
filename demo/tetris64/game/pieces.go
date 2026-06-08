package game

type PieceType int

const (
	PieceI PieceType = iota + 1
	PieceO
	PieceT
	PieceS
	PieceZ
	PieceJ
	PieceL
)

type Piece struct {
	Type  PieceType
	Rot   int
	Cells [4][4]uint8
}

var pieceData = map[PieceType][4][4][4]uint8{
	PieceI: {
		{{0, 0, 0, 0}, {1, 1, 1, 1}, {0, 0, 0, 0}, {0, 0, 0, 0}},
		{{0, 0, 1, 0}, {0, 0, 1, 0}, {0, 0, 1, 0}, {0, 0, 1, 0}},
		{{0, 0, 0, 0}, {0, 0, 0, 0}, {1, 1, 1, 1}, {0, 0, 0, 0}},
		{{0, 1, 0, 0}, {0, 1, 0, 0}, {0, 1, 0, 0}, {0, 1, 0, 0}},
	},
	PieceO: {
		{{0, 2, 2, 0}, {0, 2, 2, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
		{{0, 2, 2, 0}, {0, 2, 2, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
		{{0, 2, 2, 0}, {0, 2, 2, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
		{{0, 2, 2, 0}, {0, 2, 2, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
	},
	PieceT: {
		{{0, 3, 0, 0}, {3, 3, 3, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
		{{0, 3, 0, 0}, {0, 3, 3, 0}, {0, 3, 0, 0}, {0, 0, 0, 0}},
		{{0, 0, 0, 0}, {3, 3, 3, 0}, {0, 3, 0, 0}, {0, 0, 0, 0}},
		{{0, 3, 0, 0}, {3, 3, 0, 0}, {0, 3, 0, 0}, {0, 0, 0, 0}},
	},
	PieceS: {
		{{0, 4, 4, 0}, {4, 4, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
		{{0, 4, 0, 0}, {0, 4, 4, 0}, {0, 0, 4, 0}, {0, 0, 0, 0}},
		{{0, 0, 0, 0}, {0, 4, 4, 0}, {4, 4, 0, 0}, {0, 0, 0, 0}},
		{{4, 0, 0, 0}, {4, 4, 0, 0}, {0, 4, 0, 0}, {0, 0, 0, 0}},
	},
	PieceZ: {
		{{5, 5, 0, 0}, {0, 5, 5, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
		{{0, 0, 5, 0}, {0, 5, 5, 0}, {0, 5, 0, 0}, {0, 0, 0, 0}},
		{{0, 0, 0, 0}, {5, 5, 0, 0}, {0, 5, 5, 0}, {0, 0, 0, 0}},
		{{0, 5, 0, 0}, {5, 5, 0, 0}, {5, 0, 0, 0}, {0, 0, 0, 0}},
	},
	PieceJ: {
		{{6, 0, 0, 0}, {6, 6, 6, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
		{{0, 6, 6, 0}, {0, 6, 0, 0}, {0, 6, 0, 0}, {0, 0, 0, 0}},
		{{0, 0, 0, 0}, {6, 6, 6, 0}, {0, 0, 6, 0}, {0, 0, 0, 0}},
		{{0, 6, 0, 0}, {0, 6, 0, 0}, {6, 6, 0, 0}, {0, 0, 0, 0}},
	},
	PieceL: {
		{{0, 0, 7, 0}, {7, 7, 7, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
		{{0, 7, 0, 0}, {0, 7, 0, 0}, {0, 7, 7, 0}, {0, 0, 0, 0}},
		{{0, 0, 0, 0}, {7, 7, 7, 0}, {7, 0, 0, 0}, {0, 0, 0, 0}},
		{{7, 7, 0, 0}, {0, 7, 0, 0}, {0, 7, 0, 0}, {0, 0, 0, 0}},
	},
}

var PieceColors = map[PieceType]uint32{
	PieceI: 0x00FFFF,
	PieceO: 0xFFFF00,
	PieceT: 0xAA00FF,
	PieceS: 0x00FF00,
	PieceZ: 0xFF0000,
	PieceJ: 0x0000FF,
	PieceL: 0xFF8800,
}

func NewPiece(t PieceType) Piece {
	return Piece{Type: t, Cells: pieceData[t][0]}
}

func (p *Piece) Rotate(dir int) {
	p.Rot = (p.Rot + dir + 4) % 4
	p.Cells = pieceData[p.Type][p.Rot]
}

func (p Piece) Get(x, y int) uint8 {
	if x < 0 || x >= 4 || y < 0 || y >= 4 {
		return 0
	}
	return p.Cells[y][x]
}

// Bounds returns the bounding box (inclusive) of the set cells within the 4x4
// piece grid. For an empty piece maxX/maxY are -1.
func (p Piece) Bounds() (minX, minY, maxX, maxY int) {
	minX, minY = 4, 4
	maxX, maxY = -1, -1
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if p.Get(x, y) != 0 {
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	return
}

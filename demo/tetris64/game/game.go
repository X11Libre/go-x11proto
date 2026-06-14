package game

import "math/rand"

type State struct {
	Board    [21][10]uint8
	Current  Piece
	Next     PieceType
	CX, CY   int
	Score    int
	Lines    int
	Level    int
	GameOver bool

	bag    []PieceType
	bagIdx int
}

var pieceTypes = []PieceType{PieceI, PieceO, PieceT, PieceS, PieceZ, PieceJ, PieceL}

func New() *State {
	s := &State{}
	s.fillBag()
	s.Next = s.drawFromBag()
	s.SpawnPiece()
	return s
}

func (s *State) fillBag() {
	s.bag = make([]PieceType, len(pieceTypes))
	copy(s.bag, pieceTypes)
	rand.Shuffle(len(s.bag), func(i, j int) {
		s.bag[i], s.bag[j] = s.bag[j], s.bag[i]
	})
	s.bagIdx = 0
}

func (s *State) drawFromBag() PieceType {
	if s.bagIdx >= len(s.bag) {
		s.fillBag()
	}
	t := s.bag[s.bagIdx]
	s.bagIdx++
	return t
}

func (s *State) Reset() {
	clear(s.Board[:])
	s.Score = 0
	s.Lines = 0
	s.Level = 0
	s.GameOver = false
	s.fillBag()
	s.Next = s.drawFromBag()
	s.SpawnPiece()
}

func (s *State) SpawnPiece() {
	s.Current = NewPiece(s.Next)
	s.Next = s.drawFromBag()
	s.CX = 3
	s.CY = 0
	if s.Collides(0, 0, s.Current.Rot) {
		s.GameOver = true
	}
}

func (s *State) Collides(dx, dy, rot int) bool {
	p := s.Current
	p.Rot = rot
	p.Cells = pieceData[p.Type][rot]
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if p.Get(x, y) == 0 {
				continue
			}
			bx := s.CX + x + dx
			by := s.CY + y + dy
			if bx < 0 || bx >= 10 || by >= 21 {
				return true
			}
			if by < 0 {
				continue
			}
			if s.Board[by][bx] != 0 {
				return true
			}
		}
	}
	return false
}

func (s *State) MoveLeft() bool {
	if !s.Collides(-1, 0, s.Current.Rot) {
		s.CX--
		return true
	}
	return false
}

func (s *State) MoveRight() bool {
	if !s.Collides(1, 0, s.Current.Rot) {
		s.CX++
		return true
	}
	return false
}

func (s *State) MoveDown() bool {
	if !s.Collides(0, 1, s.Current.Rot) {
		s.CY++
		return true
	}
	return false
}

func (s *State) Rotate() {
	nr := (s.Current.Rot + 1) % 4
	if !s.Collides(0, 0, nr) {
		s.Current.Rotate(1)
		return
	}
	// wall kick: try shifting left/right
	for _, kick := range []int{-1, 1, -2, 2} {
		if !s.Collides(kick, 0, nr) {
			s.CX += kick
			s.Current.Rotate(1)
			return
		}
	}
	// try shifting up (for I-piece in tight spaces)
	if !s.Collides(0, -1, nr) {
		s.CY--
		s.Current.Rotate(1)
		return
	}
}

func (s *State) HardDrop() {
	for s.MoveDown() {
	}
	s.LockPiece()
}

func (s *State) LockPiece() {
	p := s.Current
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if p.Get(x, y) == 0 {
				continue
			}
			bx := s.CX + x
			by := s.CY + y
			if by >= 0 && by < 21 && bx >= 0 && bx < 10 {
				s.Board[by][bx] = p.Get(x, y)
			}
		}
	}
	s.CheckLines()
	s.SpawnPiece()
}

func (s *State) CheckLines() {
	cleared := 0
	for y := 20; y >= 0; {
		full := true
		for x := 0; x < 10; x++ {
			if s.Board[y][x] == 0 {
				full = false
				break
			}
		}
		if full {
			for y2 := y; y2 > 0; y2-- {
				s.Board[y2] = s.Board[y2-1]
			}
			s.Board[0] = [10]uint8{}
			cleared++
		} else {
			y--
		}
	}
	if cleared > 0 {
		s.Lines += cleared
		s.Level = s.Lines / 10
		scoring := []int{0, 10, 30, 50, 100}
		if cleared >= len(scoring) {
			cleared = len(scoring) - 1
		}
		s.Score += scoring[cleared] * (s.Level + 1)
	}
}

func (s *State) DropDistance() int {
	d := 0
	for !s.Collides(0, d+1, s.Current.Rot) {
		d++
	}
	return d
}

func (s *State) TickSpeed() int64 {
	ms := int64(1000 - s.Level*100)
	if ms < 50 {
		ms = 50
	}
	return ms
}

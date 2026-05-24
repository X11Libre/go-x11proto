package base

func RoundFullUnits(sz uint) uint {
	// could be done via mask operations -> perhaps a bit faster
	u := sz >> 2
	u2 := u << 2
	if u2 == sz {
		return sz
	}
	return u2 + 4
}

func UnitsToBytes(units CARD32) uint {
	return uint(units * 4)
}

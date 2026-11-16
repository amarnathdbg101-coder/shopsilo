package tinylfu

// cmSketch implements a 4-bit Count-Min Sketch frequency estimator with periodic aging.
type cmSketch struct {
	table []byte
	mask  uint64
	size  int
}

func newCMSketch(numCounters int) *cmSketch {
	if numCounters < 64 {
		numCounters = 64
	}
	// Round up to power of 2
	size := 1
	for size < numCounters {
		size <<= 1
	}
	return &cmSketch{
		table: make([]byte, size/2), // 2 counters per byte (4-bit nibbles)
		mask:  uint64(size - 1),
		size:  size,
	}
}

func (s *cmSketch) increment(hash uint64) {
	h1 := hash
	h2 := hash >> 32

	for i := 0; i < 4; i++ {
		combined := (h1 + uint64(i)*h2) & s.mask
		byteIdx := combined >> 1
		isUpper := (combined & 1) == 1

		val := s.table[byteIdx]
		if isUpper {
			count := (val >> 4) & 0x0F
			if count < 15 {
				s.table[byteIdx] = (val & 0x0F) | ((count + 1) << 4)
			}
		} else {
			count := val & 0x0F
			if count < 15 {
				s.table[byteIdx] = (val & 0xF0) | (count + 1)
			}
		}
	}
}

func (s *cmSketch) estimate(hash uint64) uint8 {
	h1 := hash
	h2 := hash >> 32
	minCount := uint8(15)

	for i := 0; i < 4; i++ {
		combined := (h1 + uint64(i)*h2) & s.mask
		byteIdx := combined >> 1
		isUpper := (combined & 1) == 1

		val := s.table[byteIdx]
		var count uint8
		if isUpper {
			count = (val >> 4) & 0x0F
		} else {
			count = val & 0x0F
		}

		if count < minCount {
			minCount = count
		}
	}

	return minCount
}

// reset halves all counters (aging factor).
func (s *cmSketch) reset() {
	for i := range s.table {
		b := s.table[i]
		upper := ((b >> 4) & 0x0F) >> 1
		lower := (b & 0x0F) >> 1
		s.table[i] = (upper << 4) | lower
	}
}

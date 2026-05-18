package id

import (
	"fmt"
	"math/rand/v2"
	"time"
)

// --- Execution Methods ---

// Must generates an ID of the specified length or panics on error.
func (g *Generator) Must(length int) string {
	res, err := g.Sortable(length)
	if err != nil {
		panic(err)
	}
	return res
}

// Sortable generates a unique, sortable (monotonically increasing) ID.
func (g *Generator) Sortable(lengths ...int) (string, error) {
	if g.err != nil {
		return "", g.err
	}

	length := 22
	if len(lengths) > 0 && lengths[0] > 0 {
		length = lengths[0]
	}

	cs := g.charset
	if len(cs) == 0 {
		cs = charset62
	}

	// Always use time-based sortable strategy for Generate.
	if length < 22 {
		return genShortAdaptive(length, cs, g.epoch)
	}

	return generate(cs, length, g.epoch)
}

// Random creates a Random ID (No Repeats by default).
func (g *Generator) Random(lengths ...int) (string, error) {
	if g.err != nil {
		return "", g.err
	}

	length := 22
	if len(lengths) > 0 && lengths[0] > 0 {
		length = lengths[0]
	}

	cs := g.charset
	if len(cs) == 0 {
		cs = charset62
	}

	if g.repeat {
		return genRandomRepeated(length, cs), nil
	}
	return genPureRandom(length, cs), nil
}

// --- Internal Engine ---

// genPureRandom generates an ID by shuffling the charset.
// Uses math/rand/v2 for lock-free, zero-alloc speed.
func genPureRandom(length int, charset []byte) string {
	l := len(charset)
	if length > l {
		length = l
	}

	avail := make([]byte, l)
	copy(avail, charset)

	// Partial Shuffle
	for i := 0; i < length; i++ {
		j := i + rand.N(l-i)
		avail[i], avail[j] = avail[j], avail[i]
	}

	return string(avail[:length])
}

// genRandomRepeated generates a random ID allowing repeating characters.
func genRandomRepeated(length int, charset []byte) string {
	l := len(charset)
	res := make([]byte, length)
	for i := 0; i < length; i++ {
		res[i] = charset[rand.N(l)]
	}
	return string(res)
}

// genShortAdaptive uses a custom epoch to generate sortable IDs.
func genShortAdaptive(length int, charset []byte, instanceEpoch int64) (string, error) {
	charsetLen := len(charset)
	avail := make([]byte, charsetLen)
	copy(avail, charset)

	now := time.Now().UnixMilli()
	targetEpoch := instanceEpoch
	if targetEpoch == 0 {
		targetEpoch = since.UnixMilli()
	}
	if now < targetEpoch {
		now = targetEpoch
	}

	t := uint64(now - targetEpoch)
	t = (t << 12) | uint64(seq.Add(1)&0xfff)

	timeLen := 0
	capacity := uint64(1)
	for i := 0; i < charsetLen; i++ {
		multiplier := uint64(charsetLen - i)
		if capacity > ^uint64(0)/multiplier { // Prevent overflow
			break
		}
		capacity *= multiplier
		timeLen++
	}

	if timeLen > length {
		return "", fmt.Errorf("id: length %d insufficient to encode timestamp (needed %d)", length, timeLen)
	}

	tsChars, activeLen := encodeTimePart(t, avail, timeLen, charsetLen)
	randomLen := length - timeLen

	shuffledAvail := shuffleRemaining(avail[:activeLen])
	if randomLen > len(shuffledAvail) {
		randomLen = len(shuffledAvail)
	}

	return string(tsChars) + string(shuffledAvail[:randomLen]), nil
}

// generate uses UnixNano to create a globally unique sortable ID.
func generate(charset []byte, totalLen int, instanceEpoch int64) (string, error) {
	charsetLen := len(charset)
	avail := make([]byte, charsetLen)
	copy(avail, charset)

	now := time.Now().UnixNano()
	targetEpoch := instanceEpoch
	if targetEpoch == 0 {
		targetEpoch = since.UnixMilli()
	}
	targetEpochNano := targetEpoch * 1000000

	if now < targetEpochNano {
		now = targetEpochNano
	}
	t := uint64(now - targetEpochNano)
	t = (t &^ 0xfff) | uint64(seq.Add(1)&0xfff)

	timeLen := 0
	capacity := uint64(1)
	for i := 0; i < charsetLen; i++ {
		multiplier := uint64(charsetLen - i)
		if capacity > ^uint64(0)/multiplier {
			break
		}
		capacity *= multiplier
		timeLen++
	}

	if timeLen > totalLen {
		return "", fmt.Errorf("id: length %d insufficient to encode timestamp (needed %d)", totalLen, timeLen)
	}

	tsChars, activeLen := encodeTimePart(t, avail, timeLen, charsetLen)
	randomLen := totalLen - timeLen

	shuffledAvail := shuffleRemaining(avail[:activeLen])
	if randomLen > len(shuffledAvail) {
		randomLen = len(shuffledAvail)
	}

	return string(tsChars) + string(shuffledAvail[:randomLen]), nil
}

// --- Helpers ---

// encodeTimePart extracts time encoding and returns the used characters and the new active length of avail.
func encodeTimePart(t uint64, avail []byte, timeLen int, charsetLen int) ([]byte, int) {
	var resArr [256]byte
	var idxArr [256]int
	
	res := resArr[:timeLen]
	idxs := idxArr[:timeLen]

	currentT := t
	startBase := uint64(charsetLen - (timeLen - 1))
	for i := timeLen - 1; i >= 0; i-- {
		base := startBase + uint64((timeLen-1)-i)
		idxs[i] = int(currentT % base)
		currentT /= base
	}

	activeLen := charsetLen
	for i, idx := range idxs {
		res[i] = avail[idx]
		// In-place byte shift (Zero-allocation instead of append/slicing)
		for j := idx; j < activeLen-1; j++ {
			avail[j] = avail[j+1]
		}
		activeLen--
	}
	return res, activeLen
}

func shuffleRemaining(avail []byte) []byte {
	limit := len(avail)
	for i := limit - 1; i > 0; i-- {
		j := rand.N(i + 1)
		avail[i], avail[j] = avail[j], avail[i]
	}
	return avail
}

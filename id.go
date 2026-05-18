// Package id provides a suite of high-performance, unique, and optionally sortable ID generators.
// It supports multiple formats including Base62, Base58, Base36, and pure numeric IDs.
package id

import (
	"fmt"
	"sync/atomic"
	"time"
)

var (
	// Default Epoch set to 2025-12-16 08:35:00 UTC.
	// Used for generating sortable short IDs.
	since = time.Date(2025, 12, 16, 8, 35, 0, 0, time.UTC)

	// Predefined charsets for different ID flavors.
	digits = "0123456789"
	lower  = "abcdefghijklmnopqrstuvwxyz"
	upper  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	// Base58 excludes ambiguous characters: 0, O, I, l
	base58 = "123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"

	charset10 = []byte(digits)
	charset26 = []byte(lower)
	charset36 = []byte(digits + lower)
	charset58 = []byte(base58)
	charset62 = []byte(digits + lower + upper)

	seq atomic.Uint32
)

// --- Global Helpers ---

func Charset(n int) *Generator       { return New().Charset(n) }
func Custom(chars string) *Generator { return New().Custom(chars) }

func Entity() string    { return Charset(36).Must(36) }
func Short() string     { return Charset(62).Repeat(true).Must(6) }
func Shortlink() string { return Charset(58).Must(8) }

// DecodeEntity parses a standard 36-character Entity ID and returns its generation time.
func DecodeEntity(idStr string) (time.Time, error) {
	if len(idStr) != 36 {
		return time.Time{}, fmt.Errorf("id: invalid length for Entity ID, expected 36")
	}

	charset := charset36
	charsetLen := len(charset)

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

	timeChars := idStr[:timeLen]
	avail := make([]byte, charsetLen)
	copy(avail, charset)

	idxs := make([]int, timeLen)
	activeLen := charsetLen
	for i := 0; i < timeLen; i++ {
		char := timeChars[i]
		idx := -1
		for j := 0; j < activeLen; j++ {
			if avail[j] == char {
				idx = j
				break
			}
		}
		if idx == -1 {
			return time.Time{}, fmt.Errorf("id: invalid character '%c' in ID", char)
		}
		idxs[i] = idx
		for j := idx; j < activeLen-1; j++ {
			avail[j] = avail[j+1]
		}
		activeLen--
	}

	currentT := uint64(0)
	startBase := uint64(charsetLen - (timeLen - 1))
	for i := 0; i < timeLen; i++ {
		base := startBase + uint64((timeLen-1)-i)
		currentT = currentT*base + uint64(idxs[i])
	}

	// Revert the 12-bit sequence mask to get the true nanosecond delta
	nanoDelta := currentT &^ 0xfff
	targetEpochNano := since.UnixMilli() * 1000000
	timestampNano := targetEpochNano + int64(nanoDelta)

	return time.Unix(0, timestampNano).UTC(), nil
}

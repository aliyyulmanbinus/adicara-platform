package domain

import "errors"

// Limits of the projection parameters (05 BE 3.4). A whole percentage, not a decimal factor: 200 x 1.1 becomes 221
// with float64 (220.00000000000003 is rounded up) while the answer is 220.
const (
	DefaultBufferPercent = 25
	MaxBufferPercent     = 100
	MaxBufferPax         = 1000
)

var ErrBuffer = errors.New("buffer out of range")

// RecommendedPortions = ceil(confirmedPax x (100 + bufferPercent) / 100) + bufferPax, in integer arithmetic.
func RecommendedPortions(confirmedPax, bufferPercent, bufferPax int) (int, error) {
	if confirmedPax < 0 || bufferPercent < 0 || bufferPercent > MaxBufferPercent || bufferPax < 0 || bufferPax > MaxBufferPax {
		return 0, ErrBuffer
	}
	return (confirmedPax*(100+bufferPercent)+99)/100 + bufferPax, nil
}

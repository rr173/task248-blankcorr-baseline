// Package correct implements the numerical core of the blank-correction
// service: instrument-drift modelling from standards, blank subtraction
// combined with drift correction, and conversion of a corrected isotope
// ratio into an age with first-order uncertainty propagation.
package correct

import "math"

// CorrectedRatio computes the blank- and drift-corrected isotope ratio:
//
//	R = (x - b) * k
//
// where x is the sample ratio, b the matched blank ratio and k the
// instrument drift (recovery) factor. Uncertainties are propagated linearly:
//
//	σ_R² = k²·(σ_x² + σ_b²) + (x - b)²·σ_k²
func CorrectedRatio(x, sigX, b, sigB, k, sigK float64) (r, sigR float64) {
	diff := x - b
	r = diff * k
	sigR2 := k*k*(sigX*sigX+sigB*sigB) + diff*diff*(sigK*sigK)
	if sigR2 < 0 {
		sigR2 = 0
	}
	sigR = math.Sqrt(sigR2)
	return r, sigR
}

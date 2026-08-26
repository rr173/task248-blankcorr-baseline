package correct

import (
	"fmt"
	"math"
)

// Age converts a corrected ratio into an age for a decay system whose ratio
// decays as R(t) = R0 · exp(-λ t):
//
//	t = (1/λ) · ln(R0 / R)
//
// The 1-sigma uncertainty is propagated (∂t/∂R = -(1/λ)·(1/R)):
//
//	σ_t = (1/λ) · (1/R) · σ_R
//
// Age and its uncertainty are returned in the same time unit as lambda.
func Age(lambda, r0, r, sigR float64) (age, sigAge float64, err error) {
	if lambda <= 0 || r0 <= 0 || r <= 0 {
		return 0, 0, fmt.Errorf("age undefined for non-positive inputs (lambda=%g r0=%g r=%g)", lambda, r0, r)
	}
	age = (1.0 / lambda) * math.Log(r0/r)
	sigAge = (1.0 / lambda) * (1.0 / r) * sigR
	return age, sigAge, nil
}

// AgeWithInterval returns the central age, its 1-sigma uncertainty and the
// 2-sigma interval [low, high] used for anomaly detection.
func AgeWithInterval(lambda, r0, r, sigR float64) (age, sigAge, low, high float64, err error) {
	a, sa, err := Age(lambda, r0, r, sigR)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return a, sa, a - 2*sa, a + 2*sa, nil
}

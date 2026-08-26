package measure

import (
	"fmt"

	"task248-blankcorr/internal/model"
)

// BlankSuspicion reports whether a blank measurement looks contaminated. A
// blank should contribute almost no signal; if its ratio is far above the
// typical blank level it is a candidate for exclusion. This is a heuristic
// advisory used by the review step, not a hard rule.
func BlankSuspicion(b *model.Measurement, medianBlank float64, threshold float64) (bool, string) {
	if b.Kind != model.KindBlank {
		return false, ""
	}
	if medianBlank <= 0 {
		// no baseline yet: flag only extreme values
		if b.Ratio > threshold {
			return true, fmt.Sprintf("blank ratio %.4g exceeds absolute threshold %.4g", b.Ratio, threshold)
		}
		return false, ""
	}
	if b.Ratio > medianBlank*threshold {
		return true, fmt.Sprintf("blank ratio %.4g exceeds %.1fx median blank %.4g", b.Ratio, threshold, medianBlank)
	}
	return false, ""
}

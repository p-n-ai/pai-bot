package progress

import "math"

// SM2Result holds the output of an SM-2 calculation.
type SM2Result struct {
	Repetitions  int
	EaseFactor   float64
	IntervalDays int
}

// SM2Calculate implements the SuperMemo 2 algorithm.
// quality: 0-5 (0=blackout, 5=perfect)
func SM2Calculate(quality, repetitions int, easeFactor float64, intervalDays int) SM2Result {
	if quality < 3 {
		// Failed — reset.
		return SM2Result{
			Repetitions:  0,
			EaseFactor:   math.Max(1.3, easeFactor-0.2),
			IntervalDays: 1,
		}
	}

	// Update ease factor.
	newEF := easeFactor + (0.1 - float64(5-quality)*(0.08+float64(5-quality)*0.02))
	if newEF < 1.3 {
		newEF = 1.3
	}

	// Calculate new interval.
	var newInterval int
	newReps := repetitions + 1

	switch repetitions {
	case 0:
		newInterval = 1
	case 1:
		newInterval = 6
	default:
		newInterval = int(math.Round(float64(intervalDays) * newEF))
	}

	return SM2Result{
		Repetitions:  newReps,
		EaseFactor:   newEF,
		IntervalDays: newInterval,
	}
}

// DeltaToQuality converts a mastery delta (0.0-1.0) to an SM-2 quality rating (0-5).
func DeltaToQuality(delta float64) int {
	q := int(math.Round(delta * 5))
	if q < 0 {
		return 0
	}
	if q > 5 {
		return 5
	}
	return q
}

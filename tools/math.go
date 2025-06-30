package tools

import "golang.org/x/exp/constraints"

func Clamp[T constraints.Ordered](val, min, max T) T {
	if val < min {
		return min
	} else if val > max {
		return max
	}
	return val
}

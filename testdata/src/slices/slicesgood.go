package slices

func sliceIndexCheck() int64 {
	x := make([]int64, 4, 16)

	var b int64
	b = 0
	if 30 < len(x) {
		b = x[30]
	}

	return b
}

func sliceIndexCheckCap() int64 {
	x := make([]int64, 4, 16)

	var b int64
	b = 0
	if 30 < cap(x) {
		b = x[30]
	}

	return b
}

func sliceIndexCheckCapRHS() int64 {
	x := make([]int64, 4, 16)

	var b int64
	b = 0
	if cap(x) > 30 {
		b = x[30]
	}

	return b
}

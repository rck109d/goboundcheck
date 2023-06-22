package slices

func sliceIndexNoCheck() int64 {
	x := make([]int64, 4, 16)
	b := x[30] // want "Slice or array access is not enclosed in an if-statement that validates capacity!"
	return b
}

func sliceIndexNoCheckMultiple() int64 {
	x := make([]int64, 4, 16)
	a := x[10]  // want "Slice or array access is not enclosed in an if-statement that validates capacity!"
	b := x[30]  // want "Slice or array access is not enclosed in an if-statement that validates capacity!"
	c := x[200] // want "Slice or array access is not enclosed in an if-statement that validates capacity!"
	return a * b * c
}

func sliceIndexNoCheckInIf() int64 {
	x := make([]int64, 4, 16)

	if true {
		return x[1000] // want "Slice or array access is not enclosed in an if-statement that validates capacity!"
	}

	return x[10] // want "Slice or array access is not enclosed in an if-statement that validates capacity!"
}

func sliceIndexOtherCallIfEq() int64 {
	x := make([]int64, 4, 16)

	if sliceIndexCheck() == 0 {
		return x[1000] // want "Slice or array access is not enclosed in an if-statement that validates capacity!"
	} else {
		return 99
	}
}

func sliceIndexCheckOtherCap() int64 {
	x := make([]int64, 4, 16)
	y := []int64{}

	if cap(y) > 1 {
		return x[1] // want "Slice or array access is not enclosed in an if-statement that validates capacity!"
	} else {
		return 99
	}
}

func sliceExprOtherCap() []int64 {
	x := make([]int64, 4, 16)
	y := []int64{}

	if cap(y) > 2 && 1 == 1 {
		return x[1:2] // want "Slice or array access is not enclosed in an if-statement that validates capacity!"
	} else {
		return []int64{1, 2}
	}
}

func sliceExprOtherCall() []int64 {
	x := make([]int64, 4, 16)
	y := []int64{}

	if cap(y) > 2 || 1 == 1 || cap(append(y, []int64{1, 2}...)) == 2 {
		return x[1:2] // want "Slice or array access is not enclosed in an if-statement that validates capacity!"
	} else {
		return []int64{1, 2}
	}
}

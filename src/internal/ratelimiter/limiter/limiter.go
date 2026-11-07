package limiter

func CheckLimit(limit *int) (bool, int) {
	if *limit > 0 {
		*limit -= 1
		return true, *limit
	} else {
		return false, *limit
	}
}

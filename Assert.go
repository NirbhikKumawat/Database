package Database

func check(cond bool) {
	if !cond {
		panic("assertion failed")
	}
}

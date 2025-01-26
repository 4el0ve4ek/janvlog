package main

func PairFirst[T, U any](t T, u U) T {
	return t
}

func PairSecond[T, U any](t T, u U) U {
	return u
}

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

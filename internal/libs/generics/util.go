package generics

func First[T, U any](t T, u U) T {
	return t
}

func Second[T, U any](t T, u U) U {
	return u
}

func Must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}

	return t
}

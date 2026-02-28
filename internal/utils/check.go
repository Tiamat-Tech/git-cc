package utils

func Check(err error) {
	if err != nil {
		panic(err)
	}
}
func Must[T any](t T, err error) T {
	Check(err)
	return t
}

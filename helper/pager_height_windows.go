//go:build windows

package helper

func stdoutHeight() (int, bool) {
	return 0, false
}

// StdoutWidth defaults to 80 on Windows when size cannot be queried.
func StdoutWidth() int {
	return 80
}

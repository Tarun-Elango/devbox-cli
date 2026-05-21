package cmd

import "syscall"

const (
	ioctlReadTermios  = syscall.TIOCGWINSZ // unused sentinel for Windows
	ioctlWriteTermios = syscall.TIOCSWINSZ
)

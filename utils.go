package inclient

import (
	"golang.org/x/sys/unix"
)

func getStdoutFd() int {
	return unix.Stdout
}

func getStdinFd() int {
	return unix.Stdin
}

package utils

import (
	"github.com/mattn/go-isatty"
	"os"
)

func FileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		panic(err)
	}
	return info.IsDir()
}

// Isatty checks if a given file is a terminal or not
func Isatty(f *os.File) bool {
	if isatty.IsTerminal(f.Fd()) {
		return true
	} else {
		return false
	}
}

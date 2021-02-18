package utils

import "os"

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

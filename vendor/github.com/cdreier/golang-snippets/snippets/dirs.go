package snippets

import (
	"log"
	"os"
)

// EnsureDir checks if given directory exist, creates if not
func EnsureDir(dir string) {
	if !DirExist(dir) {
		err := os.Mkdir(dir, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// DirExist checks if directory exist
func DirExist(dir string) bool {
	_, err := os.Stat(dir)
	if err == nil {
		return true
	}
	return !os.IsNotExist(err)
}

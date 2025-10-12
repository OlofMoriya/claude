package logger

import (
	"log"
	"os"

	"github.com/mitchellh/go-homedir"
)

// Global logger - accessible from anywhere
var Debug *log.Logger

// Init sets up the logger - call this from main
func Init(filename string) error {
	expandedPath, err := homedir.Expand(filename)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(expandedPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	Debug = log.New(f, "", log.LstdFlags|log.Lshortfile)
	Debug.Println("Logger initialized")
	return nil
}

package logger

import (
	"log"
	"os"
)

// Global logger - accessible from anywhere
var Debug *log.Logger

// Init sets up the logger - call this from main
func Init(filename string) error {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	Debug = log.New(f, "", log.LstdFlags|log.Lshortfile)
	Debug.Println("Logger initialized")
	return nil
}

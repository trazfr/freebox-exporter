package log

import (
	"io/ioutil"
	"log"
	"os"
)

var (
	Debug   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

func InitDebug() {
	Debug = log.New(os.Stdout,
		"DEBUG: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Info = log.New(os.Stdout,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(os.Stdout,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Error = log.New(os.Stderr,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}

func Init() {
	Debug = log.New(ioutil.Discard, "", 0)
	Info = log.New(os.Stdout, "", 0)
	Warning = log.New(os.Stdout, "", 0)
	Error = log.New(os.Stderr, "", 0)
}

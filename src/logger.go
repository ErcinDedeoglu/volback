package main

import (
	"fmt"
	"log"
	"os"
)

var logger *log.Logger

const indent = "\t"

func init() {
	logger = log.New(os.Stdout, "", log.LstdFlags)
}

func logHeader(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	logger.Printf("%s", msg)
}

func logStep(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	logger.Printf("%s", msg)
}

func logSubStep(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	logger.Printf("%s%s", indent, msg)
}

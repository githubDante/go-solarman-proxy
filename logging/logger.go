package logging

import (
	"fmt"
	"github.com/fatih/color"
	"os"
	"time"
)

func getT() string {
	return time.Now().Format("[2006-01-02 15:04:05.000]")
}

var debug = false

func EnableDebug() {
	debug = true
}

var green = color.New(color.FgHiGreen).FprintfFunc()
var yellow = color.New(color.FgHiYellow).FprintfFunc()
var red = color.New(color.FgHiRed).FprintfFunc()
var blue = color.New(color.FgHiBlue).FprintfFunc()

// LogInfof Info message
func LogInfof(message string, args ...any) {
	msg := fmt.Sprintf(message, args...)
	green(os.Stdout, "%25s [INFO] - %s", getT(), msg)
}

// LogErrorf Error message
func LogErrorf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	red(os.Stdout, "%25s [ERROR] - %s", getT(), msg)
}

// LogWarnf Warning message
func LogWarnf(message string, args ...any) {
	msg := fmt.Sprintf(message, args...)
	yellow(os.Stdout, "%25s [WARN] - %s", getT(), msg)
}

// LogDebugf Debug message
func LogDebugf(message string, args ...any) {
	if !debug {
		return
	}
	msg := fmt.Sprintf(message, args...)
	blue(os.Stdout, "%25s [DEBUG] - %s", getT(), msg)
}

package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

var (
	levelNames = map[Level]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
	}
	currentLevel = INFO
	logger       *log.Logger
)

// Init initializes the logger with the specified level and log directory
func Init(levelStr string, logDir string) error {
	// Parse log level
	switch strings.ToLower(levelStr) {
	case "debug":
		currentLevel = DEBUG
	case "info":
		currentLevel = INFO
	case "warn":
		currentLevel = WARN
	case "error":
		currentLevel = ERROR
	default:
		currentLevel = INFO
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file with date
	logFile := filepath.Join(logDir, fmt.Sprintf("feishu-github-tracker-%s.log", time.Now().Format("2006-01-02")))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Write to both file and stdout
	multiWriter := io.MultiWriter(os.Stdout, file)
	logger = log.New(multiWriter, "", log.LstdFlags)

	return nil
}

func logMessage(level Level, format string, v ...interface{}) {
	if level < currentLevel {
		return
	}
	msg := fmt.Sprintf(format, v...)
	logger.Printf("[%s] %s", levelNames[level], msg)
}

func Debug(format string, v ...interface{}) {
	logMessage(DEBUG, format, v...)
}

func Info(format string, v ...interface{}) {
	logMessage(INFO, format, v...)
}

func Warn(format string, v ...interface{}) {
	logMessage(WARN, format, v...)
}

func Error(format string, v ...interface{}) {
	logMessage(ERROR, format, v...)
}

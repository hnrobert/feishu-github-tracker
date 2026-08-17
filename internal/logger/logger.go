package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	logger       = log.New(io.Discard, "", log.LstdFlags)
	stateMu      sync.RWMutex
	fileWriter   *dailyWriter
)

type dailyWriter struct {
	mu   sync.Mutex
	dir  string
	date string
	file *os.File
}

func (w *dailyWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	date := time.Now().Format("2006-01-02")
	if w.file == nil || w.date != date {
		if w.file != nil {
			_ = w.file.Close()
			w.file = nil
		}
		file, err := os.OpenFile(filepath.Join(w.dir, "feishu-github-tracker-"+date+".log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return 0, err
		}
		w.date = date
		w.file = file
	}
	return w.file.Write(p)
}

func (w *dailyWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

// Init initializes the logger with the specified level and log directory
func Init(levelStr string, logDir string) error {
	// Parse log level
	level := INFO
	switch strings.ToLower(levelStr) {
	case "debug":
		level = DEBUG
	case "info":
		level = INFO
	case "warn":
		level = WARN
	case "error":
		level = ERROR
	default:
		level = INFO
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	stateMu.Lock()
	defer stateMu.Unlock()
	currentLevel = level
	if fileWriter != nil {
		_ = fileWriter.Close()
	}
	fileWriter = &dailyWriter{dir: logDir}
	// Write to both file and stdout. The file is opened lazily so rotation can
	// switch to the next date without restarting the process.
	logger = log.New(io.MultiWriter(os.Stdout, fileWriter), "", log.LstdFlags)

	return nil
}

// Close releases the current log file. It is safe to call more than once.
func Close() error {
	stateMu.Lock()
	defer stateMu.Unlock()
	if fileWriter == nil {
		return nil
	}
	err := fileWriter.Close()
	fileWriter = nil
	logger = log.New(io.Discard, "", log.LstdFlags)
	return err
}

func logMessage(level Level, format string, v ...any) {
	stateMu.RLock()
	defer stateMu.RUnlock()
	if level < currentLevel {
		return
	}
	msg := fmt.Sprintf(format, v...)
	logger.Printf("[%s] %s", levelNames[level], msg)
}

func Debug(format string, v ...any) {
	logMessage(DEBUG, format, v...)
}

func Info(format string, v ...any) {
	logMessage(INFO, format, v...)
}

func Warn(format string, v ...any) {
	logMessage(WARN, format, v...)
}

func Error(format string, v ...any) {
	logMessage(ERROR, format, v...)
}

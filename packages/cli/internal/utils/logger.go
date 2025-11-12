package utils

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type Logger struct {
	name   string
	level  string
	output *log.Logger
}

func NewLogger(name, level string) *Logger {
	return &Logger{
		name:   name,
		level:  strings.ToUpper(level),
		output: log.New(os.Stdout, "", 0),
	}
}

func (l *Logger) SetLevel(level string) {
	l.level = strings.ToUpper(level)
}

func (l *Logger) Debug(msg string, args ...interface{}) {
	if l.shouldLog("DEBUG") {
		l.log("DEBUG", msg, args...)
	}
}

func (l *Logger) Info(msg string, args ...interface{}) {
	if l.shouldLog("INFO") {
		l.log("INFO", msg, args...)
	}
}

func (l *Logger) Warn(msg string, args ...interface{}) {
	if l.shouldLog("WARN") {
		l.log("WARN", msg, args...)
	}
}

func (l *Logger) Error(msg string, args ...interface{}) {
	if l.shouldLog("ERROR") {
		l.log("ERROR", msg, args...)
	}
}

func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.log("FATAL", msg, args...)
	os.Exit(1)
}

func (l *Logger) log(level, msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	
	levelColor := l.getLevelColor(level)
	resetColor := "\033[0m"
	
	formattedMsg := msg
	if len(args) > 0 {
		formattedMsg = fmt.Sprintf(msg, args...)
	}
	
	logLine := fmt.Sprintf("%s [%s%s%s] %s",
		timestamp,
		levelColor,
		level,
		resetColor,
		formattedMsg,
	)
	
	l.output.Println(logLine)
}

func (l *Logger) shouldLog(level string) bool {
	levels := map[string]int{
		"DEBUG": 0,
		"INFO":  1,
		"WARN":  2,
		"ERROR": 3,
		"FATAL": 4,
	}
	
	currentLevel, exists := levels[l.level]
	if !exists {
		currentLevel = levels["INFO"]
	}
	
	messageLevel, exists := levels[level]
	if !exists {
		return false
	}
	
	return messageLevel >= currentLevel
}

func (l *Logger) getLevelColor(level string) string {
	colors := map[string]string{
		"DEBUG": "\033[36m",
		"INFO":  "\033[32m",
		"WARN":  "\033[33m",
		"ERROR": "\033[31m",
		"FATAL": "\033[35m",
	}
	
	if color, exists := colors[level]; exists {
		return color
	}
	
	return ""
}

func (l *Logger) WithField(key string, value interface{}) *Logger {
	return l
}

func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	return l
}
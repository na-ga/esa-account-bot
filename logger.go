package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

type Logger struct {
	in *log.Logger
}

type Entry struct {
	Time     string `json:"time"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

var logger = &Logger{
	in: log.New(os.Stdout, "", 0),
}

func (l *Logger) Error(msg string) {
	l.Write("ERROR", msg)
}

func (l *Logger) Errorf(format string, a ...interface{}) {
	l.Write("ERROR", fmt.Sprintf(format, a...))
}

func (l *Logger) Warning(msg string) {
	l.Write("WARNING", msg)
}

func (l *Logger) Warningf(format string, a ...interface{}) {
	l.Write("WARNING", fmt.Sprintf(format, a...))
}

func (l *Logger) Info(msg string) {
	l.Write("INFO", msg)
}

func (l *Logger) Infof(format string, a ...interface{}) {
	l.Write("INFO", fmt.Sprintf(format, a...))
}

func (l *Logger) Debug(msg string) {
	l.Write("DEBUG", msg)
}

func (l *Logger) Debugf(format string, a ...interface{}) {
	l.Write("DEBUG", fmt.Sprintf(format, a...))
}

func (l *Logger) Write(severity, msg string) {
	now := time.Now().Format(time.RFC3339Nano)
	entry := Entry{
		Time:     now,
		Severity: severity,
		Message:  msg,
	}
	b, _ := json.Marshal(entry)
	l.in.Print(string(b))
}

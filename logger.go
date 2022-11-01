package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

var logger = &Logger{}

type Logger struct {
	File      *os.File
	LevelInfo map[LogLevel]LogLevelInfo
}

type LogLevelInfo struct {
	Color  *color.Color
	Prefix string
}

type LogLevel int

const (
	LL_DEBUG LogLevel = iota
	LL_WARN
	LL_ERROR
	LL_PANIC
	LL_SUCCESS
)

func InitLogger() {
	f, err := os.Create("log.txt")
	PanicIfErr(err)
	logger.File = f

	logger.LevelInfo = map[LogLevel]LogLevelInfo{
		LL_DEBUG: {
			Prefix: "DEBUG",
			Color:  color.New(color.FgCyan),
		},
		LL_WARN: {
			Prefix: "WARNING",
			Color:  color.New(color.FgYellow),
		},
		LL_ERROR: {
			Prefix: "ERROR",
			Color:  color.New(color.FgRed),
		},
		LL_PANIC: {
			Prefix: "PANIC",
			Color:  color.New(color.FgRed),
		},
		LL_SUCCESS: {
			Prefix: "SUCCESS",
			Color:  color.New(color.FgGreen),
		},
	}
}

func (l Logger) Logf(level LogLevel, msg string, args ...any) {
	levelInfo := l.LevelInfo[level]

	msg = fmt.Sprintf(msg, args...)

	date := time.Now().Format(`02 Jan 2006 15:04:05`)
	prefix := fmt.Sprintf("[%v] [%v] ", levelInfo.Prefix, date)

	levelInfo.Color.Println(prefix + msg)
	l.File.WriteString(prefix + msg + "\n")
	if level != LL_DEBUG && level != LL_SUCCESS && DEBUG_DISC_WEBHOOK != "" {
		content := prefix + "\n```\n" + msg + "```"
		if DEBUG_DISC_MENTION != "" {
			content = DEBUG_DISC_MENTION + ", " + content
		}

		content = strings.ReplaceAll(content, `"`, `\"`)

		buf := []byte(`{"content":"` + content + `"}`)
		
		http.Post(DEBUG_DISC_WEBHOOK, "application/json", bytes.NewReader(buf))
	}
	if level == LL_PANIC {
		os.Exit(1)
	}
}

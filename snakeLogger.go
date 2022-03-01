package snakeLoggerFile

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

var writeChan chan LogData

// SnakeLoggerLevel defines the levels
type SnakeLoggerLevel uint8

const (
	// DebugLevel tracks minor steps in code
	DebugLevel SnakeLoggerLevel = iota
	// InfoLevel tracks major steps in code
	InfoLevel
	// WarnLevel for things that should not be happening, but we can recover
	WarnLevel
	// ErrorLevel are major issues that are breaking functionality
	ErrorLevel
	// ReportLevel for logs that go to splunk for reports
	ReportLevel
	// NullLevel is for null loggers for testing
	NullLevel
)

var levelMap = map[SnakeLoggerLevel]string{
	DebugLevel:  "debug",
	InfoLevel:   "info",
	WarnLevel:   "warn",
	ErrorLevel:  "error",
	NullLevel:   "null",
	ReportLevel: "report",
}

// LogData is the format for a log
type LogData struct {
	ID            string
	Sev           string
	Msg           string
	Timestamp     string
	UnixTimeStamp int64
	Turn          int
	Function      string
	SnakeName     string
}

// SnakeLogger is a custom logger for tracking battlesnakes
type SnakeLogger struct {
	level       SnakeLoggerLevel
	isNull      bool
	id          string
	currentFunc string
	currentTurn int
	name        string
}

func (s *SnakeLogger) updateLogLevel(l SnakeLoggerLevel) {
	s.level = l
}

func (s *SnakeLogger) UpdateID(newid string) {
	s.id = newid
}

func (s *SnakeLogger) UpdateFunc(f string) {
	s.currentFunc = f
}

func (s *SnakeLogger) ResetFunc() {
	s.currentFunc = ""
}

func (s *SnakeLogger) UpdateTurn(t int) {
	s.currentTurn = t
}

// parseLog builds a struct for the log
//   then puts that struct on a channel for the file writer
func (s *SnakeLogger) parseLog(level SnakeLoggerLevel, msg string, t time.Time) {
	var thisLog LogData

	if s.level > level {
		return
	}

	timestamp := t.Format("2006-01-02T15:04:05.000000000")
	unixstamp := t.UnixNano()

	thisLog = LogData{
		Msg:           msg,
		Timestamp:     timestamp,
		UnixTimeStamp: unixstamp,
		ID:            s.id,
		Sev:           levelMap[level],
		Turn:          s.currentTurn,
		Function:      s.currentFunc,
		SnakeName:     s.name,
	}

	// add in ability to write to generic log from anywhere
	// start the message with GENERIC
	if strings.HasPrefix(msg, "GENERIC") {
		thisLog.Msg = msg[8:]
		thisLog.ID = ""
	}
	writeChan <- thisLog

}

func (s *SnakeLogger) Debugf(format string, v ...interface{}) {
	now := time.Now()
	msg := fmt.Sprintf(format, v...)
	go s.parseLog(DebugLevel, msg, now)

}

func (s *SnakeLogger) Infof(format string, v ...interface{}) {
	now := time.Now()
	msg := fmt.Sprintf(format, v...)
	go s.parseLog(InfoLevel, msg, now)
}

func (s *SnakeLogger) Warnf(format string, v ...interface{}) {
	now := time.Now()
	msg := fmt.Sprintf(format, v...)
	go s.parseLog(WarnLevel, msg, now)
}

func (s *SnakeLogger) Errorf(format string, v ...interface{}) {
	now := time.Now()
	msg := fmt.Sprintf(format, v...)
	go s.parseLog(ErrorLevel, msg, now)
}

func (s *SnakeLogger) Reportf(format string, v ...interface{}) {
	now := time.Now()
	msg := fmt.Sprintf(format, v...)
	go s.parseLog(ReportLevel, msg, now)
}

func (s *SnakeLogger) Debug(m string) {
	now := time.Now()
	go s.parseLog(DebugLevel, m, now)
}

func (s *SnakeLogger) Info(m string) {
	now := time.Now()
	go s.parseLog(InfoLevel, m, now)
}

func (s *SnakeLogger) Warn(m string) {
	now := time.Now()
	go s.parseLog(WarnLevel, m, now)
}

func (s *SnakeLogger) Error(m string) {
	now := time.Now()
	go s.parseLog(ErrorLevel, m, now)
}

func (s *SnakeLogger) Report(m string) {
	now := time.Now()
	go s.parseLog(ReportLevel, m, now)
}

//NewLogger returns a new copy of the local logger
func NewLogger(level string, index uint64) *SnakeLogger {

	s := SnakeLogger{
		level: InfoLevel,
		id:    "",
	}
	for l, v := range levelMap {
		if v == level {
			s.level = l
			break
		}
	}
	return &s
}

func (s *SnakeLogger) UpdateName(n string) {
	s.name = n
}

// writeChan listens on a channel for log data and writes it to a file
// this is the only place that should listen to a channel and writes to files, so it should
// be thread safe
// filename is based on supplied snake name (on the logger)
// this way each snake has its own file
// this is different than how it was working before (one file per game)
// since this will be read by splunk, we don't need new files
func writeToFile(c chan LogData) {
	// make sure path is setup
	// find home directory, since I am running this on similar linux systems, this should be all we need
	var (
		dir      string
		basedir  string
		filename string
		err      error
	)
	dir = os.Getenv("HOME")
	if dir == "" {
		fmt.Println("cannot get home dir, sending to tmp")
		dir = "/tmp"
	}
	basedir = dir + "/battlesnakeLogs"
	err = os.Mkdir(basedir, 0755)
	if err != nil {
		if !errors.Is(err, os.ErrExist) {
			fmt.Println("I don't understand this error: ", err)
		}
	}

	for m := range c {
		if m.SnakeName == "" {
			filename = basedir + "generic.log"
		} else {
			filename = fmt.Sprintf("%s/%s.log", basedir, m.SnakeName)
		}
		f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}

		if _, err := f.Write(m.Bytes()); err != nil {
			fmt.Println(err)
			break
		}
		cerr := f.Close()
		if cerr != nil {
			fmt.Println(cerr)
		}

	}
}

// String returns a nice clean string for the log
func (l LogData) String() string {
	res := fmt.Sprintf("%s %s (%v) <%s> [%s] %s \n", l.Timestamp, l.ID, l.Turn, l.Function, l.Sev, l.Msg)
	return res
}

// Bytes returns a string representation, but in bytes
func (l LogData) Bytes() []byte {
	return []byte(l.String())
}

func init() {
	writeChan = make(chan LogData)
	go writeToFile(writeChan)

}

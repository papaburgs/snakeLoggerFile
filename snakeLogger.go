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

type SnakeLoggerLevel uint8

const (
	DebugLevel SnakeLoggerLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	NullLevel
)

var levelMap = map[SnakeLoggerLevel]string{
	DebugLevel: "debug",
	InfoLevel:  "info",
	WarnLevel:  "warn",
	ErrorLevel: "error",
	NullLevel:  "null",
}

// LogData is the format for a log
type LogData struct {
	ID   string
	Sev  string
	Msg  string
	Time int64
}

// Logger is a type that can be shared in a package
type SnakeLogger struct {
	level  SnakeLoggerLevel
	isNull bool
	id     string
}

func (s *SnakeLogger) updateLogLevel(l SnakeLoggerLevel) {
	s.level = l
}

func (s *SnakeLogger) UpdateID(newid string) {
	s.id = newid
}

// parseLog builds a struct for the log
//   then puts that struct on a channel for the file writer
func (s *SnakeLogger) parseLog(level SnakeLoggerLevel, msg string, stamp int64) {
	var thisLog LogData

	if s.level > level {
		return
	}

	thisLog = LogData{
		Msg:  msg,
		Time: stamp,
		ID:   s.id,
		Sev:  levelMap[level],
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
	now := time.Now().UnixNano()
	msg := fmt.Sprintf(format, v...)
	go s.parseLog(DebugLevel, msg, now)

}

func (s *SnakeLogger) Infof(format string, v ...interface{}) {
	now := time.Now().UnixNano()
	msg := fmt.Sprintf(format, v...)
	go s.parseLog(InfoLevel, msg, now)
}

func (s *SnakeLogger) Errorf(format string, v ...interface{}) {
	now := time.Now().UnixNano()
	msg := fmt.Sprintf(format, v...)
	go s.parseLog(ErrorLevel, msg, now)
}

func (s *SnakeLogger) Debug(m string) {
	now := time.Now().UnixNano()
	go s.parseLog(DebugLevel, m, now)
}

func (s *SnakeLogger) Info(m string) {
	now := time.Now().UnixNano()
	go s.parseLog(InfoLevel, m, now)
}

func (s *SnakeLogger) Error(m string) {
	now := time.Now().UnixNano()
	go s.parseLog(ErrorLevel, m, now)
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

// writeChan listens on a channel for log data and writes it to a file
// this is the only place that should listen to a channel and writes to files, so it should
// be thread safe
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
		if m.ID == "" {
			filename = basedir + "/generic.log"
		} else {
			filename = basedir + "/game-" + m.ID + ".log"
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
	//res := l.Time + " [" + l.Sev + "] " + l.Msg + "\n"
	res := fmt.Sprintf("%v [%s] %s \n", l.Time, l.Sev, l.Msg)
	return res
}

// StringWithID returns a nice clean string with ID
func (l LogData) StringWithID() string {
	res := fmt.Sprintf("%v [%s] %s %s \n", l.Time, l.Sev, l.ID, l.Msg)
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

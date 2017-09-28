package log

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

// Logger formatter
const (
	// while flags Ldate | Ltime | Lmicroseconds | Llongfile produce,
	//	2009/01/23 01:23:23.123123 /a/b/c/d.go:23: message
	Ldate         = 1 << iota     // the date in the local time zone: 2009/01/23
	Ltime                         // the time in the local time zone: 01:23:23
	Lmicroseconds                 // microsecond resolution: 01:23:23.123123.  assumes Ltime.
	Llongfile                     // full file name and line number: /a/b/c/d.go:23
	Lshortfile                    // final file name element and line number: d.go:23. overrides Llongfile
	LUTC                          // if Ldate or Ltime is set, use UTC rather than the local time zone
	LstdFlags     = Ldate | Ltime // initial values for the standard logger
)

// Logger Levels
const (
	DEBUG int = iota
	INFO
	WARN
	ERROR
	FATAL
)

var leverStrings = [...]string{"[DEBUG]", "[INFO]", "[WARN]", "[ERROR]", "[FATL]"}

// Logger struct
type Logger struct {
	level int
	mu    sync.Mutex
	flag  int
	out   io.Writer
	buf   []byte
}

var std = Logger{out: os.Stderr, level: DEBUG, flag: LstdFlags}

// Set Logger
func Set(level int, out io.Writer, flag int) {
	if out == nil || level < DEBUG || level > FATAL {
		panic("error logger arguments")
	}
	std = Logger{out: out, level: level, flag: flag}
}

func itoa(buf *[]byte, i int, wid int) {
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

func (l *Logger) formatHeader(buf *[]byte, t time.Time, level string, file string, line int) {
	if l.flag&LUTC != 0 {
		t = t.UTC()
	}

	if l.flag&(Ldate|Ltime|Lmicroseconds) != 0 {
		if l.flag&Ldate != 0 {
			year, month, day := t.Date()
			itoa(buf, year, 4)
			*buf = append(*buf, '_')
			itoa(buf, int(month), 2)
			*buf = append(*buf, '_')
			itoa(buf, day, 2)
			*buf = append(*buf, ' ')
		}

		if l.flag&(Ltime|Lmicroseconds) != 0 {
			hour, min, sec := t.Clock()
			itoa(buf, hour, 2)
			*buf = append(*buf, ':')
			itoa(buf, min, 2)
			*buf = append(*buf, ':')
			itoa(buf, sec, 2)
			if l.flag&Lmicroseconds != 0 {
				*buf = append(*buf, '.')
				itoa(buf, t.Nanosecond()/1e3, 6)
			}
			*buf = append(*buf, ' ')
		}
	}

	*buf = append(*buf, level...)
	*buf = append(*buf, ' ')

	if l.flag&(Lshortfile|Llongfile) != 0 {
		if l.flag&Lshortfile != 0 {
			short := file
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' {
					short = file[i+1:]
					break
				}
			}
			file = short
		}
		*buf = append(*buf, file...)
		*buf = append(*buf, ':')
		itoa(buf, line, -1)
		*buf = append(*buf, ": "...)
	}
}

// Output outputs the string of lv level to the writer.
func (l *Logger) Output(lv int, s string) error {
	now := time.Now()
	var file string
	var line int
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.flag&(Lshortfile|Llongfile) != 0 {
		// release lock while getting caller info
		l.mu.Unlock()
		var ok bool
		_, file, line, ok = runtime.Caller(2)
		if !ok {
			file = "???"
			line = 0
		}
		l.mu.Lock()
	}

	l.buf = l.buf[:0]
	l.formatHeader(&l.buf, now, leverStrings[lv], file, line)
	l.buf = append(l.buf, s...)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		l.buf = append(l.buf, '\n')
	}
	_, err := l.out.Write(l.buf)
	return err
}

// SetOutput sets the output destination for the standard logger.
func SetOutput(w io.Writer) {
	if w == nil {
		panic("output can not be nil")
	}
	std.mu.Lock()
	defer std.mu.Unlock()
	std.out = w
}

// SetLevel sets the log level for the standard logger.
func SetLevel(level int) {
	if level < DEBUG || level > FATAL {
		panic("wrong log level")
	}
	std.mu.Lock()
	defer std.mu.Unlock()
	std.level = level
}

// SetFlags sets the output flags for the standard logger.
func SetFlags(flag int) {
	std.flag = flag
}

// Debug output the debug info if currrent level is not less than DEBUG.
func Debug(format string, a ...interface{}) {
	if DEBUG < std.level {
		return
	}
	std.Output(DEBUG, fmt.Sprintf(format, a...))
}

// Info output the debug info if currrent level is not less than INFO.
func Info(format string, a ...interface{}) {
	if INFO < std.level {
		return
	}
	std.Output(INFO, fmt.Sprintf(format, a...))
}

// Warn output the debug info if currrent level is not less than WARN.
func Warn(format string, a ...interface{}) {
	if WARN < std.level {
		return
	}
	std.Output(WARN, fmt.Sprintf(format, a...))
}

// Error output the debug info if currrent level is not less than ERROR.
func Error(format string, a ...interface{}) {
	if ERROR < std.level {
		return
	}
	std.Output(ERROR, fmt.Sprintf(format, a...))
}

// Fatal output the debug info if currrent level is not less than Fatal.
func Fatal(format string, a ...interface{}) {
	if FATAL < std.level {
		return
	}
	std.Output(FATAL, fmt.Sprintf(format, a...))
}

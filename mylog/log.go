package mylog

import (
	"fmt"
	"log"
	"os"
)

var logger = log.New(os.Stderr, "", log.Lshortfile|log.LstdFlags)

func Der(v error) {
	if v != nil {
		logger.Fatalf("error:%v", v)
	}
}
func Dd(v ...interface{}) {
	ft := ""
	for _ = range v {
		ft += "%v,"
	}
	logger.Output(2, fmt.Sprintf(ft, v...))

}
func Logf(f string, v ...interface{}) {

	logger.Output(2, fmt.Sprintf(f, v...))

}

type logHelper struct {
	prefix string
}

func (l *logHelper) Write(p []byte) (n int, err error) {

	logger.Printf("%s%s\n", l.prefix, p)
	return len(p), nil
	// }
	// return len(p), nil
}

func newLogHelper(prefix string) *logHelper {
	return &logHelper{prefix}
}

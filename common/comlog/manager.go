package comlog

import (
	"cloud.google.com/go/logging"
)

type OurLog struct {
	isProduction bool
	//TODO: Implement google cloud log
	localLog *Logger
}

func GetLog() *OurLog {
	return &OurLog{
		isProduction: true,
		localLog:     &Logger{},
	}
}

func (ol *OurLog) Log(log logging.Entry) {
	if log.HTTPRequest != nil {
		ol.localLog.Error("%v\n %v", log, log.HTTPRequest.Request)
	} else {
		ol.localLog.Error("%v", log)
	}
}

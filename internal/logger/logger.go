package logger

import (
	"log"
	"os"
)

type Logger struct {
	infoLog  *log.Logger
	warnLog  *log.Logger
	errorLog *log.Logger
	fatalLog *log.Logger
}

func (logger *Logger) Init() {

	logger.infoLog = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	logger.warnLog = log.New(os.Stdout, "WARN\t", log.Ldate|log.Ltime)
	logger.errorLog = log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
	logger.fatalLog = log.New(os.Stderr, "FATAL\t", log.Ldate|log.Ltime|log.Lshortfile)
	logger.infoLog.Print("----Logger initialized----")

}

func (logger *Logger) LogInfo(msg string) {
	logger.infoLog.Print(msg)
}

func (logger *Logger) LogWarn(msg string) {
	logger.warnLog.Print(msg)
}

func (logger *Logger) LogErrorMessage(msg string) {
	logger.errorLog.Print(msg)
}

func (logger *Logger) LogError(err error) {
	logger.errorLog.Print(err)
}

func (logger *Logger) LogFatal(err error) {
	logger.fatalLog.Print(err)
	os.Exit(1)
}

func (logger *Logger) LogFatalMessage(msg string) {
	logger.fatalLog.Print(msg)
	os.Exit(1)
}

package logger

import (
	"log"
	"os"
	"sync"
)

type Logger struct {
	infoLog     *log.Logger
	warnLog     *log.Logger
	errorLog    *log.Logger
	fatalLog    *log.Logger
	initialized bool
	mx          sync.Mutex
}

func (logger *Logger) Init() {
	logger.mx.Lock()
	defer logger.mx.Unlock()
	if logger.initialized == false {
		logger.infoLog = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
		logger.warnLog = log.New(os.Stdout, "WARN\t", log.Ldate|log.Ltime)
		logger.errorLog = log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
		logger.fatalLog = log.New(os.Stderr, "FATAL\t", log.Ldate|log.Ltime|log.Lshortfile)
		logger.infoLog.Print("----Logger initialized----")
		logger.initialized = true
	}
}

func (logger *Logger) checkInit() {
	if logger.initialized == false {
		logger.Init()
	}
}
func (logger *Logger) LogInfo(msg string) {
	logger.checkInit()
	logger.infoLog.Print(msg)
}

func (logger *Logger) LogWarn(msg string) {
	logger.checkInit()
	logger.warnLog.Print(msg)
}

func (logger *Logger) LogErrorMessage(msg string) {
	logger.checkInit()
	logger.errorLog.Print(msg)
}

func (logger *Logger) LogError(err error) {
	logger.checkInit()
	logger.errorLog.Print(err)
}

func (logger *Logger) LogFatal(err error) {
	logger.checkInit()
	logger.fatalLog.Print(err)
	os.Exit(1)
}

func (logger *Logger) LogFatalMessage(msg string) {
	logger.checkInit()
	logger.fatalLog.Print(msg)
	os.Exit(1)
}

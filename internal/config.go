package internal

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	PoolSize      int
	AuthKey       string
	LogLevel      string
	RefreshPeriod time.Duration
}

var Cfg *Config

func LoadConfig() {
	godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	poolSizeStr := os.Getenv("POOL_SIZE")
	poolSize := 20
	if poolSizeStr != "" {
		if ps, err := strconv.Atoi(poolSizeStr); err == nil && ps >= 0 {
			poolSize = ps
		}
	}

	authKey := os.Getenv("AUTH_KEY")

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	refreshHoursStr := os.Getenv("REFRESH_HOURS")
	refreshHours := 10
	if refreshHoursStr != "" {
		if rh, err := strconv.Atoi(refreshHoursStr); err == nil && rh > 0 {
			refreshHours = rh
		}
	}

	Cfg = &Config{
		Port:          port,
		PoolSize:      poolSize,
		AuthKey:       authKey,
		LogLevel:      logLevel,
		RefreshPeriod: time.Duration(refreshHours) * time.Hour,
	}
}

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var (
	logger   *log.Logger
	logLevel LogLevel
)

func InitLogger() {
	logger = log.New(os.Stdout, "", 0)

	levelStr := strings.ToLower(Cfg.LogLevel)
	switch levelStr {
	case "debug":
		logLevel = DEBUG
	case "info":
		logLevel = INFO
	case "warn":
		logLevel = WARN
	case "error":
		logLevel = ERROR
	default:
		logLevel = INFO
	}
}

func formatLog(level string, format string, v ...interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, v...)
	return fmt.Sprintf("[%s] [%s] %s", timestamp, level, message)
}

func LogDebug(format string, v ...interface{}) {
	if logLevel <= DEBUG {
		logger.Println(formatLog("DEBUG", format, v...))
	}
}

func LogInfo(format string, v ...interface{}) {
	if logLevel <= INFO {
		logger.Println(formatLog("INFO", format, v...))
	}
}

func LogWarn(format string, v ...interface{}) {
	if logLevel <= WARN {
		logger.Println(formatLog("WARN", format, v...))
	}
}

func LogError(format string, v ...interface{}) {
	if logLevel <= ERROR {
		logger.Println(formatLog("ERROR", format, v...))
	}
}

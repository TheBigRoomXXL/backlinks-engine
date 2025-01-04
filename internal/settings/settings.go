package settings

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

type Settings struct {
	DB_USER          string
	DB_PASSWORD      string
	DB_HOSTNAME      string
	DB_PORT          string
	DB_NAME          string
	DB_OPTIONS       string
	HTTP_TIMEOUT     time.Duration // in seconds
	HTTP_RATE_LIMIT  rate.Limit    // per domaine rate limit in req/s
	HTTP_MAX_RETRY   int
	LOG_PATH         string
	TELEMETRY_LISTEN string
}

var (
	settings  *Settings
	initError error
	initOnce  sync.Once
)

func New() (*Settings, error) {
	initOnce.Do(initSettings)
	return settings, initError
}

func initSettings() {
	err := godotenv.Load(".env")
	if err != nil && err.Error() != "open .env: no such file or directory" {
		initError = fmt.Errorf("failed to load .env file: %w", err)
	}

	dbUser, ok := os.LookupEnv("DB_USER")
	if !ok {
		initError = errors.New("environnment $DB_USER must be set")
	}
	dbPassword, ok := os.LookupEnv("DB_PASSWORD")
	if !ok {
		initError = errors.New("environnment $DB_PASSWORD must be set")
	}
	dbHostname, ok := os.LookupEnv("DB_HOSTNAME")
	if !ok {
		dbHostname = "localhost"
	}

	dbPort, ok := os.LookupEnv("DB_PORT")
	if !ok {
		dbPort = "9000"
	}

	dbName, ok := os.LookupEnv("DB_NAME")
	if !ok {
		dbName = "backlinks"
	}

	dbOptions, ok := os.LookupEnv("DB_OPTIONS")
	if !ok {
		dbName = ""
	}

	var httpTimeout time.Duration
	httpTimeoutStr, ok := os.LookupEnv("HTTP_TIMEOUT")
	if !ok {
		httpTimeout = 180 * time.Second // Google Timeout
	} else {
		i, err := strconv.Atoi(httpTimeoutStr)
		if err != nil {
			initError = fmt.Errorf("failed to parse HTTP_TIMEOUT as an int : %w", err)
			return
		}
		httpTimeout = time.Duration(i * int(time.Second))
	}

	var httpRateLimit rate.Limit
	httpRateLimitStr, ok := os.LookupEnv("HTTP_RATE_LIMIT")
	if !ok {
		httpRateLimit = rate.Limit(rate.Every(5 * time.Second))
	} else {
		i, err := strconv.Atoi(httpRateLimitStr)
		if err != nil {
			initError = fmt.Errorf("failed to parse HTTP_TIMEOUT as an int : %w", err)
			return
		}
		if i == 0 {
			slog.Warn("HTTP rate limiting is disable. Please only do so in local tests.")
		}
		httpRateLimit = rate.Limit(rate.Every(time.Duration(i * int(time.Second))))
	}

	var httpMaxRetry int
	httpMaxRetryStr, ok := os.LookupEnv("HTTP_TIMEOUT")
	if !ok {
		httpMaxRetry = 3
	} else {
		httpMaxRetry, err = strconv.Atoi(httpMaxRetryStr)
		if err != nil {
			initError = fmt.Errorf("failed to parse HTTP_TIMEOUT as an int : %w", err)
			return
		}
	}

	logPath, ok := os.LookupEnv("LOG_PATH")
	if !ok {
		logPath = "errors.log"
	}

	telemetryListen, ok := os.LookupEnv("TELEMETRY_LISTEN")
	if !ok {
		telemetryListen = "127.0.0.1:4009"
	}

	settings = &Settings{
		DB_USER:          dbUser,
		DB_PASSWORD:      dbPassword,
		DB_HOSTNAME:      dbHostname,
		DB_PORT:          dbPort,
		DB_NAME:          dbName,
		DB_OPTIONS:       dbOptions,
		HTTP_TIMEOUT:     httpTimeout,
		HTTP_RATE_LIMIT:  httpRateLimit,
		HTTP_MAX_RETRY:   httpMaxRetry,
		LOG_PATH:         logPath,
		TELEMETRY_LISTEN: telemetryListen,
	}
}

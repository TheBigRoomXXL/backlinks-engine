package settings

import (
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

type Settings struct {
	DB_USER                string
	DB_PASSWORD            string
	DB_HOSTNAME            string
	DB_PORT                string
	DB_NAME                string
	DB_OPTIONS             string
	HTTP_TIMEOUT           time.Duration // in seconds
	HTTP_RATE_LIMIT        rate.Limit    // per domaine rate limit in req/s
	HTTP_MAX_RETRY         int
	CRAWLER_MAX_CONCURENCY int
	LOG_PATH               string
	TELEMETRY_PORT         string
}

var (
	settings *Settings
	initOnce sync.Once
	initOk   = true
)

// Initialize a new settings object from environnment variable and .env
//
// The level of strictness we want while parsing the settings depends on the context. So
// rather than returning an error when we enconter an issue, we emit a warning and return
// an "ok" status. That way, the caller can ignore it or stop itself based on the strictness
// it require.
func New() (*Settings, bool) {
	initOnce.Do(initSettings)
	return settings, initOk
}

func initSettings() {
	err := godotenv.Load(".env")
	if err != nil && err.Error() != "open .env: no such file or directory" {
		initOk = false
		slog.Warn("failed to load .env file: " + err.Error())
	}

	dbUser, ok := os.LookupEnv("DB_USER")
	if !ok {
		dbUser = "backlinks-engine"
	}
	dbPassword, ok := os.LookupEnv("DB_PASSWORD")
	if !ok {
		initOk = false
		slog.Warn("environnment $DB_PASSWORD is not set, defaulting to \"\"")
		dbPassword = ""
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
			initOk = false
			slog.Warn("failed to parse HTTP_TIMEOUT as an int (defaulting to 180s) : " + err.Error())
			i = 180
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
			initOk = false
			slog.Warn("failed to parse HTTP_RATE_LIMIT as an int (defaulting to 5): " + err.Error())
			i = 5
		}
		if i == 0 {
			initOk = false
			slog.Warn("HTTP rate limiting is disable. Please only do so in local tests.")
		}
		httpRateLimit = rate.Limit(rate.Every(time.Duration(i * int(time.Second))))
	}

	var httpMaxRetry int
	httpMaxRetryStr, ok := os.LookupEnv("HTTP_MAX_RETRY")
	if !ok {
		httpMaxRetry = 3
	} else {
		httpMaxRetry, err = strconv.Atoi(httpMaxRetryStr)
		if err != nil {
			initOk = false
			slog.Warn("failed to parse HTTP_MAX_RETRY as an int (defaulting to 3): " + err.Error())
			httpMaxRetry = 3
		}
	}

	var crawlerMaxConcurency int
	crawlerMaxConcurencyStr, ok := os.LookupEnv("CRAWLER_MAX_CONCURENCY")
	if !ok {
		crawlerMaxConcurency = 1024
	} else {
		crawlerMaxConcurency, err = strconv.Atoi(crawlerMaxConcurencyStr)
		if err != nil {
			initOk = false
			slog.Warn("failed to parse CRAWLER_MAX_CONCURENCY as an int (defaulting to 1024): " + err.Error())
			crawlerMaxConcurency = 1024
		}
	}

	logPath, ok := os.LookupEnv("LOG_PATH")
	if !ok {
		logPath = "errors.log"
	}

	telemetryPort, ok := os.LookupEnv("TELEMETRY_PORT")
	if !ok {
		telemetryPort = "4009"
	}

	settings = &Settings{
		DB_USER:                dbUser,
		DB_PASSWORD:            dbPassword,
		DB_HOSTNAME:            dbHostname,
		DB_PORT:                dbPort,
		DB_NAME:                dbName,
		DB_OPTIONS:             dbOptions,
		HTTP_TIMEOUT:           httpTimeout,
		HTTP_RATE_LIMIT:        httpRateLimit,
		HTTP_MAX_RETRY:         httpMaxRetry,
		CRAWLER_MAX_CONCURENCY: crawlerMaxConcurency,
		LOG_PATH:               logPath,
		TELEMETRY_PORT:         telemetryPort,
	}
}

package sys

import (
	"database/sql"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"time"
)

// Configs contains all the configs gathered from env vars
var Configs struct {
	Http struct {
		Port            string
		ShutdownTimeout time.Duration
		ReadTimeout     time.Duration
		WriteTimeout    time.Duration
		IdleTimeout     time.Duration
	}
	Swagger struct {
		Protocol string
		Host     string
	}
	Database struct {
		ConnectionURL    string
		PingTimeout      time.Duration
		OperationTimeout time.Duration
	}
	Cache struct {
		ConnectionURL    string
		User             string
		Pass             string
		PingTimeout      time.Duration
		OperationTimeout time.Duration
		CacheTTL         time.Duration
	}
	Messaging struct {
		TopicName       string
		MaxWorkers      int
		WaitTime        time.Duration
		ShutdownTimeout time.Duration
	}
	NewRelic struct {
		AppName           string
		Licence           string
		Enabled           bool
		ConnectionTimeout time.Duration
		ShutdownTimeout   time.Duration
	}
}

// R holds static resources across the project
var R struct {
	Log      *zap.SugaredLogger
	Cache    *redis.Client
	Database *sql.DB
}

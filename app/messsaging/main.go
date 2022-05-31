package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/newrelic/go-agent/v3/integrations/nrgin"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/ribgsilva/note-api/app/api/handlers"
	"github.com/ribgsilva/note-api/app/messsaging/consumers/v1/notes"
	env2 "github.com/ribgsilva/note-api/platform/env"
	"github.com/ribgsilva/note-api/platform/logger"
	"github.com/ribgsilva/note-api/sys"
	"go.uber.org/automaxprocs/maxprocs"
	"go.uber.org/zap"
	"gocloud.dev/pubsub/awssnssqs"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func main() {

	log, err := logger.New("Notes-Messaging")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer func(log *zap.SugaredLogger) {
		_ = log.Sync()
	}(log)

	if err := run(log); err != nil {
		log.Errorw("startup", "ERROR", err)
		_ = log.Sync()
		os.Exit(1)
	}
}

func run(log *zap.SugaredLogger) error {

	// =======================================================================================================
	// Setup max procs
	if _, err := maxprocs.Set(); err != nil {
		return fmt.Errorf("maxprocs: %w", err)
	}
	log.Infow("startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	// =======================================================================================================
	// Setup configs
	sys.Configs.Database.ConnectionURL = env2.OrDefault(log, "DATABASE_CONNECTION_URL", "root:admin@localhost:3306/note")
	sys.Configs.Database.PingTimeout = env2.DurationDefault(log, "DATABASE_PING_TIMEOUT", "2s")
	sys.Configs.Database.OperationTimeout = env2.DurationDefault(log, "DATABASE_OPERATION_TIMEOUT", "5s")
	sys.Configs.Cache.ConnectionURL = env2.OrDefault(log, "CACHE_CONNECTION_URL", "localhost:6379")
	sys.Configs.Cache.User = env2.OrDefault(log, "CACHE_USER", "")
	sys.Configs.Cache.Pass = env2.OrDefault(log, "CACHE_PASS", "")
	sys.Configs.Cache.PingTimeout = env2.DurationDefault(log, "CACHE_PING_TIMEOUT", "2s")
	sys.Configs.Cache.OperationTimeout = env2.DurationDefault(log, "CACHE_PING_TIMEOUT", "10s")
	sys.Configs.Cache.CacheTTL = env2.DurationDefault(log, "CACHE_CACHE_TTL", "24h")
	sys.Configs.NewRelic.AppName = env2.OrDefault(log, "NEW_RELIC_APP_NAME", "person-api")
	sys.Configs.NewRelic.Licence = env2.OrDefault(log, "NEW_RELIC_LICENCE", "")
	sys.Configs.NewRelic.Enabled = env2.BoolDefault(log, "NEW_RELIC_ENABLED", "f")
	sys.Configs.NewRelic.ConnectionTimeout = env2.DurationDefault(log, "NEW_RELIC_CONNECTION_TIMEOUT", "10s")
	sys.Configs.NewRelic.ShutdownTimeout = env2.DurationDefault(log, "NEW_RELIC_SHUTDOWN_TIMEOUT", "10s")
	sys.Configs.Messaging.TopicName = env2.Must(log, "MESSAGING_TOPIC_NAME")
	sys.Configs.Messaging.MaxWorkers = env2.IntDefault(log, "MESSAGING_MAX_WORKERS", "1")
	sys.Configs.Messaging.WaitTime = env2.DurationDefault(log, "MESSAGING_WAIT_TIME", "10s")
	sys.Configs.Messaging.ShutdownTimeout = env2.DurationDefault(log, "MESSAGING_SHUTDOWN_TIMEOUT", "10s")

	// =======================================================================================================
	// Setup static resources

	// logger
	sys.R.Log = log

	// mysql
	var db *sql.DB
	if err := func() error {
		mysqlDb, err := sql.Open("mysql", sys.Configs.Database.ConnectionURL)
		if err != nil {
			return fmt.Errorf("error to connecto to database: %w", err)
		}
		dbCtx, dbCancel := context.WithTimeout(context.Background(), sys.Configs.Database.PingTimeout)
		defer dbCancel()
		if err := mysqlDb.PingContext(dbCtx); err != nil {
			return fmt.Errorf("could not connect to database: %w", err)
		}
		db = mysqlDb
		return nil
	}(); err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Errorf("could not close db conn gracefully: %s", err)
		}
	}()
	sys.R.Database = db

	// redis
	// doing in a func, so I can use defer to cancel the contexts
	var rdb *redis.Client
	if err := func() error {
		rdb = redis.NewClient(&redis.Options{
			Addr:     sys.Configs.Cache.ConnectionURL,
			Username: sys.Configs.Cache.User,
			Password: sys.Configs.Cache.Pass,
		})
		rdsCtx, rdsCancel := context.WithTimeout(context.Background(), sys.Configs.Cache.PingTimeout)
		defer rdsCancel()
		if err := rdb.Ping(rdsCtx).Err(); err != nil {
			return fmt.Errorf("could not connect to redis: %w", err)
		}
		return nil
	}(); err != nil {
		return err
	}
	defer func() {
		if err := rdb.Close(); err != nil {
			log.Errorf("could not close redis conn gracefully: %s", err)
		}
	}()

	sys.R.Cache = rdb

	// =======================================================================================================
	// NR

	nrApp, err := newrelic.NewApplication(
		newrelic.ConfigAppName(sys.Configs.NewRelic.AppName),
		newrelic.ConfigLicense(sys.Configs.NewRelic.Licence),
		newrelic.ConfigEnabled(sys.Configs.NewRelic.Enabled),
	)
	if err != nil {
		return err
	}
	if err := nrApp.WaitForConnection(sys.Configs.NewRelic.ConnectionTimeout); err != nil {
		return err
	}
	defer nrApp.Shutdown(sys.Configs.NewRelic.ShutdownTimeout)

	// =======================================================================================================
	// Messaging configuration

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return err
	}

	sqsCli := sqs.NewFromConfig(cfg)

	subscription := awssnssqs.OpenSubscriptionV2(
		context.Background(),
		sqsCli,
		sys.Configs.Messaging.TopicName,
		&awssnssqs.SubscriptionOptions{
			Raw:      true,
			WaitTime: sys.Configs.Messaging.WaitTime,
		})

	defer func() {
		stdCtx, stdCancel := context.WithTimeout(context.Background(), sys.Configs.Messaging.ShutdownTimeout)
		defer stdCancel()

		if err := subscription.Shutdown(stdCtx); err != nil {
			log.Errorf("could not stop subscription gracefully: %s", err)
		}
	}()

	// =======================================================================================================
	// Router configuration

	router := gin.New()
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/v1/healthcheck"},
	}), gin.Recovery(), nrgin.Middleware(nrApp))

	handlers.MapDefaults(router)

	// =======================================================================================================
	// App start and shutdown

	svr := &http.Server{
		Addr:    fmt.Sprintf(":%s", sys.Configs.Http.Port),
		Handler: router,
	}

	go func() {
		log.Info("started healthcheck http server")
		if err = svr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("error in server http server: %s", err)
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	withCancel, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	go func() {
		sig := <-shutdown
		log.Infow("shutdown", "status", "shutdown started", "signal", sig)
		defer log.Infow("shutdown", "status", "shutdown complete", "signal", sig)
		cancelFunc()
	}()

	if err := notes.Consume(withCancel, subscription, sys.Configs.Messaging.MaxWorkers); err != nil {
		return fmt.Errorf("listener error: %w", err)
	}

	return nil
}

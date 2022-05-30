package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/newrelic/go-agent/v3/integrations/nrgin"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/ribgsilva/note-api/app/api/docs"
	"github.com/ribgsilva/note-api/app/api/handlers"
	"github.com/ribgsilva/note-api/platform/platform/env"
	"github.com/ribgsilva/note-api/platform/platform/logger"
	"github.com/ribgsilva/note-api/sys"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
	"go.uber.org/automaxprocs/maxprocs"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
)

// @title Note API
// @version 1.0
// @description Service to store handle notes.
// @contact.name Gabriel Ribeiro Silva
func main() {
	log, err := logger.New("Notes-API")
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
	sys.Configs.Http.Port = env.OrDefault(log, "HTTP_PORT", "8080")
	sys.Configs.Http.ReadTimeout = env.DurationDefault(log, "HTTP_SHUTDOWN_TIMEOUT", "5s")
	sys.Configs.Http.IdleTimeout = env.DurationDefault(log, "HTTP_SHUTDOWN_TIMEOUT", "120s")
	sys.Configs.Http.WriteTimeout = env.DurationDefault(log, "HTTP_SHUTDOWN_TIMEOUT", "10s")
	sys.Configs.Http.ShutdownTimeout = env.DurationDefault(log, "HTTP_SHUTDOWN_TIMEOUT", "60s")
	sys.Configs.Swagger.Protocol = env.OrDefault(log, "SWAGGER_PROTOCOL", "http")
	sys.Configs.Swagger.Host = env.OrDefault(log, "SWAGGER_HOST", "localhost:"+sys.Configs.Http.Port)
	sys.Configs.Database.ConnectionURL = env.OrDefault(log, "DATABASE_CONNECTION_URL", "root:admin@localhost:3306/note")
	sys.Configs.Database.PingTimeout = env.DurationDefault(log, "DATABASE_PING_TIMEOUT", "2s")
	sys.Configs.Database.OperationTimeout = env.DurationDefault(log, "DATABASE_OPERATION_TIMEOUT", "5s")
	sys.Configs.Cache.ConnectionURL = env.OrDefault(log, "CACHE_CONNECTION_URL", "localhost:6379")
	sys.Configs.Cache.User = env.OrDefault(log, "CACHE_USER", "")
	sys.Configs.Cache.Pass = env.OrDefault(log, "CACHE_PASS", "")
	sys.Configs.Cache.PingTimeout = env.DurationDefault(log, "CACHE_PING_TIMEOUT", "2s")
	sys.Configs.Cache.OperationTimeout = env.DurationDefault(log, "CACHE_PING_TIMEOUT", "10s")
	sys.Configs.Cache.CacheTTL = env.DurationDefault(log, "CACHE_CACHE_TTL", "24h")
	sys.Configs.NewRelic.AppName = env.OrDefault(log, "NEW_RELIC_APP_NAME", "person-api")
	sys.Configs.NewRelic.Licence = env.OrDefault(log, "NEW_RELIC_LICENCE", "")
	sys.Configs.NewRelic.Enabled = env.BoolDefault(log, "NEW_RELIC_ENABLED", "f")
	sys.Configs.NewRelic.ConnectionTimeout = env.DurationDefault(log, "NEW_RELIC_CONNECTION_TIMEOUT", "10s")
	sys.Configs.NewRelic.ShutdownTimeout = env.DurationDefault(log, "NEW_RELIC_SHUTDOWN_TIMEOUT", "10s")

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
		_ = db.Close()
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
		_ = rdb.Close()
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
	// Router configuration

	router := gin.New()
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/v1/healthcheck"},
	}), gin.Recovery(), nrgin.Middleware(nrApp))

	handlers.MapDefaults(router)
	handlers.MapApi(router)

	docs.SwaggerInfo.Host = sys.Configs.Swagger.Host
	url := ginSwagger.URL(fmt.Sprintf("%s://%s/swagger/doc.json", sys.Configs.Swagger.Protocol, sys.Configs.Swagger.Host))
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))

	// =======================================================================================================
	// App start and shutdown

	svr := &http.Server{
		Addr:         fmt.Sprintf(":%s", sys.Configs.Http.Port),
		Handler:      router,
		ReadTimeout:  sys.Configs.Http.ReadTimeout,
		WriteTimeout: sys.Configs.Http.WriteTimeout,
		IdleTimeout:  sys.Configs.Http.IdleTimeout,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	serverErrors := make(chan error, 1)
	go func() {
		log.Info("started http server")
		serverErrors <- svr.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		log.Infow("shutdown", "status", "shutdown started", "signal", sig)
		defer log.Infow("shutdown", "status", "shutdown complete", "signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), sys.Configs.Http.ShutdownTimeout)
		defer cancel()

		if err := svr.Shutdown(ctx); err != nil {
			_ = svr.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}
	return nil
}

package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/ribgsilva/note-api/app/api/handlers"
	"github.com/ribgsilva/note-api/business/v1/note"
	"github.com/ribgsilva/note-api/persistence/v1/schema"
	"github.com/ribgsilva/note-api/platform/env"
	"github.com/ribgsilva/note-api/platform/logger"
	"github.com/ribgsilva/note-api/sys"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/proullon/ramsql/driver"
)

type NoteTests struct {
	app http.Handler
}

func TestNote(t *testing.T) {
	log, err := logger.New("Note-API-Tests")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// =======================================================================================================
	// Mocks

	// miniredis
	s := miniredis.RunT(t)

	// =======================================================================================================
	// Setup configs
	sys.Configs.Database.ConnectionURL = env.OrDefault(log, "DATABASE_CONNECTION_URL", "localhost:3306")
	sys.Configs.Database.PingTimeout = env.DurationDefault(log, "DATABASE_PING_TIMEOUT", "2s")
	sys.Configs.Database.OperationTimeout = env.DurationDefault(log, "DATABASE_OPERATION_TIMEOUT", "5s")
	//sys.Configs.Cache.ConnectionURL = env.OrDefault(log, "CACHE_CONNECTION_URL", "localhost:6379")
	sys.Configs.Cache.ConnectionURL = s.Addr()
	sys.Configs.Cache.User = env.OrDefault(log, "CACHE_USER", "")
	sys.Configs.Cache.Pass = env.OrDefault(log, "CACHE_PASS", "")
	sys.Configs.Cache.PingTimeout = env.DurationDefault(log, "CACHE_PING_TIMEOUT", "2s")
	sys.Configs.Cache.OperationTimeout = env.DurationDefault(log, "CACHE_PING_TIMEOUT", "10s")
	sys.Configs.Cache.CacheTTL = env.DurationDefault(log, "CACHE_CACHE_TTL", "24h")

	// =======================================================================================================
	// Setup resources

	// logger
	sys.R.Log = log

	// mysql
	var db *sql.DB
	if err := func() error {
		mysqlDb, err := sql.Open("ramsql", "NoteTest")
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
		t.Fatal(err)
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
		t.Fatal(err)
	}
	defer func() {
		_ = rdb.Close()
	}()

	sys.R.Cache = rdb

	// =======================================================================================================
	// Database setup

	if err := schema.Create(context.Background()); err != nil {
		t.Fatalf("sql.Exec: Error: %s\n", err)
	}
	defer schema.Drop(context.Background())

	batch := []string{
		`INSERT INTO notes (title, notes, updatedAt, createdAt) VALUES ('my notes', 'my notes text', ?, ?)`,
	}

	for _, b := range batch {
		n := time.Now().UTC()
		_, err = sys.R.Database.Exec(b, n, n)
		if err != nil {
			t.Fatalf("sql.Exec: Error: %s\n", err)
		}
	}

	// =======================================================================================================
	// Setup router
	engine := gin.Default()

	handlers.MapApi(engine)

	tests := NoteTests{
		engine,
	}

	// =======================================================================================================
	// Tun tests

	tests.getNote200(t)
	if !s.Exists("notes.1") {
		t.Fatalf("notes 1 not in cache")
	}
	tests.getNote200(t)
}

func (nt *NoteTests) getNote200(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/v1/notes/1", nil)
	w := httptest.NewRecorder()

	nt.app.ServeHTTP(w, r)

	var resp note.Note
	if w.Code != http.StatusOK {
		t.Fatalf("Test getNote200: Should receive a status code of 200 for the response : %v", w.Code)
	}

	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Test getNote200: Should be able to unmarshal the response : %v", err)
	}

	if resp.Id != 1 {
		t.Fatalf("Test getNote200: Should have received \"1\" as id id in the response: %v", resp)
	}
	if resp.Title != "my notes" {
		t.Fatalf("Test getNote200: Should have received \"my notes\" as title in the response: %v", resp)
	}
	if resp.Text != "my notes text" {
		t.Fatalf("Test getNote200: Should have received \"my notes text\" as text in the response: %v", resp)
	}
}

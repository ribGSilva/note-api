package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/ribgsilva/note-api/app/messaging/consumers/v1/notes"
	"github.com/ribgsilva/note-api/business/v1/note"
	"github.com/ribgsilva/note-api/platform/env"
	"github.com/ribgsilva/note-api/platform/logger"
	"github.com/ribgsilva/note-api/sys"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/mempubsub"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type NoteTests struct {
	topic *pubsub.Topic
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
		database, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			return fmt.Errorf("error to connecto to database: %w", err)
		}
		dbCtx, dbCancel := context.WithTimeout(context.Background(), sys.Configs.Database.PingTimeout)
		defer dbCancel()
		if err := database.PingContext(dbCtx); err != nil {
			return fmt.Errorf("could not connect to database: %w", err)
		}
		db = database
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

	batch := []string{
		`CREATE TABLE IF NOT EXISTS notes(
			id INTEGER PRIMARY KEY,
			title VARCHAR(100),
			notes TEXT,
			updatedAt DATETIME,
			createdAt DATETIME
		)`,
	}

	for _, b := range batch {
		_, err = sys.R.Database.Exec(b)
		if err != nil {
			t.Fatalf("sql.Exec: Error: %s\n", err)
		}
	}

	// =======================================================================================================
	// Messaging configuration

	topic := mempubsub.NewTopic()
	defer func() {
		_ = topic.Shutdown(context.Background())
	}()
	subscription := mempubsub.NewSubscription(topic, 1*time.Second)

	defer func() {
		stdCtx, stdCancel := context.WithTimeout(context.Background(), sys.Configs.Messaging.ShutdownTimeout)
		defer stdCancel()

		_ = subscription.Shutdown(stdCtx)
	}()

	withCancel, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	go func(tst *testing.T) {
		if err := notes.Consume(withCancel, subscription, 1); err != nil {
			t.Fatal("listener error: ", err)
		}
	}(t)

	// =======================================================================================================
	// Tun tests

	noteTests := NoteTests{topic: topic}

	t.Run("testCrud", noteTests.testCrud)
}

func (nt *NoteTests) testCrud(t *testing.T) {
	nt.testInsertSuccess(t)
}

func (nt *NoteTests) testInsertSuccess(t *testing.T) {
	event := note.Event{
		Type: "create",
		Data: note.NewNote{
			Title: "other",
			Text:  "other text",
		},
	}

	marshal, err := json.Marshal(event)
	if err != nil {
		t.Fatal("Test testInsertSuccess: failed to parse insert request body")
	}

	if err := nt.topic.Send(context.Background(), &pubsub.Message{
		Body: marshal,
	}); err != nil {
		t.Fatal("Test testInsertSuccess: failed to post message to topic: ", err)
	}

	time.Sleep(time.Second * 1)

	row := sys.R.Database.QueryRow("SELECT * FROM notes WHERE id = 1")
	if row.Err() != nil {
		t.Fatal("Test testInsertSuccess: failed to get inserted message: ", err)
	}

	var found note.Note
	if err := row.Scan(&found.Id, &found.Title, &found.Text, &found.UpdatedAt, &found.CreatedAt); err != nil {
		t.Fatalf("error parsing db data: %s", err)
	}

	if found.Id == 0 {
		t.Fatalf("Test testInsertSuccess: should have received \"other\" as title in the response: %v", found)
	}

	if found.Title != "other" {
		t.Fatalf("Test testInsertSuccess: should have received \"other\" as title in the response: %v", found)
	}

	if found.Text != "other text" {
		t.Fatalf("Test testInsertSuccess: should have received \"other text\" as text in the response: %v", found)
	}
}

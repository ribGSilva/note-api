package note

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/ribgsilva/note-api/sys"
)

func Find(ctx context.Context, id uint64) (Note, error) {
	logger := sys.R.Log
	cache := sys.R.Cache
	db := sys.R.Database

	key := fmt.Sprintf(noteKey, id)

	tcCtx, tcCancel := context.WithTimeout(ctx, sys.Configs.Cache.OperationTimeout)
	defer tcCancel()
	get, err := cache.Get(tcCtx, key).Result()
	if err != nil && err != redis.Nil {
		logger.Error("failure to get notes ", id, " from cache: ", err.Error())
	}
	if get != "" {
		var note Note
		if err := json.Unmarshal([]byte(get), &note); err != nil {
			logger.Error("error parsing cached response for key %s: %w", key, err)
		} else {
			return note, nil
		}
	}

	dbCtx, dbCancel := context.WithTimeout(ctx, sys.Configs.Database.OperationTimeout)
	defer dbCancel()
	stmt, err := db.PrepareContext(dbCtx, "SELECT * FROM notes WHERE id = ?")
	if err != nil {
		return Note{}, fmt.Errorf("failed to prepare find stmt: %w", err)
	}
	row := stmt.QueryRowContext(dbCtx, id)
	switch {
	case row.Err() != nil && row.Err() != sql.ErrNoRows:
		return Note{}, fmt.Errorf("failed to query find stmt: %w", err)
	case row.Err() != nil && row.Err() == sql.ErrNoRows:
		return Note{}, nil
	default:
		var note Note
		if err := row.Scan(&note.Id, &note.Title, &note.Text, &note.UpdatedAt, &note.CreatedAt); err != nil {
			return Note{}, fmt.Errorf("error parsing db data: %w", err)
		}

		if data, err := json.Marshal(note); err != nil {
			logger.Error("error parsing data to cache cached response for key %s: %w", key, err)
		} else {
			tcCtx, tcCancel := context.WithTimeout(ctx, sys.Configs.Cache.OperationTimeout)
			defer tcCancel()

			if err := cache.Set(tcCtx, key, string(data), sys.Configs.Cache.CacheTTL).Err(); err != nil {
				logger.Error("failure to set notes", id, "into cache: ", err.Error())
			}
		}

		return note, nil
	}

}

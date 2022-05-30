package note

import (
	"context"
	"fmt"
	"github.com/ribgsilva/note-api/sys"
	"time"
)

func Insert(ctx context.Context, newN NewNote) error {
	db := sys.R.Database

	n := time.Now().UTC()

	dbCtx, dbCancel := context.WithTimeout(ctx, sys.Configs.Database.OperationTimeout)
	defer dbCancel()
	stmt, err := db.PrepareContext(dbCtx, "INSERT INTO notes (title, notes, updatedAt, createdAt) VALUES (?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare insert stmt: %w", err)
	}
	_, err = stmt.ExecContext(dbCtx, newN.Title, newN.Text, n, n)
	if err != nil {
		return fmt.Errorf("failed to exec insert stmt: %w", err)
	}
	return nil
}

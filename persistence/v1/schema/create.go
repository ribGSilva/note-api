package schema

import (
	"context"
	"errors"
	"github.com/ribgsilva/note-api/sys"
)

func Create(ctx context.Context) error {
	db := sys.R.Database

	_, err := db.ExecContext(ctx, schema)
	if err != nil {
		return errors.New("create schema: " + err.Error())
	}

	return nil
}

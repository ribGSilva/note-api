package schema

import (
	"context"
	"errors"
	"github.com/ribgsilva/note-api/sys"
)

func Drop(ctx context.Context) error {
	db := sys.R.Database

	_, err := db.ExecContext(ctx, dropSchema)
	if err != nil {
		return errors.New("drop schema: " + err.Error())
	}

	return nil
}

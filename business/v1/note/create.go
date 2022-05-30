package note

import (
	"context"
	"github.com/ribgsilva/note-api/persistence/v1/note"
)

func Create(ctx context.Context, newN NewNote) error {
	return note.Insert(ctx, note.NewNote(newN))
}

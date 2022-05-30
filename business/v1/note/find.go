package note

import (
	"context"
	"github.com/ribgsilva/note-api/persistence/v1/note"
)

func Find(ctx context.Context, id uint64) (Note, error) {
	find, err := note.Find(ctx, id)
	if err != nil {
		return Note{}, err
	}
	if find.Id == 0 {
		return Note{}, nil
	}
	return Note(find), nil
}

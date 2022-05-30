package note

import "time"

const noteKey = "notes.%d"

type Note struct {
	Id        uint64
	Title     string
	Text      string
	UpdatedAt time.Time
	CreatedAt time.Time
}

type NewNote struct {
	Title string
	Text  string
}

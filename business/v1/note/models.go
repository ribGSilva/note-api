package note

import "time"

type Note struct {
	Id        uint64    `json:"id" example:"1"`
	Title     string    `json:"title" example:"my note"`
	Text      string    `json:"text" example:"my note text"`
	UpdatedAt time.Time `json:"updatedAt" example:"2006-01-02T15:04:05Z"`
	CreatedAt time.Time `json:"createdAt" example:"2006-01-02T15:04:05Z"`
}

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type NewNote struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

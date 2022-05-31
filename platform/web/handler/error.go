package handler

type Error struct {
	Field   string `json:"field,omitempty" example:"name"`
	Message string `json:"messsaging"`
}

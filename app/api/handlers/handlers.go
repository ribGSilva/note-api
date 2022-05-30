package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/ribgsilva/note-api/app/api/handlers/v1/healthcheck"
	"github.com/ribgsilva/note-api/app/api/handlers/v1/notes"
	"github.com/ribgsilva/note-api/platform/platform/web/handler"
)

func MapDefaults(r *gin.Engine) {
	r.GET("/v1/healthcheck", handler.Wrapper(healthcheck.Get))
}

func MapApi(r *gin.Engine) {
	r.GET("/v1/notes/:id", handler.Wrapper(notes.Get))
}

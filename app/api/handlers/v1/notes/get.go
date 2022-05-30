package notes

import (
	"github.com/gin-gonic/gin"
	"github.com/ribgsilva/note-api/business/v1/note"
	"github.com/ribgsilva/note-api/platform/platform/web/handler"
	"net/http"
	"strconv"
)

// Get godoc
// @Summary Find a notes
// @Description Find a notes using its id
// @Tags Note
// @Produce json
// @Param id path string true "Note id"
// @Success 200 {object} note.Note
// @Failure 400 {array} handler.Error
// @Failure 404 {object} handler.Error
// @Router /v1/notes/{id} [get]
func Get(ctx *gin.Context) handler.Result {

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		return handler.Result{
			Status: http.StatusBadRequest,
			Body:   handler.Error{Message: "invalid id"},
		}
	}

	get, err := note.Find(ctx, id)

	switch {
	case err != nil:
		return handler.Result{
			Status: http.StatusInternalServerError,
			Body:   handler.Error{Message: err.Error()},
		}
	case get.Id == 0:
		return handler.Result{
			Status: http.StatusNotFound,
			Body:   handler.Error{Message: "notes not found"},
		}
	default:
		return handler.Result{
			Status: http.StatusOK,
			Body:   get,
		}
	}
}

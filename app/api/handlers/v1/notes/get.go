package notes

import (
	"github.com/gin-gonic/gin"
	"github.com/ribgsilva/note-api/business/v1/note"
	handler2 "github.com/ribgsilva/note-api/platform/web/handler"
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
func Get(ctx *gin.Context) handler2.Result {

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		return handler2.Result{
			Status: http.StatusBadRequest,
			Body:   handler2.Error{Message: "invalid id"},
		}
	}

	get, err := note.Find(ctx, id)

	switch {
	case err != nil:
		return handler2.Result{
			Status: http.StatusInternalServerError,
			Body:   handler2.Error{Message: err.Error()},
		}
	case get.Id == 0:
		return handler2.Result{
			Status: http.StatusNotFound,
			Body:   handler2.Error{Message: "notes not found"},
		}
	default:
		return handler2.Result{
			Status: http.StatusOK,
			Body:   get,
		}
	}
}

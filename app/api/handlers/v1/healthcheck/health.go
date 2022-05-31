package healthcheck

import (
	"github.com/gin-gonic/gin"
	"github.com/ribgsilva/note-api/platform/web/handler"
	"net/http"
)

// Get godoc
// @Summary Check if ist is running
// @Description Check if ist is running
// @Tags Healthcheck
// @Success 200
// @Router /v1/healthcheck [get]
func Get(ctx *gin.Context) handler.Result {
	return handler.Result{Status: http.StatusOK}
}

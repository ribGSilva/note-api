package handler

import "github.com/gin-gonic/gin"

// Wrapper centralizes the response of the handlers
func Wrapper(h HandleFunc) func(*gin.Context) {
	return func(c *gin.Context) {
		result := h(c)
		switch {
		case result.Body != nil:
			c.JSON(result.Status, result.Body)
		default:
			c.Status(result.Status)
		}
	}
}

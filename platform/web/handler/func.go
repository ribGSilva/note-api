package handler

import "github.com/gin-gonic/gin"

// HandleFunc adds a return of the default gin handler func
type HandleFunc func(*gin.Context) Result

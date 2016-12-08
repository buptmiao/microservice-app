package apigateway

import "github.com/gin-gonic/gin"

func Register(router *gin.Engine) {
	r := router.Group("/api")
	RegisterFeed(r)
	RegisterProfile(r)
	RegisterTopic(r)
}

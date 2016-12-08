package apigateway

import (
	topic_client "github.com/buptmiao/microservice-app/client/topic"
	"github.com/buptmiao/microservice-app/proto/topic"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
	"net/http"
	"strconv"
)

func RegisterTopic(router *gin.RouterGroup) {
	r := router.Group("/topic")
	r.GET("/view", view)
}

func view(c *gin.Context) {
	topicID, err := strconv.ParseInt(c.Query("topic_id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	req := &topic.GetTopicRequest{topicID}
	resp, err := topic_client.GetClient().GetTopic(context.Background(), req)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.IndentedJSON(http.StatusOK, resp)
}

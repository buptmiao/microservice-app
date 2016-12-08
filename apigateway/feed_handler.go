package apigateway

import (
	feed_client "github.com/buptmiao/microservice-app/client/feed"
	"github.com/buptmiao/microservice-app/proto/feed"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
	"net/http"
	"strconv"
)

func RegisterFeed(router *gin.RouterGroup) {
	r := router.Group("/feed")
	r.GET("/get_feeds", GetFeeds)
	r.PUT("create_feed", CreateFeed)
}

func GetFeeds(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Query("user_id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	size, err := strconv.ParseInt(c.Query("size"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	req := &feed.GetFeedsRequest{userID, size}
	resp, err := feed_client.GetClient().GetFeeds(context.Background(), req)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.IndentedJSON(http.StatusOK, resp)
}

func CreateFeed(c *gin.Context) {
	req := &feed.FeedRecord{}
	if err := c.BindJSON(req); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	resp, err := feed_client.GetClient().CreateFeed(context.Background(), req)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, resp)
}

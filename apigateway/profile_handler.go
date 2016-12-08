package apigateway

import (
	profile_client "github.com/buptmiao/microservice-app/client/profile"
	"github.com/buptmiao/microservice-app/proto/profile"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
	"net/http"
	"strconv"
)

func RegisterProfile(router *gin.RouterGroup) {
	r := router.Group("/profile")
	r.GET("/get_profile", GetProfile)
}

func GetProfile(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Query("user_id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	req := &profile.GetProfileRequest{userID}
	resp, err := profile_client.GetClient().GetProfile(context.Background(), req)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.IndentedJSON(http.StatusOK, resp)
}

package profile

import (
	"errors"
	"github.com/buptmiao/microservice-app/proto/profile"
	"golang.org/x/net/context"
	"sync"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

var (
	mem map[int64]*UserInfo
	mu  sync.RWMutex
)

type UserInfo struct {
	UserID  int64
	Name    string
	Company string
	Title   string
}

// NewFeedService returns a naive, stateless implementation of Profile Service.
func NewProfileService() profile.ProfileServer {
	return service{}
}

type service struct{}

func (s service) GetProfile(_ context.Context, req *profile.GetProfileRequest) (*profile.GetProfileResponse, error) {
	userID := req.GetUserId()
	mu.RLock()
	defer mu.RUnlock()
	if ui, ok := mem[userID]; ok {
		resp := &profile.GetProfileResponse{}
		resp.UserId = userID
		resp.Name = ui.Name
		resp.Company = ui.Company
		resp.Title = ui.Title
		//resp.Feeds =
		return resp, nil
	}
	return nil, ErrUserNotFound
}

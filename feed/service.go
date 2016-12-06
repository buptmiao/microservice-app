package feed

import (
	"errors"
	"github.com/buptmiao/microservice-app/proto/feed"
	"golang.org/x/net/context"
	"sync"
)

// Storage
var (
	mem map[int64]map[int64]*feed.FeedRecord
	mu  sync.RWMutex
)

func init() {
	mem = make(map[int64]map[int64]*feed.FeedRecord)
}

var (
	ErrUserNotFound = errors.New("user not found")
)

// NewFeedService returns a naive, stateless implementation of Feed Service.
func NewFeedService() feed.FeedServer {
	return service{}
}

type service struct{}

func (s service) GetFeeds(_ context.Context, req *feed.GetFeedsRequest) (*feed.GetFeedsResponse, error) {
	userID := req.GetUserId()
	size := req.GetSize()
	feeds := []*feed.FeedRecord{}
	mu.RLock()
	defer mu.RUnlock()
	if v, ok := mem[userID]; !ok {
		return nil, ErrUserNotFound
	} else {
		for _, f := range v {
			if size <= 0 {
				break
			}
			feeds = append(feeds, f)
			size--
		}
	}
	return &feed.GetFeedsResponse{Feeds: feeds}, nil
}

func (s service) CreateFeed(_ context.Context, req *feed.FeedRecord) (*feed.OkResponse, error) {
	mu.Lock()
	defer mu.Unlock()
	userFeeds, ok := mem[req.UserId]
	if !ok {
		mem[req.UserId] = map[int64]*feed.FeedRecord{req.Id: req}
		return &feed.OkResponse{}, nil
	}
	userFeeds[req.Id] = req
	return &feed.OkResponse{}, nil
}

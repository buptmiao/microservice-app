package topic

import (
	"errors"
	"github.com/buptmiao/microservice-app/proto/topic"
	"golang.org/x/net/context"
	"sync"
)

var (
	ErrTopicNotFound = errors.New("topic not found")
)

var (
	mem map[int64]*Topic
	mu  sync.RWMutex
)

type Topic struct {
	TopicID int64
	Subject string
	Content string
}

// NewFeedService returns a naive, stateless implementation of Topic Service.
func NewTopicService() topic.TopicServer {
	return service{}
}

type service struct{}

func (s service) GetTopic(_ context.Context, req *topic.GetTopicRequest) (*topic.GetTopicResponse, error) {
	TopicID := req.GetTopicId()
	mu.RLock()
	defer mu.RUnlock()
	if ti, ok := mem[TopicID]; ok {
		resp := &topic.GetTopicResponse{}
		resp.TopicId = TopicID
		resp.Subject = ti.Subject
		resp.Content = ti.Content
		return resp, nil
	}
	return nil, ErrTopicNotFound
}

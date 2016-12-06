package util

import (
	"golang.org/x/net/context"
)

func DummyEncode(_ context.Context, request interface{}) (interface{}, error) {
	return request, nil
}

func DummyDecode(_ context.Context, response interface{}) (interface{}, error) {
	return response, nil
}

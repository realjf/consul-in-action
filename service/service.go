package service

import (
	"context"
	"errors"

	"github.com/realjf/consul-in-action/config"
	"github.com/realjf/consul-in-action/discover"
)

type DiscoveryService interface {
	// 健康检查接口
	HealthCheck() bool

	// 打招呼接口
	SayHello() string

	// 服务发现接口
	DiscoveryService(ctx context.Context, serviceName string) ([]interface{}, error)
}

var ErrNotDiscoveryService = errors.New("discovery service not found")

type discoveryService struct {
	discoveryClient discover.DiscoveryClient
}

func NewDiscoveryService(discoveryClient discover.DiscoveryClient) DiscoveryService {
	return &discoveryService{
		discoveryClient: discoveryClient,
	}
}

func (s *discoveryService) SayHello() string {
	return "Hello World"
}

func (s *discoveryService) DiscoveryService(ctx context.Context, serviceName string) ([]interface{}, error) {
	ins := s.discoveryClient.DiscoverServices(serviceName, config.Logger)

	if ins == nil || len(ins) == 0 {
		return nil, ErrNotDiscoveryService
	}
	return ins, nil
}

func (s *discoveryService) HealthCheck() bool {
	return true
}

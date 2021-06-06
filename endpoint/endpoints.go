package endpoint

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/realjf/consul-in-action/service"
)

type DiscoveryEndpoints struct {
	SayHelloEndpoint    endpoint.Endpoint
	DiscoveryEndpoint   endpoint.Endpoint
	HealthCheckEndpoint endpoint.Endpoint
}

//实现sayHello请求：包括请求结构体、响应结构体和创建方法
type SayHelloRequest struct {
}

type SayHelloResponse struct {
	Message string `json"message"`
}

func NewSayHelloEndpoint(svc service.DiscoveryService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		message := svc.SayHello()
		return SayHelloResponse{
			Message: message,
		}, nil
	}
}

// 实现服务发现请求：请求结构体、响应结构体和创建方法
type DiscoveryRequest struct {
	ServiceName string
}

type DiscoveryResponse struct {
	Instances []interface{} `json:"instances"`
	Error     string        `json:"error"`
}

func NewDiscoveryEndpoint(svc service.DiscoveryService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(DiscoveryRequest)
		instances, err := svc.DiscoveryService(ctx, req.ServiceName)
		var errString = ""
		if err != nil {
			errString = err.Error()
		}
		return &DiscoveryResponse{
			Instances: instances,
			Error:     errString,
		}, nil
	}
}

// 实现健康检查请求：请求结构体、响应结构体和创建方法
type HealthCheckRequest struct {
}

type HealthCheckResponse struct {
	Status bool `json:"status"`
}

func NewHealthCheckEndpoint(svc service.DiscoveryService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		status := svc.HealthCheck()
		return HealthCheckResponse{
			Status: status,
		}, nil
	}
}

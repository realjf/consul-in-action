package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/realjf/consul-in-action/config"
	"github.com/realjf/consul-in-action/discover"
	"github.com/realjf/consul-in-action/endpoint"
	"github.com/realjf/consul-in-action/service"
	"github.com/realjf/consul-in-action/transport"

	uuid "github.com/satori/go.uuid"
)

func main() {
	var (
		// 服务地址和服务名
		servicePort = flag.Int("service.port", 12000, "service port")
		serviceHost = flag.String("service.host", "127.0.0.1", "service host")
		serviceName = flag.String("service.name", "SayHello", "service name")
		// consul 地址
		consulPort    = flag.Int("consul.port", 8500, "consul port")
		consulAddress = flag.String("consul.host", "127.0.0.1", "consul host")
	)
	flag.Parse()

	ctx := context.Background()
	errChan := make(chan error)

	// 声明服务发现客户端
	var discoveryClient discover.DiscoveryClient

	discoveryClient, err := discover.NewKitDiscoverClient(*consulAddress, *consulPort)
	// 获取服务发现客户端失败，直接关闭服务
	if err != nil {
		config.Logger.Println("Get Consul Client failed")
		os.Exit(-1)
	}

	// 声明并初始化 Service
	var svc = service.NewDiscoveryService(discoveryClient)

	// 创建Endpoint
	sayHelloEndpoint := endpoint.NewSayHelloEndpoint(svc)
	discoveryEndpoint := endpoint.NewDiscoveryEndpoint(svc)
	healthCheckEndpoint := endpoint.NewHealthCheckEndpoint(svc)

	endpoints := endpoint.DiscoveryEndpoints{
		SayHelloEndpoint:    sayHelloEndpoint,
		DiscoveryEndpoint:   discoveryEndpoint,
		HealthCheckEndpoint: healthCheckEndpoint,
	}

	// 创建http.Handler
	r := transport.NewHttpHandler(ctx, endpoints, config.KitLogger)
	// 定义服务实例ID
	instanceId := *serviceName + "-" + uuid.NewV4().String()
	// 启动http server
	go func() {
		config.Logger.Println("Http Server start at port:" + strconv.Itoa(*servicePort))
		// 启动前执行注册
		if !discoveryClient.Register(*serviceName, instanceId, "/health", *serviceHost, *servicePort, nil, config.Logger) {
			config.Logger.Printf("string-service for service %s failed.", serviceName)
			// 注册失败，服务启动失败
			os.Exit(-1)
		}
		handler := r
		errChan <- http.ListenAndServe(":"+strconv.Itoa(*servicePort), handler)
	}()

	go func() {
		// 监控系统信号，等待ctrl + c 系统信号通知服务关闭
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errChan <- fmt.Errorf("%s", <-c)
	}()

	error := <-errChan
	// 服务退出取消注册
	discoveryClient.DeRegister(instanceId, config.Logger)
	config.Logger.Println(error)
}

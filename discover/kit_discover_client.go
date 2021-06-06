package discover

import (
	"log"
	"strconv"
	"sync"

	"github.com/go-kit/kit/sd/consul"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
)

type KitDiscoverClient struct {
	Addr         string // Consul Host Address
	Port         int    // Consul port
	client       consul.Client
	config       *api.Config // 连接consul的配置
	mutex        sync.Mutex
	instancesMap sync.Map // 服务实例缓存字段
}

func NewKitDiscoverClient(consulAddress string, consulPort int) (DiscoveryClient, error) {
	// 通过consul host和consul port创建一个consul.Client
	consulConfig := api.DefaultConfig()
	consulConfig.Address = consulAddress + ":" + strconv.Itoa(consulPort)
	apiClient, err := api.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}
	client := consul.NewClient(apiClient)
	return &KitDiscoverClient{
		Addr:   consulAddress,
		Port:   consulPort,
		config: consulConfig,
		client: client,
	}, err
}

func (k *KitDiscoverClient) Register(
	serviceName, instanceId, healthCheckUrl string,
	instanceAddress string,
	instancePort int,
	meta map[string]string,
	logger *log.Logger) bool {
	// 构建服务实例元数据
	serviceRegisteration := &api.AgentServiceRegistration{
		ID:      instanceId,
		Name:    serviceName,
		Address: instanceAddress,
		Port:    instancePort,
		Meta:    meta,
		Check: &api.AgentServiceCheck{
			DeregisterCriticalServiceAfter: "30s",
			HTTP:                           "http://" + instanceAddress + ":" + strconv.Itoa(instancePort) + healthCheckUrl,
			Interval:                       "15s",
		},
	}

	// 发送服务注册到consul 中
	err := k.client.Register(serviceRegisteration)
	if err != nil {
		log.Println("Register Service Error!")
		return false
	}
	log.Println("Register Service Success!")
	return true
}

func (k *KitDiscoverClient) DeRegister(instanceId string, logger *log.Logger) bool {
	// 构建包含服务实例 ID 的元数据结构体
	serviceRegisteration := &api.AgentServiceRegistration{
		ID: instanceId,
	}
	// 发送服务注销请求
	err := k.client.Deregister(serviceRegisteration)
	if err != nil {
		logger.Println("DeRegister Service Error!")
		return false
	}
	log.Println("Register Service Success!")
	return true
}

func (k *KitDiscoverClient) DiscoverServices(serviceName string, logger *log.Logger) []interface{} {
	// 该服务已监控并缓存
	instanceList, ok := k.instancesMap.Load(serviceName)
	if ok {
		return instanceList.([]interface{})
	}
	// 申请锁
	k.mutex.Lock()
	defer k.mutex.Unlock()
	// 再次检查是否监控
	instanceList, ok = k.instancesMap.Load(serviceName)
	if ok {
		return instanceList.([]interface{})
	} else {
		// 注册监控
		go func() {
			// 使用consul服务实例监控来监控某个服务名的服务实例列表变化
			params := make(map[string]interface{})
			params["type"] = "service"
			params["service"] = serviceName
			plan, _ := watch.Parse(params)
			plan.Handler = func(u uint64, i interface{}) {
				if i == nil {
					return
				}
				v, ok := i.([]*api.ServiceEntry)
				if !ok {
					return // 数据异常，忽略
				}
				// 没有服务实例在线
				if len(v) == 0 {
					k.instancesMap.Store(serviceName, []interface{}{})
				}
				var healthServices []interface{}
				for _, service := range v {
					if service.Checks.AggregatedStatus() == api.HealthPassing {
						healthServices = append(healthServices, service.Service)
					}
				}
				k.instancesMap.Store(serviceName, healthServices)
			}
			defer plan.Stop()
			plan.Run(k.config.Address)
		}()
	}

	// 根据服务名请求服务实例列表
	entries, _, err := k.client.Service(serviceName, "", false, nil)
	if err != nil {
		k.instancesMap.Store(serviceName, []interface{}{})
		logger.Println("Discover Service Error!")
		return nil
	}
	instances := make([]interface{}, len(entries))
	for i := 0; i < len(instances); i++ {
		instances[i] = entries[i].Service
	}
	k.instancesMap.Store(serviceName, instances)
	return instances
}

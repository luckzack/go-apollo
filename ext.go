package apollo

import (
	"io/ioutil"
	"net/http"
	"time"
)

type NamespaceConfig struct {
	conf      *Config
	Namespace string
}

func (config *Config) GlobalSettings() *NamespaceConfig {
	return config.GetNamespace("westudy.global.settings")
}

func (config *Config) GrpcMiddleware() *NamespaceConfig {
	return config.GetNamespace("westudy.grpc.middleware")
}

func (config *Config) GetNamespace(ns string) *NamespaceConfig {
	return &NamespaceConfig{
		conf:      config,
		Namespace: ns,
	}
}

func (nsConfig *NamespaceConfig) GetString(key string, defaultValue string) string {
	return nsConfig.conf.GetStringByNameSpace(nsConfig.Namespace, key, defaultValue)
}

// 增强版http.Get, 增加了最多3次重试（因为在istio-proxy的pod中，envoy要从控制面拉取配置而启动较晚，导致业务容器启动后请求配置中心失败）
func httpGet(url string) ([]byte, error) {
	var try = 0
REQUEST:
	try++
	rsp, err := http.Get(url)
	if err != nil {
		if try < 5 {
			logger.Infof("http get fail: %s, try again after one second, tried: %d", err.Error(), try)
			time.Sleep(time.Second)
			goto REQUEST
		}
		return nil, err
	}
	defer rsp.Body.Close()
	data, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		if try < 5 {
			logger.Infof("http get fail: %s, try again after one second, tried: %d", err.Error(), try)
			time.Sleep(time.Second)
			goto REQUEST
		}
		return nil, err
	}
	return data, nil
}

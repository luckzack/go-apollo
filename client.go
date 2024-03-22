package apollo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type conf struct {
	env, appID, cluster, namespace, server string
}

var (
	defaultConf = &conf{
		cluster:   "default",
		namespace: "application",
	}
)

func SetAppIDAndEnv(appID, envName string) {
	defaultConf.appID = appID

	switch strings.ToLower(envName) {
	case "", "local":
		defaultConf.env = ENV_DEV
	case "dev", "development", "fat":
		defaultConf.env = ENV_FAT
	case "test", "uat":
		defaultConf.env = ENV_UAT
	case "pro", "prod", "production":
		defaultConf.env = ENV_PRO
	}

}

func SetMetaServer(m map[string]string) {
	metaServer = m
}

func Start() error {
	return startWithCluster(defaultConf.appID, defaultConf.env, "default")
}

func start(appID, envName string) error {
	return startWithCluster(appID, envName, "default")
}

func startWithCluster(appID, envName, cluster string) error {

	defer func() {
		if err := recover(); err != nil {
			logger.Errorf("recover: %v", err)
		}
	}()

	defaultConf.appID = appID
	defaultConf.env = envName
	defaultConf.cluster = cluster
	if defaultConf.env != "" {
		url, ok := metaServer[defaultConf.env]
		if ok {
			defaultConf.server = url
		}
	}

	if defaultConf.appID == "" {
		return fmt.Errorf("app.id not define")
	}

	if defaultConf.env == "" {
		return fmt.Errorf("env not define")
	}

	if defaultConf.cluster == "" {
		defaultConf.cluster = "default"
	}

	logger.Infof("start config with %+v", *defaultConf)

	server := configServer{}

	no := notify{
		notifications: make(map[string]int),
	}

	no.put(defaultConf.namespace, -1)

	config := &Config{
		conf:   defaultConf,
		server: &server,
		notify: &no,
		nCache: make(map[string]*cache),
	}

	//启动第一次获取配置
	err := server.updateServers(defaultConf)
	if err != nil {
		logger.Warnf("get meta servers fail ,try to get config from local, err: %v", err)
		err = loadFromLocal(config)
		if err != nil {
			logger.Errorf("get config from local fail, err: %v", err)
			return err
		}
	}

	//默认初始化 application 命名空间的配置
	err = config.updateConfig(defaultConf.namespace)
	if err != nil {
		logger.Warnf("updateConfig failed, err: %v\n", err)
		err = loadFromLocal(config)
		if err != nil {
			logger.Errorf("loadFromLocal failed, err: %v\n", err)
			return err
		}
	}
	go config.doNotify()
	go config.doUpdateMeta()
	defaultConfig = config
	return nil
}

func loadFromLocal(config *Config) error {
	f, err := os.Open(getFileName(config.conf))
	if err != nil {
		return err
	}
	defer f.Close()
	d, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	return unmarshalData(d, config, defaultConf.namespace)
}

type cf struct {
	AppId string `json:"app.id,omitempty"`
	Env   string `json:"env,omitempty"`
}

// 从文件中读取app.id及env
// 格式如下
// {
// "app.id":"SampleApp",
// "env":"DEV"
// }
func StartWithFile(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	res := &cf{}
	err = json.NewDecoder(f).Decode(&res)
	if err != nil {
		return err
	}
	return start(res.AppId, res.Env)
}

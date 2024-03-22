package apollo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var logger = log.WithField("module", "apollo")

type Config struct {
	conf       *conf
	server     configServerOpt
	notify     *notify
	nCache     map[string]*cache
	lock       sync.RWMutex
	handlers   []Handler
	lastUpdate time.Time
}

type Handler func(notice *Notice)

type Notice struct {
	Namespace string
	OldValues map[string]string
	NewValues map[string]string
}

// 检查指定key是不是有更新
func (notice *Notice) IsChange(key string) bool {

	v, ok := notice.NewValues[key]
	if !ok {
		return false
	}
	o, ok := notice.OldValues[key]
	if !ok {
		return true
	}
	return v == o
}

// 获取变更的key 列表
func (notice *Notice) GetChangeKeys() (keys []string) {

	keys = make([]string, 0)

	for k, v := range notice.NewValues {
		if o, ok := notice.OldValues[k]; !ok {
			keys = append(keys, k)
		} else {
			if o != v {
				keys = append(keys, k)
			}
		}
	}
	return
}

var (
	defaultConfig *Config
	once          sync.Once
)

func GetConfig() (config *Config, err error) {

	if defaultConfig == nil {
		once.Do(func() {
			err = Start()
			if err != nil {
				logger.Errorf("apollo start err: %s", err.Error())
			}
		})
	}
	config = defaultConfig
	if config == nil {
		err = fmt.Errorf("apollo cfg not init")
	}
	return
}

func (config *Config) Watch(handler Handler) {
	config.handlers = append(config.handlers, handler)
}

type cache struct {
	lock sync.RWMutex
	v    map[string]string
}

type configuration struct {
	AppID         string            `json:"appID,omitempty"`
	Cluster       string            `json:"cluster,omitempty"`
	NameSpace     string            `json:"namespaceName,omitempty"`
	Configuration map[string]string `json:"configurations,omitempty"`
}

func (config *Config) updateConfig(namespace string) error {

	c := &conf{
		env:       defaultConf.env,
		appID:     defaultConf.appID,
		cluster:   defaultConf.cluster,
		server:    defaultConf.server,
		namespace: namespace,
	}

	url, err := config.server.getConfigUrl(c)
	if err != nil {
		return err
	}
	rsp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	data, err := ioutil.ReadAll(rsp.Body)

	logger.Debugf("config data: %s %v, url: %s, status: %d", data, err, url, rsp.StatusCode)

	if err != nil {
		return err
	}

	if len(data) > 0 {
		err = unmarshalData(data, config, namespace)
		if err != nil {
			return fmt.Errorf("json parse [%s] fail: %s", string(data), err.Error())
		}
	} else {
		logger.Warnf("config data empty, url: %s, status: %d", url, rsp.StatusCode)
	}

	logger.Infof("Loaded lasted config from apollo success %s %s", config.conf.appID, config.conf.env)
	config.lastUpdate = time.Now()
	return saveToFile(data, c)

}

func unmarshalData(data []byte, config *Config, namespace string) error {
	cf := configuration{}
	err := json.Unmarshal(data, &cf)
	if err != nil {
		return err
	}
	config.lock.Lock()
	defer config.lock.Unlock()

	c, ok := config.nCache[namespace]
	if ok {
		c.lock.Lock()
		defer c.lock.Unlock()
		for _, v := range config.handlers {
			f := v
			go f(&Notice{
				Namespace: namespace,
				OldValues: c.v,
				NewValues: cf.Configuration,
			})
		}
		c.v = cf.Configuration
	} else {
		config.nCache[namespace] = &cache{
			v: cf.Configuration,
		}
		for _, v := range config.handlers {
			f := v
			go f(&Notice{
				Namespace: namespace,
				OldValues: nil,
				NewValues: cf.Configuration,
			})
		}
	}
	return nil
}

func GetStringValue(key string, defaultValue string) string {
	cfg, err := GetConfig()
	if err != nil {
		return defaultValue
	}
	return cfg.GetStringValue(key, defaultValue)
}

func (config *Config) GetStringValue(key string, defaultValue string) string {
	config.lock.RLock()
	defer config.lock.RUnlock()
	cache, ok := config.nCache[config.conf.namespace]
	if !ok {
		return defaultValue
	}
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	v, ok := cache.v[key]
	if ok {
		return v
	} else {
		return defaultValue
	}
}

func (config *Config) GetString(key string) (v string, ok bool) {
	config.lock.RLock()
	defer config.lock.RUnlock()
	cache, ok := config.nCache[config.conf.namespace]
	if !ok {
		return
	}
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	v, ok = cache.v[key]
	return
}

func (config *Config) GetAllKeys() (keys []string) {
	keys = make([]string, 0)
	config.lock.RLock()
	defer config.lock.RUnlock()
	cache, ok := config.nCache[config.conf.namespace]
	if !ok {
		return
	}
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	for k := range cache.v {
		keys = append(keys, k)
	}
	return
}

func (config *Config) GetAllKeysWithPrefix(prefix string) (keys []string) {
	keys = make([]string, 0)
	config.lock.RLock()
	defer config.lock.RUnlock()
	cache, ok := config.nCache[config.conf.namespace]
	if !ok {
		return
	}
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	for k := range cache.v {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	return
}

func (config *Config) GetAllKeysByNamespace(namespace string) (keys []string) {
	config.lock.RLock()
	cache, ok := config.nCache[namespace]
	if !ok {
		config.lock.RUnlock()
		_ = config.updateConfig(namespace)
		config.lock.RLock()
		cache, ok = config.nCache[namespace]
		config.lock.RUnlock()
		if !ok {
			return nil
		}
		config.notify.put(namespace, -1)
		cache.lock.RLock()
		defer cache.lock.RUnlock()
		for k := range cache.v {
			keys = append(keys, k)
		}
		return keys
	}
	defer config.lock.RUnlock()
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	for k := range cache.v {
		keys = append(keys, k)
	}
	return keys
}

func (config *Config) GetStringByNameSpace(namespace string, key string, defaultValue string) string {
	config.lock.RLock()
	cache, ok := config.nCache[namespace]
	if !ok {
		config.lock.RUnlock()
		_ = config.updateConfig(namespace)
		config.lock.RLock()
		cache, ok = config.nCache[namespace]
		config.lock.RUnlock()
		if !ok {
			return defaultValue
		}
		config.notify.put(namespace, -1)
		cache.lock.RLock()
		defer cache.lock.RUnlock()
		v, ok := cache.v[key]
		if ok {
			return v
		} else {
			return defaultValue
		}
	}
	defer config.lock.RUnlock()
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	v, ok := cache.v[key]
	if ok {
		return v
	} else {
		return defaultValue
	}
}

func GetBool(key string, defaultValue bool) bool {
	cfg, err := GetConfig()
	if err != nil {
		return defaultValue
	}
	return cfg.GetBool(key, defaultValue)
}

func (config *Config) GetBool(key string, defaultValue bool) bool {

	v, ok := config.GetString(key)
	if ok {
		if v == "true" {
			return true
		}
		return false
	} else {
		return defaultValue
	}
}

func (config *Config) GetList(key string) (array []string, ok bool) {

	v, ok := config.GetString(key)
	if ok {
		array = strings.Split(v, ",")
		return
	} else {
		return
	}
}

func (config *Config) GetJson(key string, vv interface{}) (ok bool, err error) {
	v, ok := config.GetString(key)
	if ok {
		err = json.Unmarshal([]byte(v), vv)
		return
	} else {
		return
	}
}

func GetInt(key string, defaultValue int) int {
	cfg, err := GetConfig()
	if err != nil {
		return defaultValue
	}
	return cfg.GetInt(key, defaultValue)
}

func (config *Config) GetInt(key string, defaultValue int) int {

	v, ok := config.GetString(key)
	if !ok {
		return defaultValue
	}

	i, err := strconv.Atoi(v)
	if err != nil {
		return defaultValue
	}
	return i
}

func (config *Config) doNotify() {
	for {

		err := listen(config)
		if err != nil {

			logger.Errorf("listen err: %s", err.Error())
			//有问题休息一下，然后重试
			time.Sleep(time.Second * 30)
		}
	}
}
func (config *Config) doUpdateMeta() {

	for {
		time.Sleep(time.Second * 30)
		err := config.server.updateServers(config.conf)
		if err != nil {

			logger.Errorf("updateServers failed, err: %v, conf: %+v", err, config.conf)
		}
	}
}

func (config *Config) GetAllValue() (string, error) {

	config.lock.RLock()
	defer config.lock.RUnlock()
	cache, ok := config.nCache[config.conf.namespace]
	if !ok {
		return "", fmt.Errorf("cache namespace: %v not found", config.conf.namespace)
	}
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	return fmt.Sprintf("%v", cache.v), nil
}

func listen(config *Config) error {

	defer func() {
		if err := recover(); err != nil {
			logger.Errorf("recover: %v", err)
		}
	}()

	notifyUrl, err := config.server.getNotifyUrl(config.notify, config.conf)
	if err != nil {
		return err
	}
	rsp, err := http.Get(notifyUrl)
	if err != nil {
		logger.Errorf("http get '%s' err: %s", notifyUrl, err.Error())
		return err
	}
	//logger.Infof("listen request: %s, status: %d",
	//	notifyUrl, rsp.StatusCode)
	defer rsp.Body.Close()
	if rsp.StatusCode == http.StatusNotModified {
		// 超过12小时，往配置中心注册一下自己
		if time.Since(config.lastUpdate) > 12*time.Hour {
			config.lastUpdate = time.Now()
			// map 遍历是安全的
			for name := range config.nCache {
				config.updateConfig(name)
			}
		}
		//没有变化，说明等了很久了，直接重试
		return nil
	}
	if rsp.StatusCode != http.StatusOK {
		//有问题，返回等待重试
		logger.Errorf("http get '%s' fail: %s", notifyUrl, rsp.Status)
		return fmt.Errorf(rsp.Status)
	}
	data, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return err
	}

	logger.Infof("listen result: %s", string(data))
	var notifications []*notification

	err = json.Unmarshal(data, &notifications)
	if err != nil {
		return err
	}

	for _, v := range notifications {
		config.notify.put(v.NamespaceName, v.NotificationID)
		config.updateConfig(v.NamespaceName)
	}
	return nil
}

func saveToFile(bytes []byte, conf *conf) error {

	fileName := getFileName(conf)
	err := os.MkdirAll(getDir(conf), os.ModePerm)
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(bytes)
	return err
}
func getDir(conf *conf) string {
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("%s\\.apollo\\%s\\config-cache", getHomeDir(), conf.appID)
	default:
		return fmt.Sprintf("%s/.apollo/%s/config-cache", getHomeDir(),
			conf.appID)
	}
}
func getFileName(conf *conf) string {
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("%s\\.apollo\\%s\\config-cache\\%s+%s+%s.properties", getHomeDir(),
			conf.appID, conf.appID, conf.cluster, conf.namespace)
	default:
		return fmt.Sprintf("%s/.apollo/%s/config-cache/%s+%s+%s.properties", getHomeDir(),
			conf.appID, conf.appID, conf.cluster, conf.namespace)
	}
}

func getHomeDir() string {
	user, _ := user.Current()
	return user.HomeDir
}

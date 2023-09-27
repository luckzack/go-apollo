package apollo

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
)

const (
	ENV_DEV = "DEV"
	ENV_FAT = "FAT"
	ENV_UAT = "UAT"
	ENV_PRO = "PRO"
)

var metaServer = map[string]string{
	"LOCAL": "http://dev.apollo.ai-arena.qq.com",
	ENV_DEV: "http://dev.apollo.ai-arena.qq.com",
	ENV_FAT: "http://172.16.80.59:31760",
	ENV_UAT: "http://172.16.0.157:30682",
	ENV_PRO: "http://172.16.1.173:31639",
}

type application struct {
	XmlName  xml.Name   `xml:"application"` // root
	Name     string     `xml:"name"`        // name
	Instance []instance `xml:"instance"`    //服务实例
}

type portState struct {
	Enable bool `xml:"enabled,attr"`
	Port   int  `xml:",chardata"`
}

type dcInfo struct {
	Class string `xml:"class,attr"`
	Name  string `xml:"name"`
}

type leaseInfo struct {
	Renew     int    `xml:"renewalIntervalInSecs"`
	Duration  int    `xml:"durationInSecs"`
	Register  uint64 `xml:"registrationTimestamp"`
	LastRenew uint64 `xml:"lastRenewalTimestamp"`
	Evict     int    `xml:"evictionTimestamp"`
	ServiceUp uint64 `xml:"serviceUpTimestamp"`
}

type meta struct {
	Class string `xml:"class,attr"`
}

type instance struct {
	ID          string    `xml:"instanceId"`
	Host        string    `xml:"hostName"`
	App         string    `xml:"app"`
	Ip          string    `xml:"ipAddr"`
	Status      string    `xml:"status"`
	Ovs         string    `xml:"overriddenstatus"`
	Port        portState `xml:"port"`
	SPort       portState `xml:"securePort"`
	CId         int       `xml:"countryId"`
	DC          dcInfo    `xml:"dataCenterInfo"`
	Lease       leaseInfo `xml:"leaseInfo"`
	Meta        meta      `xml:"metadata"`
	HomePage    string    `xml:"homePageUrl"`
	StatusPage  string    `xml:"statusPageUrl"`
	HealthCheck string    `xml:"healthCheckUrl"`
	Vip         string    `xml:"vipAddress"`
	SVip        string    `xml:"secureVipAddress"`
	IsCoor      bool      `xml:"isCoordinatingDiscoveryServer"`
	LastUpdate  uint64    `xml:"lastUpdatedTimestamp"`
	LastDirty   uint64    `xml:"lastDirtyTimestamp"`
	ActionType  string    `xml:"actionType"`
}

type configServer struct {
	instances []instance
	lock      sync.RWMutex
	count     int64
}

type configServerOpt interface {
	getMetaServer(conf *conf) (string, error)
	getConfigUrl(conf *conf) (string, error)
	updateServers(conf *conf) error
	getNotifyUrl(notify *notify, conf *conf) (string, error)
}

func (c *configServer) getMetaServer(conf *conf) (string, error) {
	return fmt.Sprintf("%s/eureka/apps/APOLLO-CONFIGSERVICE", conf.server), nil
}

func (c *configServer) getConfigUrl(conf *conf) (string, error) {
	addr := conf.server
	//if conf.env != ENV_DEV {
	//	ins, err := c.getOneInstance()
	//	if err != nil {
	//		return "", err
	//	}
	//	addr = ins.HomePage
	//}

	return fmt.Sprintf("%s/configs/%s/%s/%s?ip=%s",
		addr,
		conf.appId,
		conf.cluster,
		conf.namespace,
		LocalIP()), nil
}

func (c *configServer) getNotifyUrl(notify *notify, conf *conf) (string, error) {
	addr := conf.server
	//if conf.env != ENV_DEV {
	//	ins, err := c.getOneInstance()
	//	if err != nil {
	//		return "", err
	//	}
	//	addr = ins.HomePage
	//}

	n := notify.getNotifyString()
	return fmt.Sprintf(
		"%s/notifications/v2?appId=%s&cluster=%s&notifications=%s",
		addr, conf.appId, conf.cluster, url.QueryEscape(n)), nil
}

func (c *configServer) getOneInstance() (instance, error) {
	size := len(c.instances)
	if size == 0 {
		return instance{}, fmt.Errorf("meta server all down %v", *c)
	}
	c.lock.RLock()
	defer c.lock.RUnlock()
	p := int(atomic.LoadInt64(&c.count) % int64(size))
	ins := c.instances[p]
	//简单的负载均衡
	atomic.AddInt64(&c.count, 1)
	return ins, nil
}

func (c *configServer) updateServers(conf *conf) error {

	defer func() {
		if err := recover(); err != nil {
			logger.Errorf("recover: %v", err)
		}
	}()

	if conf.server == "" {
		return nil
	}
	//fmt.Printf("updating instances with conf %v \n", *conf)
	serverUrl, err := c.getMetaServer(conf)
	if err != nil {
		return err
	}
	//rsp, err := http.Get(serverUrl)
	//if err != nil {
	//	return err
	//}
	//defer rsp.Body.Close()
	//data, err := ioutil.ReadAll(rsp.Body)
	//if err != nil {
	//	return err
	//}
	data, err := httpGet(serverUrl)
	if err != nil {
		return err
	}
	app := application{}
	err = xml.Unmarshal(data, &app)
	if err != nil {
		return fmt.Errorf("XML Unmarshal Fail err=%s , GET data=%s", err.Error(), string(data))
	}
	tmp := make([]instance, 0)
	for _, v := range app.Instance {
		if v.Status == "UP" {
			tmp = append(tmp, v)
		}
	}

	//fmt.Printf("up instance %v \n", tmp)

	c.lock.Lock()
	defer c.lock.Unlock()
	c.instances = tmp

	return nil
}

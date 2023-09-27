package apollo

import (
	"net"
	"os"
	"strings"
)

var ip string

const (
	prefixIP = "ip="
)

func init() {

	for _, v := range os.Args {
		if strings.HasPrefix(v, prefixIP) {
			ip = strings.Replace(v, prefixIP, "", 1)
			return
		}
	}

	conn, err := net.Dial("udp", "1.2.3.4:80")
	if err != nil {
		ip = "127.0.0.0"
		return
	}
	defer conn.Close()
	local := conn.LocalAddr().(*net.UDPAddr)
	ip = local.IP.String()
}

func LocalIP() string {
	return ip
}

func IpsNoLoopBack() (map[string]string, error) {
	// 去掉 回环地址
	ips := make(map[string]string)

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, i := range interfaces {
		byName, err := net.InterfaceByName(i.Name)
		if err != nil {
			return nil, err
		}
		addresses, err := byName.Addrs()
		// addresses, err := byName.MulticastAddrs()
		for _, address := range addresses {
			// if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet, ok := address.(*net.IPNet); ok {
				if ipnet.IP.To4() != nil {
					gInnerIP := ipnet.IP.String()
					ips[byName.Name] = gInnerIP
					// log.Println(gInnerIP, address.String())
				}
			}
			// ips[byName.Name] = v.String()
		}
	}
	return ips, nil
}

func IpsLoopBack() (map[string]string, error) {
	// 获取所有地址
	ips := make(map[string]string)

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, i := range interfaces {
		byName, err := net.InterfaceByName(i.Name)
		if err != nil {
			return nil, err
		}
		addresses, err := byName.Addrs()
		// addresses, err := byName.MulticastAddrs()
		for _, address := range addresses {
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				// if ipnet, ok := address.(*net.IPNet); ok {
				if ipnet.IP.To4() != nil {
					gInnerIP := ipnet.IP.String()
					ips[byName.Name] = gInnerIP
					// log.Println(gInnerIP, address.String())
				}
			}
			// ips[byName.Name] = v.String()
		}
	}
	return ips, nil
}

func IPsUsing() (map[string]string, error) {
	// 只获取正在使用的ip地址
	ips := make(map[string]string)

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, i := range interfaces {
		if (i.Flags & net.FlagUp) != 0 {
			byName, err := net.InterfaceByName(i.Name)
			if err != nil {
				return nil, err
			}
			addresses, err := byName.Addrs()
			for _, address := range addresses {
				if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {

					if ipnet.IP.To4() != nil {
						gInnerIP := ipnet.IP.String()
						ips[byName.Name] = gInnerIP

					}
				}

			}
		}

	}
	return ips, nil

}

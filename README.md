
<p align="center"><a href="https://github.com/luckzack/go-apollo"><img src="https://cdn.jsdelivr.net/gh/apolloconfig/apollo@master/doc/images/logo/logo-simple.png" alt="go-minimax" width="300" /></a></p>
<p align="center"><b>🚀 Apollo 配置中心 Go SDK </b></p>

---

## 安装
```bash
go get github.com/luckzack/go-apollo
```

## 快速使用
```go

import (
	apollo "github.com/luckzack/go-apollo"
)

func init(){
	// 注册你的多个环境的 meta server
	apollo.SetMetaServer(map[string]string{
        ENV_DEV: "http://127.0.0.1:8080",
        ENV_FAT: "http://127.0.0.2:8080",
        ENV_UAT: "http://127.0.0.3:8080",
        ENV_PRO: "http://127.0.0.4:8080",
    })
	
	// 配置当前使用的应用id和环境名
  	apollo.SetAppIDAndEnv("<app_id>", "dev")
	
}

func main(){
	// 获取配置实例
	c, err := apollo.GetConfig()
	if err != nil {
		return nil, err
	}
	// 拉取指定配置
	fmt.Println(c.GetString("sample_string"))
}
```

👉 [更多示例](./client_test.go)

## 参考

[Apollo开源地址](https://github.com/ctripcorp/apollo)

[Apollo配置中心介绍](https://www.apolloconfig.com/#/zh/design/apollo-introduction)
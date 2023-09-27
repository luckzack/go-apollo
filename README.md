# Usage

```go

import (
	apollo "github.com/luckpunk/apollo_client_go"
)

func init(){
  	apollo.SetAppIDAndEnv("<app_id>", "dev")
	err := apollo.Start()
	if err != nil {
		return nil, err
	}
}

func main(){
	c, err := apollo.GetConfig()
	if err != nil {
		return nil, err
	}
	fmt.Println(c.GetString("sample_string"))
}

```
package apollo

import (
	"fmt"
	"testing"
)

const ProjectKey = "project"
const ServiceKey = "service"
const EnvKey = "env"

func init() {
	SetMetaServer(map[string]string{
		ENV_DEV: "http://127.0.0.1:8080",
		ENV_FAT: "http://127.0.0.2:8080",
		ENV_UAT: "http://127.0.0.3:8080",
		ENV_PRO: "http://127.0.0.4:8080",
	})
	SetAppIDAndEnv("test_app", "local")
}

func getInst() (*Config, error) {
	err := start("test_app", ENV_DEV)
	if err != nil {
		return nil, err
	}

	c, err := GetConfig()
	if err != nil {
		return nil, err
	}
	return c, nil
}

// go test ./ -v -test.run=TestConfig_GetInt
func TestConfig_GetInt(t *testing.T) {

	c, err := GetConfig()
	if err != nil {
		t.Fatalf("apollo err: %s", err.Error())
		t.Fail()
		return
	}

	t.Log(c.GetInt("sample_int", 0))
}

// go test ./ -v -test.run=TestConfig_GetString
func TestConfig_GetString(t *testing.T) {
	c, err := GetConfig()
	if err != nil {
		t.Fatalf("apollo err: %s", err.Error())
		t.Fail()
		return
	}

	t.Log(c.GetString("sample_string"))
	// return aaa true

	t.Log(c.GetStringValue("sample_string", ""))
	// return aaa
}

// go test ./ -v -test.run=TestConfig_Watch
func TestConfig_Watch(t *testing.T) {

	c, err := GetConfig()
	if err != nil {
		t.Fatalf("apollo err: %s", err.Error())
		t.Fail()
		return
	}

	t.Log("watching...")
	go c.Watch(updateServiceHandler)

	select {}
}

func updateServiceHandler(n *Notice) {
	changed := []string{}
	for _, k := range n.GetChangeKeys() {

		changed = append(changed, k)
	}
	fmt.Println("changed keys:", changed)

	// then you can reload specified key as you wish
}

// go test ./ -v -test.run=TestConfig_GetList
func TestConfig_GetList(t *testing.T) {
	c, err := GetConfig()
	if err != nil {
		t.Fatalf("apollo err: %s", err.Error())
		t.Fail()
		return
	}

	t.Log(c.GetList("sample_list"))
	// return:  [1 2 3 4 5 a b c] true
}

// go test ./ -v -test.run=TestConfig_GetMap
func TestConfig_GetMap(t *testing.T) {

	c, err := GetConfig()
	if err != nil {
		t.Fatalf("apollo err: %s", err.Error())
		t.Fail()
		return
	}

	m := map[string]interface{}{}
	ok, err := c.GetJson("sample_map", &m)
	t.Log(ok, err, m)
	// return:  true <nil> map[a:1 a123:[1 2 3] aaa:xxxxxxxxxxx]
}

// go test ./ -v -test.run=TestConfig_GetStruct
func TestConfig_GetStruct(t *testing.T) {

	c, err := GetConfig()
	if err != nil {
		t.Fatalf("apollo err: %s", err.Error())
		t.Fail()
		return
	}

	s := struct {
		A    int    `json:"a"`
		A123 []int  `json:"a123"`
		AAA  string `json:"aaa"`
	}{}
	ok, err := c.GetJson("sample_map", &s)
	t.Log(ok, err, s)
	// return:  true <nil> {1 [1 2 3] xxxxxxxxxxx}
}

// go test ./ -v -test.run=TestConfig_GetStringInOtherNamespace
func TestConfig_GetStringInOtherNamespace(t *testing.T) {

	c, err := GetConfig()
	if err != nil {
		t.Fatalf("apollo err: %s", err.Error())
		t.Fail()
		return
	}
	str := c.GetStringByNameSpace("other_namespace", "other_key", "")
	t.Log(str)

	str2 := c.GetStringByNameSpace("other_namespace", "another_key", "")
	t.Log(str2)
}

// go test -bench=. -benchmem -benchtime=3s -run=BenchmarkConfig_GetString
func BenchmarkConfig_GetString(b *testing.B) {
	c, err := GetConfig()
	if err != nil {
		b.Fatalf("apollo err: %s", err.Error())
		b.Fail()
		return
	}

	for i := 0; i < b.N; i++ {
		c.GetString("sample_string")
	}
}

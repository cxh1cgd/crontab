/**
*FileName: config
*Create on 2018-12-18 16:32
*Create by mok
 */

package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

var Conf *Config = &Config{
	EtcdClient: &EtcdClient{Endpoints: make([]string, 0)},
}

type Config struct {
	*Server     `json:"server"`
	*EtcdClient `json:"etcd_client"`
}

//服务器配置
type Server struct {
	Cpu uint8 `json:"cpu"`
}

//etcd客户端配置
type EtcdClient struct {
	Endpoints   []string `json:"endpoints"`
	DialTimeOut int      `json:"dial_time_out"`
	MaxSendSize int      `json:"max_send_size"`
}

func Init(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, Conf)
}

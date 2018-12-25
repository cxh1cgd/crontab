/**
*FileName: config
*Create on 2018-12-18 16:37
*Create by mok
 */

package config

import (
	"fmt"
	"testing"
)

func TestInit(t *testing.T) {
	err := Init("./conf.json")
	if err != nil {
		t.Errorf("解析配置文件失败，err:%s", err)
		return
	}
	fmt.Println(Conf.Endpoints)
}

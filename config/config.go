// @Author Chen_dark
// @Date 2024/9/18 16:37:00
// @Desc
package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
)

type BATconfig struct {
	//密钥可前往官网控制台 https://console.cloud.tencent.com/cam/capi 进行获取
	TencentId  string `yaml:"TencentId"`
	TencentKey string `yaml:"TencentKey"`
	//密钥可前往官网控制台 https://ram.console.aliyun.com/manage/ak 进行获取
	AlibabaId  string `yaml:"AlibabaId"`
	AlibabaKey string `yaml:"AlibabaKey"`
	//密钥可前往官网控制台 https://console.bce.baidu.com/iam 进行获取
	BaiduId  string `yaml:"BaiduId"`
	BaiduKey string `yaml:"BaiduKey"`
	//密钥可前往官网控制台 https://console.volcengine.com/iam/keymanage/ 进行获取
	VolcengineId  string `yaml:"VolcengineId"`
	VolcengineKey string `yaml:"VolcengineKey"`
	//	秘钥前往控制台https://support.huaweicloud.com/devg-apisign/api-sign-provide-aksk.html#:~:text=%E6%93%8D%E4%BD%9C%E6%AD%A5%E9%AA%A4.%20%E6%9B%B4%E6%96%B0%E6%97%B6%E9%97%B4 获取
	HuaweiID  string `yaml:"HuaweiID"`
	HuaweiKey string `yaml:"HuaweiKey"`
}

type Result struct {
	Ip       string
	IsMatch  bool   //是否匹配到cdn
	Location string //ip位置
	Type     string //cdn、waf、cloud
	Value    string //值
}

func (r Result) String() string {
	if r.IsMatch {
		return fmt.Sprintf("匹配ip成功！ip:%s location:%s type:%s value:%s", r.Ip, r.Location, r.Type, r.Value)
	}
	return fmt.Sprintf("未找到！ ip:%s location:%s type:%s value:%s", r.Ip, r.Location, r.Type, r.Value)
}

// readConfig 从指定的 YAML 文件中读取配置信息并填充到 BATconfig 结构体中
func ReadConfig(filePath string) (*BATconfig, error) {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		log.Println("配置文件为空！即将创建配置文件，请配置config.yaml后再次运行")

		configContent := []byte(`
TencentId: ""
TencentKey: ""
AlibabaId: ""
AlibabaKey: ""
BaiduId: ""
BaiduKey: ""
VolcengineId: ""
VolcengineKey: ""
HuaweiID: ""
HuaweiKey: ""
		`)
		err = os.WriteFile("config.yaml", configContent, 0644)
		if err != nil {
			log.Fatal("创建配置文件失败", err)
		}
		os.Exit(1)
	}

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var batConfig BATconfig
	err = yaml.Unmarshal(data, &batConfig)
	if err != nil {
		return nil, err
	}

	return &batConfig, nil
}

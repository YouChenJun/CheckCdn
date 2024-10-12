package client

import (
	"CheckCdn/client/ipdb"
	"CheckCdn/config"
	"context"
	"fmt"
	ali_cdn20180510 "github.com/alibabacloud-go/cdn-20180510/v3/client"
	ali_openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ali_util "github.com/alibabacloud-go/tea-utils/v2/service"
	ali_tea "github.com/alibabacloud-go/tea/tea"
	bd_bce "github.com/baidubce/bce-sdk-go/bce"
	"github.com/gogf/gf/v2/encoding/gcharset"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/text/gregex"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/global"
	huawei_cdn "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cdn/v2"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cdn/v2/model"
	huawei_region "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cdn/v2/region"
	"github.com/projectdiscovery/retryabledns"
	tx_cdn "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdn/v20180606"
	tx_common "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tx_profile "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	hs_cdn "github.com/volcengine/volcengine-go-sdk/service/cdn"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
	"io/ioutil"
	"net"
	"sync"
)

var DefaultResolvers = []string{
	"1.1.1.1:53",
	"1.0.0.1:53",
	"8.8.8.8:53",
	"8.8.4.4:53",
}

// Client checks for CDN based IPs which should be excluded
// during scans since they belong to third party firewalls.
type Client struct {
	sync.Once
	cdn          *providerScraper
	waf          *providerScraper
	cloud        *providerScraper
	retriabledns *retryabledns.Client
	config       *config.BATconfig
}

// New creates cdncheck client with default options
// NewWithOpts should be preferred over this function
func New(config *config.BATconfig) *Client {
	resolvers := DefaultResolvers
	r, _ := retryabledns.New(resolvers, 3)
	client := &Client{
		cdn:          DefaultCDNProviders,
		waf:          DefaultWafProviders,
		cloud:        DefaultCloudProviders,
		retriabledns: r,
		config:       config,
	}
	return client
}

// GetCityByIp 获取ip所属城市
func (c *Client) GetCityByIp(input net.IP) string {
	ip := input.String()
	if ip == "::1" || ip == "127.0.0.1" {
		return "内网IP"
	}
	//优先通过内置ip库查询
	result := ipdb.GetCity(input)
	if result != "" {
		return result
	}
	url := "http://whois.pconline.com.cn/ipJson.jsp?json=true&ip=" + ip
	bytes := g.Client().GetBytes(context.TODO(), url)
	src := string(bytes)
	srcCharset := "GBK"
	tmp, _ := gcharset.ToUTF8(srcCharset, src)
	json, err := gjson.DecodeToJson(tmp)
	if err != nil {
		return ""
	}
	if json.Get("addr").String() != "" {
		return json.Get("addr").String()
	}
	return fmt.Sprintf("%s %s", json.Get("pro").String(), json.Get("city").String())
}

// 调用火山云接口，判断IP是否归属火山云
func (c *Client) Checkvolcengine(input net.IP) (cdn string, isp string) {
	if c.config.TencentId == "" {
		return "", ""
	}
	ip := input.String()
	region := "cn-beijing"
	config := volcengine.NewConfig().
		WithRegion(region).
		WithCredentials(credentials.NewStaticCredentials(c.config.VolcengineId, c.config.VolcengineKey, ""))
	sess, err := session.NewSession(config)
	if err != nil {
		panic(err)
	}
	svc := hs_cdn.New(sess)
	describeCdnIPInput := &hs_cdn.DescribeCdnIPInput{
		IPs: volcengine.StringSlice([]string{ip}),
	}

	// 复制代码运行示例，请自行打印API返回值。
	response, err := svc.DescribeCdnIP(describeCdnIPInput)
	if err != nil {
		// 复制代码运行示例，请自行打印API错误信息。
		fmt.Println("火山接口调用出错！", err)
		return "", ""
	}
	patternStr := `CdnIp: (.*?),`
	Platform, err := gregex.MatchString(patternStr, response.String())
	if err != nil {
		return "", ""
	}
	if Platform[1] == "True" {
		patternStr = `Location: "(.*?)"`
		result, reerr := gregex.MatchString(patternStr, response.String())
		if reerr != nil {
			return "火山云", ""
		}
		return "火山云", result[1]
	}
	return "", ""
}

// 调用腾讯云DescribeCdnIp接口，判断ip是否属于腾讯云
func (c *Client) CheckTencent(input net.IP) (cdn string, isp string) {
	if c.config.TencentId == "" {
		return "", ""
	}
	ip := input.String()
	// 实例化一个认证对象，入参需要传入腾讯云账户 SecretId 和 SecretKey，此处还需注意密钥对的保密
	// 密钥可前往官网控制台 https://console.cloud.tencent.com/cam/capi 进行获取
	credential := tx_common.NewCredential(
		c.config.TencentId,
		c.config.TencentKey,
	)
	cpf := tx_profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "cdn.tencentcloudapi.com"
	// 实例化要请求产品的client对象,clientProfile是可选的
	client, _ := tx_cdn.NewClient(credential, "", cpf)
	// 实例化一个请求对象,每个接口都会对应一个request对象
	request := tx_cdn.NewDescribeCdnIpRequest()
	request.Ips = tx_common.StringPtrs([]string{ip})
	response, err := client.DescribeCdnIp(request)
	//fmt.Print(response.ToJsonString())
	if err != nil {
		fmt.Println("腾讯接口调用出错！", err)
		return "", ""
	}
	patternStr := `"Platform":"(.*?)"`
	Platform, err := gregex.MatchString(patternStr, response.ToJsonString())
	if err != nil {
		return "", ""
	}
	if Platform[1] == "yes" {
		patternStr = `"Location":"(.*?)"`
		result, reerr := gregex.MatchString(patternStr, response.ToJsonString())
		if reerr != nil {
			return "腾讯云", ""
		}
		return "腾讯云", result[1]
	}
	return "", ""
}

// 调用阿里云DescribeIpInfo接口，判断ip是否属于阿里云
func (c *Client) CheckAliyun(input net.IP) (cdn string, isp string) {
	if c.config.AlibabaId == "" {
		return "", ""
	}
	ip := input.String()
	config := &ali_openapi.Config{
		// 必填，您的 AccessKey ID
		AccessKeyId: ali_tea.String(c.config.AlibabaId),
		// 必填，您的 AccessKey Secret
		AccessKeySecret: ali_tea.String(c.config.AlibabaKey),
	}
	config.Endpoint = ali_tea.String("cdn.aliyuncs.com")
	client := &ali_cdn20180510.Client{}
	client, err := ali_cdn20180510.NewClient(config)
	if err != nil {
		fmt.Println("阿里接口调用出错！", err)
		return "", ""
	}
	describeIpInfoRequest := &ali_cdn20180510.DescribeIpInfoRequest{IP: ali_tea.String(ip)}
	runtime := &ali_util.RuntimeOptions{}
	response, err := client.DescribeIpInfoWithOptions(describeIpInfoRequest, runtime)
	if err != nil {
		return "", ""
	}
	//fmt.Printf("%s",response.Body.String())
	json, err := gjson.DecodeToJson(response.Body.String())
	if err != nil {
		return "", ""
	}
	if json.Get("CdnIp").String() == "True" {
		return "阿里云", json.Get("ISP").String()
	} else {
		return "", ""
	}
}

// 调用百度云describeIp接口，判断ip是否属于百度云
func (c *Client) CheckBaidu(input net.IP) (cdn string, isp string) {
	if c.config.BaiduId == "" {
		return "", ""
	}
	ip := input.String()
	req := &bd_bce.BceRequest{}
	req.SetUri("/v2/utils")
	req.SetMethod("GET")
	req.SetParams(map[string]string{"action": "describeIp", "ip": ip})
	req.SetHeaders(map[string]string{"Accept": "application/json"})
	payload, _ := bd_bce.NewBodyFromString("")
	req.SetBody(payload)
	client, err := bd_bce.NewBceClientWithAkSk(c.config.BaiduId, c.config.BaiduKey, "https://cdn.baidubce.com")
	if err != nil {
		fmt.Println("百度接口调用出错！", err)
		return "", ""
	}
	resp := &bd_bce.BceResponse{}
	err = client.SendRequest(req, resp)
	if err != nil {
		return "", ""
	}
	respBody := resp.Body()
	defer respBody.Close()
	body, err := ioutil.ReadAll(respBody)
	if err != nil {
		return "", ""
	}
	json, err := gjson.DecodeToJson(string(body))
	if err != nil {
		return "", ""
	}
	if json.Get("cdnIP").String() == "true" {
		return "百度云", json.Get("isp").String()
	} else {
		return "", ""
	}
}

// 调用华为云接口，判断IP是否属于华为云
func (c *Client) CheckHuawei(input net.IP) (cdn string, isp string) {
	if c.config.HuaweiID == "" {
		return "", ""
	}
	ip := input.String()

	auth := global.NewCredentialsBuilder().
		WithAk(c.config.HuaweiID).
		WithSk(c.config.HuaweiKey).
		Build()

	client := huawei_cdn.NewCdnClient(
		huawei_cdn.CdnClientBuilder().
			WithRegion(huawei_region.ValueOf("cn-north-1")).
			WithCredential(auth).
			Build())

	request := &model.ShowIpInfoRequest{}
	request.Ips = ip
	response, err := client.ShowIpInfo(request)
	if err != nil {
		return "", ""
	}
	json, err := gjson.DecodeToJson(response)
	if err != nil {
		return "", ""
	}
	//归属
	if json.Get("belongs").String() == "true" {
		return "华为云", json.Get("ip").String()
	}
	return "", ""
}

// Check checks if ip belongs to one of CDN, WAF and Cloud . It is generic method for Checkxxx methods
func (c *Client) Check(inputIp string) (result config.Result) {
	result.Ip = inputIp
	ip := net.ParseIP(inputIp)
	if ip == nil {
		result.IsMatch = false
		return
	}
	location := c.GetCityByIp(ip)
	result.Location = location

	//腾讯
	if cdn, isp := c.CheckTencent(ip); cdn != "" {
		result.Location = location + " " + isp
		result.IsMatch = true
		result.Type = "tencent cdn-官方接口查询"
		result.Value = cdn
		return
	}
	if cdn, isp := c.Checkvolcengine(ip); cdn != "" {
		result.Location = location + " " + isp
		result.IsMatch = true
		result.Type = "火山云 cdn-官方接口查询"
		result.Value = cdn
		return
	}
	//阿里
	if cdn, isp := c.CheckAliyun(ip); cdn != "" {
		result.Location = location + " " + isp
		result.IsMatch = true
		result.Type = "Aliyun cdn-官方接口查询"
		result.Value = cdn
		return
	}
	//百度
	if cdn, isp := c.CheckBaidu(ip); cdn != "" {
		result.Location = location + " " + isp
		result.IsMatch = true
		result.Type = "Baidu cdn-官方接口查询"
		result.Value = cdn
		return
	}
	//华为
	if cdn, isp := c.CheckHuawei(ip); cdn != "" {
		result.Location = location + " " + isp
		result.IsMatch = true
		result.Type = "华为云 cdn-官方接口查询"
		result.Value = cdn
		return
	}

	//ip库做兜底
	//通过内置字典，检测cdn、waf、cloud
	if matched, value, err := c.cdn.Match(ip); err == nil && matched && value != "" {
		result.IsMatch = matched
		result.Type = "cdn-本地数据库"
		result.Value = value
		return
	}
	if matched, value, err := c.waf.Match(ip); err == nil && matched && value != "" {
		result.IsMatch = matched
		result.Type = "waf-本地数据库"
		result.Value = value
		return
	}
	if matched, value, err := c.cloud.Match(ip); err == nil && matched && value != "" {
		result.IsMatch = matched
		result.Type = "cloud-本地数据库"
		result.Value = value
		return
	}
	result.IsMatch = false
	return
}

// Check Domain with fallback checks if domain belongs to one of CDN, WAF and Cloud . It is generic method for Checkxxx methods
// Since input is domain, as a fallback it queries CNAME records and checks if domain is WAF
func (c *Client) CheckDomainWithFallback(domain string) (result config.Result) {
	result.Ip = domain
	dnsData, err := c.retriabledns.Resolve(domain)
	result.IsMatch = false
	if err != nil {
		return
	}
	result = c.CheckDNSResponse(dnsData)
	if result.IsMatch {
		return
	}
	// resolve cname
	dnsData, err = c.retriabledns.CNAME(domain)
	return c.CheckDNSResponse(dnsData)
}

// CheckDNSResponse is same as CheckDomainWithFallback but takes DNS response as input
func (c *Client) CheckDNSResponse(dnsResponse *retryabledns.DNSData) (result config.Result) {
	if dnsResponse.A != nil {
		for _, ip := range dnsResponse.A {
			result := c.Check(ip)
			if result.IsMatch {
				result.Ip = ip
				return result
			}
		}
	}
	if dnsResponse.CNAME != nil {
		matched, discovered, itemType, err := c.CheckSuffix(dnsResponse.CNAME...)
		if err != nil {
			result.IsMatch = false
			return
		}
		if matched {
			// for now checkSuffix only checks for wafs
			result.IsMatch = true
			result.Type = itemType
			result.Value = discovered
			return
		}
	}
	result.IsMatch = false
	return
}

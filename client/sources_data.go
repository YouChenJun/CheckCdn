package client

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed data/sources_data.json
var sources_data string

//go:embed data/sources_china.json
var sources_china string

var generatedData InputCompiled

var (
	DefaultCDNProviders   *providerScraper
	DefaultWafProviders   *providerScraper
	DefaultCloudProviders *providerScraper
)

func init() {
	if err := json.Unmarshal([]byte(sources_data), &generatedData); err != nil {
		panic(fmt.Sprintf("Could not parse cidr data: %s", err))
	}
	tmpCdn := generatedData.CDN
	tmpWaf := generatedData.WAF
	tempCloud := generatedData.Cloud
	if err := json.Unmarshal([]byte(sources_china), &generatedData); err != nil {
		panic(fmt.Sprintf("Could not parse cidr data: %s", err))
	}
	DefaultCDNProviders = newProviderScraper(mergeMaps(tmpCdn, generatedData.CDN))
	DefaultWafProviders = newProviderScraper(mergeMaps(tmpWaf, generatedData.WAF))
	DefaultCloudProviders = newProviderScraper(mergeMaps(tempCloud, generatedData.Cloud))
}

func mergeMaps(map1, map2 map[string][]string) map[string][]string {
	for key, value := range map2 {
		if _, ok := map1[key]; ok {
			map1[key] = append(map1[key], value...)
		} else {
			map1[key] = value
		}
	}
	return map1
}

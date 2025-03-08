package main

import (
	"flag"
	"github.com/YouChenJun/CheckCdn/client"
	"github.com/YouChenJun/CheckCdn/cmd"
)

var (
	inputIpFilePath = flag.String("input", "", "需要检测的ip文件列表")
	noCdnIP         = flag.String("output", "nocdn.txt", "不是CDN节点的ip文本列表")
	configFilePath  = flag.String("config", "config.yaml", "配置文件夹路径")
	delaySeconds    = flag.Float64("delayed", 0, "查询延迟时间,默认0s")
	dbPath          = flag.String("db", "", "ipdb数据库路径,默认为当前目录下的cdn_cache.db")
)

func main() {
	flag.Parse()
	// 设置数据库路径
	if *dbPath != "" {
		client.SetDBPath(*dbPath)
	}
	runnerConfig := &cmd.RunnerConfig{
		InputIpFilePath: *inputIpFilePath,
		NoCdnIP:         *noCdnIP,
		ConfigFilePath:  *configFilePath,
		DelaySeconds:    *delaySeconds,
		DbPath:          *dbPath,
	}
	cmd.Run(runnerConfig)
}

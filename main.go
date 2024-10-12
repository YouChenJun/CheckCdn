// @Author Chen_dark
// @Date 2024/9/18 16:35:00
// @Desc
package main

import (
	"CheckCdn/cmd"
	"CheckCdn/config"
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

var (
	inputIpFilePath = flag.String("input", "", "需要检测的ip文件列表")
	noCdnIP         = flag.String("output", "nocdn.txt", "不是CDN节点的ip文本列表")
	configFilePath  = flag.String("config", "config.yaml", "配置文件夹路径")
	delaySeconds    = flag.Float64("delayed", 0, "查询延迟时间,默认0s")
)

func main() {
	flag.Parse()
	fmt.Println(cmd.Banner)
	// 检查是否提供了输出文件路径
	if *noCdnIP == "" || *noCdnIP == "" || *configFilePath == "" {
		log.Fatal("必须提供 -output 参数以指定输出文件路径")
	}

	// 读取配置文件
	batConfig, err := config.ReadConfig(*configFilePath)
	if err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}

	client := cmd.New(batConfig)

	// 打开IP文件
	file, err := os.Open(*inputIpFilePath)
	if err != nil {
		log.Fatalf("打开IP文件失败: %v", err)
	}
	defer file.Close()

	// 创建或打开输出文件
	outputFile, err := os.Create(*noCdnIP)
	if err != nil {
		log.Fatalf("创建输出文件失败: %v", err)
	}
	defer outputFile.Close()

	writer := bufio.NewWriter(outputFile)

	scanner := bufio.NewScanner(file)
	fmt.Println("正在开始检测...使用前请检查配置文件:", *configFilePath)
	t1 := time.Now()
	for scanner.Scan() {
		ip := scanner.Text()
		time.Sleep(time.Duration(*delaySeconds) * time.Second)
		result := client.Check(ip)
		if result.Type != "" {
			fmt.Println(result)
		}
		if result.Type == "" {
			_, err := writer.WriteString(result.Ip + "\n")
			if err != nil {
				log.Fatalf("写入输出文件时出错: %v", err)
			}
		}
	}
	elapsed := time.Since(t1)
	fmt.Println("运行结束，运行时长:", elapsed)
	// 确保所有数据都被写入文件
	err = writer.Flush()
	if err != nil {
		log.Fatalf("刷新缓冲区时出错: %v", err)
	}

	if err = scanner.Err(); err != nil {
		log.Fatalf("读取IP文件时出错: %v", err)
	}
	//var ipList = []string{"117.23.61.32", "124.232.162.187", "113.105.168.118", "111.174.1.35", "36.155.132.3"}
	//for _, ip := range ipList {
	//	result := client.Check(ip)
	//	fmt.Println(result)
	//}
}

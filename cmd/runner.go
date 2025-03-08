package cmd

import (
	"bufio"
	"fmt"
	clients "github.com/YouChenJun/CheckCdn/client"
	conf "github.com/YouChenJun/CheckCdn/config"
	"log"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"
)

// RunnerConfig 包含运行程序所需的配置
type RunnerConfig struct {
	InputIpFilePath string
	NoCdnIP         string
	ConfigFilePath  string
	DelaySeconds    float64
	DbPath          string
}

// Run 执行IP检测的主要逻辑
func Run(config *RunnerConfig) {
	fmt.Println(Banner)

	// 检查配置文件路径
	if config.ConfigFilePath == "" {
		slog.Error("必须提供 -config 参数以指定配置文件路径")
		return
	}

	// 读取配置文件
	batConfig, err := conf.ReadConfig(config.ConfigFilePath)

	if err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}

	client := New(batConfig)

	// 打开IP文件
	if config.InputIpFilePath == "" {
		slog.Error("必须提供 -input 参数以指定输入文件路径")
		return
	}
	file, err := os.Open(config.InputIpFilePath)
	if err != nil {
		slog.Error("打开IP文件失败: %v", err)
		return
	}
	defer file.Close()

	// 创建或打开输出文件
	if config.NoCdnIP == "" {
		slog.Error("必须提供 -output 参数以指定输出文件路径")
		return
	}
	outputFile, err := os.Create(config.NoCdnIP)
	if err != nil {
		log.Fatalf("创建输出文件失败: %v", err)
	}
	defer outputFile.Close()

	writer := bufio.NewWriter(outputFile)

	scanner := bufio.NewScanner(file)
	fmt.Println("正在开始检测...使用前请检查配置文件:", config.ConfigFilePath)
	t1 := time.Now()

	// 使用channel来收集结果
	resultChan := make(chan conf.Result)
	done := make(chan bool)

	// 启动goroutine来处理结果
	go func() {
		for result := range resultChan {
			if result.Type != "" {
				fmt.Println(result)
			}
			if result.Type == "" {
				_, err := writer.WriteString(result.Ip + "\n")
				if err != nil {
					slog.Error("写入输出文件时出错: %v", err)
				}
			}
		}
		done <- true
	}()

	// 使用带缓冲的channel来控制并发数
	concurrencyLimit := 5 //并发数-最好不要修改，部分云服务有并发限制
	semaphore := make(chan struct{}, concurrencyLimit)
	var wg sync.WaitGroup

	for scanner.Scan() {
		ip := scanner.Text()
		parsedIP := net.ParseIP(ip)
		if parsedIP == nil || parsedIP.IsUnspecified() || clients.IsPrivateIP(parsedIP) || clients.IsCommonDNS(ip) {
			continue
		}
		wg.Add(1)
		semaphore <- struct{}{} // 获取一个信号量
		go func(ip string) {
			defer wg.Done()
			defer func() { <-semaphore }() // 释放信号量
			time.Sleep(time.Duration(config.DelaySeconds) * time.Second)
			result := client.Check(ip)
			resultChan <- result
		}(ip)
	}

	// 等待所有goroutine完成
	wg.Wait()
	close(resultChan)
	<-done

	elapsed := time.Since(t1)
	fmt.Println("运行结束，运行时长:", elapsed)
	// 确保所有数据都被写入文件
	err = writer.Flush()
	if err != nil {
		slog.Error("刷新缓冲区时出错: %v", err)
	}

	if err = scanner.Err(); err != nil {
		slog.Error("读取IP文件时出错: %v", err)
	}
}

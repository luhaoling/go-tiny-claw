package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("🚀 欢迎来到 go-tiny-claw 引擎启动序列")

	// todo: 1. 初始化模型 Provider（大脑）
	// provider:=provider.NewClaudeProvider(...)

	// todo: 2. 初始化 Tool Registry（手脚）
	// registry：=tools.NewRegistry()
	// registry.Register(tools.NewBashTool())

	// todo: 3. 初始化上下文管理器（内存管理器）
	// ctxManager:=context.NewManager(...)

	// todo: 4. 组装并启动核心 Engine（操作系统心脏）
	// engine：=engine.NewAgentEngine(provider,ctxManger)

	// fmt.Println("开始执行任务...")
	// err:=engine.Run("帮我检查一下当前目录下的文件并输出一个 README.md 大纲")
	// if err!=nil{
	//	 log.Fatalf("引擎运行崩溃:%v",err)
	// }

	log.Println("架构蓝图搭建完毕，等待各核心模块注入")

}

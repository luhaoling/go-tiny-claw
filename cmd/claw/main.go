package main

import (
	"AgentHarnes/internal/engine"
	"AgentHarnes/internal/provider"
	"AgentHarnes/internal/tools"
	"context"
	"log"
	"os"
)

func main() {
	if os.Getenv("ZHIPU_API_KEY") == "" {
		log.Fatalf("请先设置 ZHIPU_API_KEY 环境变量")
	}
	// 1. 获取工作区物理边界
	workdir, _ := os.Getwd()

	// 2. 初始化真实的 Provider 大脑(指向智谱）
	llmProvider := provider.NewZhipuOpenAIProvider("glm-4.5-air")

	// 3. 初始化真实的 Too Registry
	registry := tools.NewRegistry()

	// 4. 将真实的 readFile 工具挂载在注册表中
	registry.Register(tools.NewReadFileTool(workdir))
	registry.Register(tools.NewWriteFileTool(workdir))
	registry.Register(tools.NewBashTool(workdir))
	registry.Register(tools.NewEditFileTool(workdir))

	// 5. 实例化并运行引擎，关闭 EnableThinking =true 开启慢思考模式
	eng := engine.NewAgentEngine(llmProvider, registry, workdir, false)

	// 6. 下发一个需要通过真实工具才能完成的任务
	prompt := `
	我当前目录下有 a.txt, b.txt, c.txt 三个文件。 为了节省时间，请你同时一次性读取这三个文件，并将它们的内容综合起来，告诉我它们分别记录了什么领域的信息.
你需要创建一个 d.txt 文件，里面写入：这个是监控文件，监控项目运行指标。
	`
	err := eng.Run(context.Background(), prompt)
	if err != nil {
		log.Fatalf("引擎崩溃:%v", err)
	}

}

//
//func main() {
//	fmt.Println("🚀 欢迎来到 go-tiny-claw 引擎启动序列")
//
//	// todo: 1. 初始化模型 Provider（大脑）
//	// provider:=provider.NewClaudeProvider(...)
//
//	// todo: 2. 初始化 Tool Registry（手脚）
//	// registry：=tools.NewRegistry()
//	// registry.Register(tools.NewBashTool())
//
//	// todo: 3. 初始化上下文管理器（内存管理器）
//	// ctxManager:=context.NewManager(...)
//
//	// todo: 4. 组装并启动核心 Engine（操作系统心脏）
//	// engine：=engine.NewAgentEngine(provider,ctxManger)
//
//	// fmt.Println("开始执行任务...")
//	// err:=engine.Run("帮我检查一下当前目录下的文件并输出一个 README.md 大纲")
//	// if err!=nil{
//	//	 log.Fatalf("引擎运行崩溃:%v",err)
//	// }
//
//	log.Println("架构蓝图搭建完毕，等待各核心模块注入")
//
//}

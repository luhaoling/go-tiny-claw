package main

import (
	"AgentHarnes/internal/engine"
	"AgentHarnes/internal/feishu"
	"AgentHarnes/internal/provider"
	"AgentHarnes/internal/tools"
	"github.com/larksuite/oapi-sdk-go/v3/core/httpserverext"
	"log"
	"net/http"
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
	eng := engine.NewAgentEngine(llmProvider, registry, workdir, true)

	// 6. 初始化飞书 bot 调度器
	bot := feishu.NewFeishuBot(eng)
	handler := httpserverext.NewEventHandlerFunc(bot.GetEnentDispatcher())

	// 7. 注册路由并启动 HTTP 服务
	http.HandleFunc("/webhook/event", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[HTTP] 收到飞书回调请求: method=%s path=%s remote=%s", r.Method, r.URL.Path, r.RemoteAddr)
		handler(w, r)
	})

	port := ":48080"

	log.Printf("🚀 go-tiny-claw 飞书服务端已启动，正在监听 %s 端口\n", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("服务器启动失败:%v", err)
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

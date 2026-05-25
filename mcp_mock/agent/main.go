package main

import (
	"context"
	"fmt"
	"log"
	"os"


	"github.com/cloudwego/eino-ext/callbacks/langsmith"
	"github.com/cloudwego/eino-ext/components/model/ark"
	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func main(){
	ctx := context.Background()
	if err := godotenv.Overload(); err != nil{
		log.Fatal(err)
	}
	fmt.Println( os.Getenv("ARK_API_KEY"))
	fmt.Println( os.Getenv("LANG_SMITH_KEY"))
	traceHandler,err := langsmith.NewLangsmithHandler(&langsmith.Config{
		APIKey:  os.Getenv("LANG_SMITH_KEY"),
		// APIURL:  os.Getenv("LANG_SMITH_URL"),
	})
	if err != nil {
		log.Fatal(err)
	}
	callbacks.AppendGlobalHandlers(traceHandler)
	log.Println("LangSmith 全局回调已启用")
	mcpTools,cleanup := connectMCPSever(ctx)
	
	defer cleanup()
	fmt.Printf("从 MCP Server 获取到 %d 个工具\n\n", len(mcpTools))
	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey: os.Getenv("ARK_API_KEY"),
		Model:  os.Getenv("Model")})
	if err != nil {
		log.Fatal(err)
	}
	agent,err := react.NewAgent(ctx,&react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: mcpTools,
		},
	})
	if err != nil{
		log.Fatal(err)
	}
	    // ========== 第四步：与 Agent 对话 ==========
    questions := []string{
       "请帮我把 'hello world from mcp' 这段文字转成大写",
       "现在几点了？",
       "请帮我算一下从 2024-01-01 到 2025-04-16 一共有多少天",
       "请帮我把 '人工智能改变世界' 这句话反转一下，然后再统计一下原文有多少个字",
    }

    for i, q := range questions {
       fmt.Printf("===== 问题 %d =====\n", i+1)
       fmt.Printf("用户: %s\n\n", q)

       result, err := agent.Generate(ctx, []*schema.Message{
          {Role: schema.User, Content: q},
       })
       if err != nil {
          fmt.Printf("Agent 执行出错: %v\n\n", err)
          continue
       }
       fmt.Printf("Agent: %s\n\n", result.Content)
    }
}

func connectMCPSever(ctx context.Context)([]tool.BaseTool,func()){
	cli,err := client.NewSSEMCPClient("http://localhost:8080/sse")
	if err != nil{
		log.Fatal(err)
	}
	if err := cli.Start(ctx); err != nil{
		log.Fatal(err)
	}
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name: "eino-agent",
		Version: "1.0,0",
	}
	if _,err := cli.Initialize(ctx,initReq);err != nil{
		log.Fatal(err)
	}
	tools,err := mcpp.GetTools(ctx, &mcpp.Config{
		Cli: cli,
	})
	if err != nil{
		log.Fatal(err)
	}
	return tools,func ()  {
		cli.Close()
	}
}
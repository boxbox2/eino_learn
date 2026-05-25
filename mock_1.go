package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

/*
我们要构建一个"技术问答路由系统”:接收用户提问，先判断属于前端、后端还是通用问题，
走不同的模板分支让模型以对应角色回答，最后格式化输出。
*/
func main_1() {
	    if err := godotenv.Load(); err != nil {
        log.Println("load .env failed:", err)
    }
	ctx := context.Background()
	//定义模型
	model,err := ark.NewChatModel(ctx,&ark.ChatModelConfig{
		APIKey: os.Getenv("ARK_API_KEY"),
		Model: os.Getenv("Model"),
	})
	if err != nil{
		log.Fatal(err)
	}
	//建立节点
	g := compose.NewGraph[map[string]any,string]()
	classifier := compose.InvokableLambda(func(ctx context.Context,input map[string]any) (map[string]any, error) {
		question := input["question"].(string)
		lower := strings.ToLower(question)

		category := "general"
       if strings.ContainsAny(lower, "前端csshtml") ||
          strings.Contains(lower, "react") ||
          strings.Contains(lower, "vue") ||
          strings.Contains(lower, "javascript") {
          category = "frontend"
       } else if strings.Contains(lower, "go") ||
          strings.Contains(lower, "数据库") ||
          strings.Contains(lower, "并发") ||
          strings.Contains(lower, "微服务") ||
          strings.Contains(lower, "api") {
          category = "backend"
       }

       input["category"] = category
       return input, nil
    })
	frontendTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个前端开发专家，精通React、Vue、CSS和浏览器原理，请用简洁的语言回答问题。"),
		schema.UserMessage("{question}"),
	)
	    backendTpl := prompt.FromMessages(schema.FString,
       schema.SystemMessage("你是一个后端开发专家，精通Go、数据库、微服务架构和高并发设计，请用简洁的语言回答问题。"),
       schema.UserMessage("{question}"),
    )
    generalTpl := prompt.FromMessages(schema.FString,
       schema.SystemMessage("你是一个全栈技术顾问，请用通俗易懂的语言回答问题。"),
       schema.UserMessage("{question}"),
    )
	//添加节点
	_ = g.AddLambdaNode("classifier",classifier)
	_ = g.AddChatTemplateNode("frontend_tpl", frontendTpl)
    _ = g.AddChatTemplateNode("backend_tpl", backendTpl)
    _ = g.AddChatTemplateNode("general_tpl", generalTpl)
	_ = g.AddChatModelNode("model",model)
	_ = g.AddLambdaNode("formatter", compose.InvokableLambda(
       func(ctx context.Context, msg *schema.Message) (string, error) {
          return fmt.Sprintf("[AI助手] %s", msg.Content), nil
       },
    ))
	//连接节点
	_ = g.AddEdge(compose.START,"classifier")
	_ = g.AddBranch("classifier",compose.NewGraphBranch(
		func(ctx context.Context,input map[string]any)(string,error){
			return input["category"].(string) + "_tpl", nil
		},
		map[string]bool{
			"frontend_tpl": true,
			"backend_tpl":  true,
            "general_tpl":  true,
		},
	))
	    // 汇聚到模型
    _ = g.AddEdge("frontend_tpl", "model")
    _ = g.AddEdge("backend_tpl", "model")
    _ = g.AddEdge("general_tpl", "model")
    _ = g.AddEdge("model", "formatter")
    _ = g.AddEdge("formatter", compose.END)

	runner,err := g.Compile(ctx)
	if err != nil {
		log.Fatal("编译失败",err)

	}
	    // 测试多个问题
    questions := []string{
       "React的useEffect和useLayoutEffect有什么区别？",
       "Go的GMP调度模型是怎么工作的？",
       "新手程序员应该先学前端还是后端？",
    }
	for _, q := range questions{
		res,err := runner.Invoke(ctx,map[string]any{"question":q})
		if err != nil{
			log.Printf("错误: %v\n", err)
          	continue
		}
		fmt.Printf("问题: %s\n%s\n\n", q, res)
	}
}
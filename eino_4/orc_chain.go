package eino_4

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

func OrcChain() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	timeout := 30 * time.Second
	//初始化模型
	model, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		Timeout: &timeout,
		APIKey:  os.Getenv("API_KEY"),
		Model:   os.Getenv("MODEL"),
	})
	if err != nil {
		panic(err)
	}
	//创建节点
	lambda := compose.InvokableLambda(func(ctx context.Context, input string) (output []*schema.Message, err error) {
		desuwa := input + "回答结尾加上desuwa"
		output = []*schema.Message{
			{
				Role:    schema.User,
				Content: desuwa,
			},
		}
		return output, nil
	})
	chain := compose.NewChain[string, *schema.Message]()
	//连接节点
	chain.AppendLambda(lambda).AppendChatModel(model)
	//编译运行
	r, err := chain.Compile(ctx)
	if err != nil {
		panic(err)
	}
	answer, err := r.Invoke(ctx, "您好，请告诉我你的名字")
	if err != nil {
		panic(err)
	}
	fmt.Println(answer)
}

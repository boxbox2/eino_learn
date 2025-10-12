package main

import (
	"context"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/joho/godotenv"
)

func createOpenAIChatModel(ctx context.Context) model.ToolCallingChatModel {
	// 加载 .env 文件
	err := godotenv.Load()
	if err != nil {
		log.Println("未找到 .env 文件，将尝试从系统环境变量读取")
	}
	key := os.Getenv("OPENAI_KEY")
	modelName := os.Getenv("OPENAI_MODEL_NAME")
	baseURL := os.Getenv("OPENAI_BASE_URL")
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
		APIKey:  key,
	})
	if err != nil {
		log.Fatalf("create openai chat model failed, err=%v", err)
	}
	return chatModel
}

package main

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

func promt_test() {
	systemTpl := `你是情绪助手，你的任务是根据用户的输入，生成一段赞美的话，语句优美，韵律强。
用户姓名：{user_name}
用户年龄：{user_age}
用户性别：{user_gender}
用户喜好：{user_hobby}`

	chatTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage(systemTpl),
		schema.MessagesPlaceholder("message_histories", true),
		schema.UserMessage("{user_query}"))

	msgList, err := chatTpl.Format(context.Background(), map[string]any{"user_name": "张三",
		"user_age":    "18",
		"user_gender": "男",
		"user_hobby":  "打篮球、打游戏",
		"message_histories": []*schema.Message{
			schema.UserMessage("我喜欢打羽毛球"),
			schema.AssistantMessage("xxxxxxxx", nil),
		},
		"user_query": "请为我赋诗一首"})

	if err != nil {
		panic(err)
	}

	for _, msg := range msgList {
		fmt.Printf("- %v", msg)
	}
}

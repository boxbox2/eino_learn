package main

import (
        "context"
        "fmt"
        "log"
        "os"

        "github.com/cloudwego/eino-ext/components/model/openai"
        "github.com/cloudwego/eino/components/prompt"
        "github.com/cloudwego/eino/compose"
        "github.com/cloudwego/eino/schema"
)
/*
我们要构建一个"多维度文本分析“系统:接收一段文本输入，
同时让模型从“情感分析“和"关键词提取“两个维度做分析，最后把两个维度的结果合并成一份完整的分析报告。
*/
func main() {
        ctx := context.Background()

        // 创建模型
        model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
                BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
                APIKey:  os.Getenv("DASHSCOPE_API_KEY"),
                Model:   "qwen-plus",
        })
        if err != nil {
                log.Fatal(err)
        }

        // 定义输入
        type AnalysisInput struct {
                Text     string
                TopN     string // 提取关键词的数量，用字符串方便模板渲染
        }

        // 情感分析 Prompt 模板
        sentimentTpl := prompt.FromMessages(schema.FString,
                schema.SystemMessage("你是一个情感分析专家。请分析以下文本的情感倾向，输出格式：情感倾向（正面/负面/中性）+ 一句话理由。"),
                schema.UserMessage("{text}"),
        )

        // 关键词提取 Prompt 模板
        keywordTpl := prompt.FromMessages(schema.FString,
                schema.SystemMessage("你是一个关键词提取专家。请从以下文本中提取{top_n}个最重要的关键词，用逗号分隔输出。"),
                schema.UserMessage("{text}"),
        )

        // 情感分析链：模板 → 模型
        sentimentChain := compose.NewChain[map[string]any, *schema.Message]()
        sentimentChain.AppendChatTemplate(sentimentTpl).AppendChatModel(model)

        // 关键词提取链：模板 → 模型
        keywordChain := compose.NewChain[map[string]any, *schema.Message]()
        keywordChain.AppendChatTemplate(keywordTpl).AppendChatModel(model)

        // 构建 Workflow
        wf := compose.NewWorkflow[AnalysisInput, string]()

        // 情感分析节点：只需要 Text 字段，映射到模板变量 text
        wf.AddGraphNode("sentiment", sentimentChain).
                AddInput(compose.START, compose.MapFields("Text", "text"))

        // 关键词提取节点：需要 Text 和 TopN 两个字段
        wf.AddGraphNode("keywords", keywordChain).
                AddInput(compose.START,
                        compose.MapFields("Text", "text"),
                        compose.MapFields("TopN", "top_n"),
                )

        // 合并结果的 Lambda
        wf.AddLambdaNode("merge", compose.InvokableLambda(
                func(ctx context.Context, results map[string]any) (string, error) {
                        sentiment := results["sentiment"].(*schema.Message)
                        keywords := results["keywords"].(*schema.Message)
                        return fmt.Sprintf("=== 文本分析报告 ===\n\n【情感分析】\n%s\n\n【关键词提取】\n%s",
                                sentiment.Content, keywords.Content), nil
                },
        )).
                AddInput("sentiment", compose.ToField("sentiment")).
                AddInput("keywords", compose.ToField("keywords"))

        wf.End().AddInput("merge")

        // 编译并运行
        runner, err := wf.Compile(ctx)
        if err != nil {
                log.Fatal("编译失败:", err)
        }

        result, err := runner.Invoke(ctx, AnalysisInput{
                Text: "这款新出的Go语言框架Eino让我眼前一亮，它把大模型应用开发中最头疼的编排问题解决得很优雅，API设计简洁又不失灵活性，字节跳动内部的实战验证也让人放心，唯一的遗憾是文档还不够丰富，社区生态还在成长期。",
                TopN: "5",
        })
        if err != nil {
                log.Fatal("运行失败:", err)
        }

        fmt.Println(result)
}
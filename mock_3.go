package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "sync"
    "time"

    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/callbacks"
    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/compose"
    "github.com/cloudwego/eino/schema"
    callbacksHelper "github.com/cloudwego/eino/utils/callbacks"
)
/*
现在把它们组合起来，构建一个接近生产环境的可观测性方案。
这个方案会做三件事:全局日志追踪(每个组件的开始/结束/耗时)、ChatModel的Token用量统计、错误告警。
*/
// TokenTracker 统计 Token 用量
type TokenTracker struct {
    mu              sync.Mutex
    totalPrompt     int
    totalCompletion int
    totalTokens     int
    callCount       int
}

func (t *TokenTracker) Record(usage *model.TokenUsage) {
    if usage == nil {
       return
    }
    t.mu.Lock()
    defer t.mu.Unlock()
    t.totalPrompt += usage.PromptTokens
    t.totalCompletion += usage.CompletionTokens
    t.totalTokens += usage.TotalTokens
    t.callCount++
}

func (t *TokenTracker) Report() {
    t.mu.Lock()
    defer t.mu.Unlock()
    fmt.Printf("\n===== Token 用量统计 =====\n")
    fmt.Printf("调用次数: %d\n", t.callCount)
    fmt.Printf("输入Token总计: %d\n", t.totalPrompt)
    fmt.Printf("输出Token总计: %d\n", t.totalCompletion)
    fmt.Printf("Token总计: %d\n", t.totalTokens)
    fmt.Printf("==========================\n")
}

func main() {
    ctx := context.Background()
    tracker := &TokenTracker{}

    // ===== 第一层：全局通用回调（日志+耗时+错误告警）=====
    globalHandler := callbacks.NewHandlerBuilder().
       OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
          log.Printf("[TRACE] ▶ %s(%s) 开始执行", info.Name, info.Component)
          return context.WithValue(ctx, "trace_start_"+info.Name, time.Now())
       }).
       OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
          if start, ok := ctx.Value("trace_start_" + info.Name).(time.Time); ok {
             duration := time.Since(start)
             log.Printf("[TRACE] ◀ %s(%s) 执行完成, 耗时: %v", info.Name, info.Component, duration)
             // 如果某个组件耗时超过 5 秒，输出警告
             if duration > 5*time.Second {
                log.Printf("[WARN] ⚠ %s 执行耗时过长: %v", info.Name, duration)
             }
          }
          return ctx
       }).
       OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
          log.Printf("[ERROR] ✘ %s(%s) 执行失败: %v", info.Name, info.Component, err)
          // 这里可以接入告警系统，比如发送钉钉/飞书通知
          return ctx
       }).
       Build()
    callbacks.AppendGlobalHandlers(globalHandler)

    // ===== 第二层：类型安全的 ChatModel 回调（Token追踪）=====
    modelHandler := &callbacksHelper.ModelCallbackHandler{
       OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *model.CallbackOutput) context.Context {
          tracker.Record(output.TokenUsage)
          if output.TokenUsage != nil {
             log.Printf("[TOKEN] %s: 输入=%d, 输出=%d",
                info.Name,
                output.TokenUsage.PromptTokens,
                output.TokenUsage.CompletionTokens)
          }
          return ctx
       },
    }

    tokenHandler := callbacksHelper.NewHandlerHelper().
       ChatModel(modelHandler).
       Handler()

    // 创建模型
    chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
       BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
       APIKey:  os.Getenv("DASHSCOPE_API_KEY"),
       Model:   "qwen-plus",
    })
    if err != nil {
       log.Fatal(err)
    }

    // 构建编排
    chain := compose.NewChain[string, *schema.Message]()
    chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, input string) ([]*schema.Message, error) {
       return []*schema.Message{
          schema.SystemMessage("你是一个Go语言专家，回答简洁。"),
          schema.UserMessage(input),
       }, nil
    }), compose.WithNodeName("消息构建"))
    chain.AppendChatModel(chatModel, compose.WithNodeName("通义千问"))

    runnable, err := chain.Compile(ctx)
    if err != nil {
       log.Fatal(err)
    }

    // 模拟多次调用
    questions := []string{
       "Go的slice和array有什么区别？",
       "解释一下Go的GMP调度模型",
       "什么是context.Context？",
    }

    for i, q := range questions {
       fmt.Printf("\n--- 第 %d 次调用 ---\n", i+1)
       result, err := runnable.Invoke(ctx, q,
          compose.WithCallbacks(tokenHandler)) // Token回调作为局部回调传入
       if err != nil {
          log.Printf("调用失败: %v", err)
          continue
       }
       // 只打印前80个字符
       content := result.Content
       if len(content) > 80 {
          content = content[:80] + "..."
       }
       fmt.Printf("回复: %s\n", content)
    }

    // 最后输出 Token 用量统计
    tracker.Report()
}
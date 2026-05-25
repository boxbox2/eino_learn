package main

import (
        "context"
        "fmt"
        "log"
        "os"

        "github.com/cloudwego/eino-ext/components/model/openai"
        "github.com/cloudwego/eino/adk"
)

func main() {
        ctx := context.Background()

        chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
                BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
                APIKey:  os.Getenv("DASHSCOPE_API_KEY"),
                Model:   "qwen-plus",
        })
        if err != nil {
                log.Fatal(err)
        }

        // ===== 第一阶段：三个并行分析师 =====

        perfAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
                Name:        "perf_reviewer",
                Description: "性能评审专家",
                Instruction: `你是性能评审专家。请从性能维度评审用户提出的技术方案，输出格式：
【性能评分】1-10分
【核心发现】列出2-3个关键点
【改进建议】如有性能风险，给出具体建议`,
                Model: chatModel,
        })

        secAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
                Name:        "sec_reviewer",
                Description: "安全评审专家",
                Instruction: `你是安全评审专家。请从安全维度评审用户提出的技术方案，输出格式：
【安全评分】1-10分
【核心发现】列出2-3个关键点
【改进建议】如有安全风险，给出具体建议`,
                Model: chatModel,
        })

        costAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
                Name:        "cost_reviewer",
                Description: "成本评审专家",
                Instruction: `你是成本评审专家。请从成本维度评审用户提出的技术方案，输出格式：
【成本评分】1-10分
【核心发现】列出2-3个关键点
【改进建议】如有成本优化空间，给出具体建议`,
                Model: chatModel,
        })

        // 并行执行三个评审
        parallelReview, _ := adk.NewParallelAgent(ctx, &adk.ParallelAgentConfig{
                Name:        "parallel_review",
                Description: "并行技术评审",
                SubAgents:   []adk.Agent{perfAgent, secAgent, costAgent},
        })

        // ===== 第二阶段：综合汇总 =====

        summarizer, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
                Name:        "summarizer",
                Description: "评审报告汇总专家",
                Instruction: `你是技术评审委员会主席。你会收到三位专家（性能、安全、成本）的独立评审意见。
请综合所有评审意见，生成一份结构化的最终评审报告，格式如下：

# 技术方案评审报告

## 综合评分
（三个维度的加权平均，权重：性能40%、安全35%、成本25%）

## 各维度概要
（简要汇总每个维度的核心发现）

## 关键风险
（列出最需要关注的风险项）

## 最终结论
（通过/有条件通过/不通过，并说明理由）

## 行动项
（按优先级列出需要落实的改进项）`,
                Model: chatModel,
        })

        // ===== 组装完整流水线 =====

        fullPipeline, _ := adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
                Name:        "tech_review_system",
                Description: "完整技术方案评审系统",
                SubAgents:   []adk.Agent{parallelReview, summarizer},
        })

        // ===== 运行 =====

        runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: fullPipeline})

        proposal := `技术方案：将公司核心交易系统从单体架构迁移到微服务架构

关键设计决策：
1. 使用 Kubernetes 作为容器编排平台
2. 服务间通信采用 gRPC + Protobuf
3. 数据库从单一 MySQL 拆分为每个服务独立的数据库（Database per Service）
4. 引入 Apache Kafka 作为异步消息队列
5. 使用 Istio 服务网格处理流量治理
6. API Gateway 使用 Kong

预计影响范围：核心交易链路、用户中心、库存管理、支付系统`

        iter := runner.Query(ctx, "请评审以下技术方案：\n\n"+proposal)

        for {
                event, ok := iter.Next()
                if !ok {
                        break
                }
                if event.Err != nil {
                        log.Fatal(event.Err)
                }
                if event.Output != nil && event.Output.MessageOutput != nil {
                        fmt.Printf("\n========== [%s] ==========\n%s\n",
                                event.AgentName,
                                event.Output.MessageOutput.Message.Content)
                }
        }
}
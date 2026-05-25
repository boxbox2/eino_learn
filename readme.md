# agent-basic

这是一个基于 [CloudWeGo Eino](https://github.com/cloudwego/eino) 的 Go 大模型应用示例仓库。项目通过几个独立 Demo 展示 Eino 的图编排、Workflow、Callback 可观测性、ADK 多 Agent 流水线，以及 MCP 工具接入能力。

## 项目内容

| 文件 | 主题 | 说明 |
| --- | --- | --- |
| `mock_1.go` | Graph 路由编排 | 根据用户技术问题判断前端、后端或通用类型，路由到不同 Prompt 模板，再调用模型生成回答。 |
| `mock_2.go` | 并行 Workflow | 同时做情感分析和关键词提取，最后合并成一份文本分析报告。 |
| `mock_3.go` | Callback 观测 | 演示全局链路日志、耗时统计、错误告警，以及 ChatModel Token 用量统计。 |
| `mock_4.go` | ADK 多 Agent 评审 | 构建性能、安全、成本三个并行评审 Agent，再由总结 Agent 输出最终技术评审报告。 |
| `mock_5.go` | ADK 顺序流水线 | 先由 analyzer 分析需求，再由 executor 给出执行方案，并通过 `AsyncIterator[*AgentEvent]` 消费执行事件。 |
| `mcp_mock/server` | MCP Server | 通过 SSE 暴露字符串处理和时间处理工具。 |
| `mcp_mock/agent` | MCP Agent Client | 使用 Eino ReAct Agent 连接 MCP Server，让模型可以调用外部工具。 |

## 环境要求

- Go 1.25 或更新版本
- 可用的大模型 API Key
- 可选：LangSmith API Key，用于 MCP Agent 示例中的链路追踪

示例中同时出现了火山引擎 Ark 和 DashScope/Qwen 的 OpenAI 兼容接口配置。

## 环境变量

运行根目录下的示例时，可以在项目根目录创建 `.env` 文件：

```env
ARK_API_KEY=你的_ark_api_key
Model=你的_ark_模型名称
DASHSCOPE_API_KEY=你的_dashscope_api_key
LANG_SMITH_KEY=你的_langsmith_key
```

运行 `mcp_mock/agent` 时，可以在 `mcp_mock` 目录下创建 `.env` 文件，也可以直接在系统环境变量中配置这些值。

注意：`.env` 文件通常包含密钥，不应该提交到 GitHub。

## 安装依赖

```bash
go mod tidy
```

## 运行示例

这个仓库中的多个 Demo 文件都属于同一个 `main` package，其中有些文件定义了自己的 `main` 函数。因此建议一次只运行一个示例文件：

```bash
go run mock_2.go
go run mock_3.go
go run mock_4.go
```

部分文件使用了 `main_1`、`main_5` 这类辅助入口函数。如果要单独运行对应示例，可以先把函数名临时改成 `main`。

## MCP 示例

先启动本地 MCP Server：

```bash
cd mcp_mock/server
go run .
```

再打开另一个终端启动 Agent：

```bash
cd mcp_mock/agent
go run .
```

MCP Server 暴露了两个工具：

- `string_transform`：字符串转大写、转小写、统计字符数、反转字符串
- `time_util`：获取当前时间、计算两个日期之间相差的天数

Agent 会连接 `http://localhost:8080/sse`，自动发现 MCP 工具，并在回答用户问题时由 ReAct Agent 决定是否调用工具。

## ADK 事件迭代

ADK 的 `Runner.Query` 返回的不是单个最终字符串，而是一个 `AsyncIterator[*AgentEvent]`。每次调用 `Next()` 都会从 Agent 执行流中取出一个事件：

```go
for {
    event, ok := iter.Next()
    if !ok {
        break
    }

    if event.Err != nil {
        log.Fatal(event.Err)
    }

    if event.Output != nil && event.Output.MessageOutput != nil {
        fmt.Println(event.Output.MessageOutput.Message.Content)
    }
}
```

之所以需要 `for` 循环，是因为一次 Agent 执行可能产生多个事件，例如中间 Agent 输出、最终输出、工具调用结果、中断事件或错误事件。循环消费完整个迭代器，才能拿到完整执行过程。

## 注意事项

- 不要提交真实的 `.env` 文件或 API Key。
- 如果代码中的中文注释出现乱码，通常是文件编码不一致导致的，建议统一保存为 UTF-8。
- 当前仓库更偏向独立 Demo 集合，而不是一个完整应用；如果直接执行 `go test ./...`，可能需要先整理各示例文件的入口函数和包结构。

# agent-basic

Go examples for building LLM applications with [CloudWeGo Eino](https://github.com/cloudwego/eino). This repository collects several small demos that cover graph orchestration, workflow composition, callback observability, ADK multi-agent pipelines, and MCP tool integration.

## What Is Included

| File | Topic | Description |
| --- | --- | --- |
| `mock_1.go` | Graph routing | Routes a technical question to different prompt templates, then calls a chat model and formats the answer. |
| `mock_2.go` | Parallel workflow | Runs sentiment analysis and keyword extraction in parallel, then merges the results into a report. |
| `mock_3.go` | Callbacks | Demonstrates global tracing callbacks and model-level token usage tracking. |
| `mock_4.go` | ADK multi-agent review | Builds a technical review system with parallel reviewers and a summarizer agent. |
| `mock_5.go` | ADK sequential pipeline | Runs an analyzer agent followed by an executor agent and consumes `AsyncIterator[*AgentEvent]` events. |
| `mcp_mock/server` | MCP server | Exposes string and time tools through an SSE MCP server. |
| `mcp_mock/agent` | MCP client agent | Connects Eino's ReAct agent to the MCP server and lets the model call those tools. |

## Requirements

- Go 1.25 or newer
- A model provider API key
- Optional: LangSmith API key for tracing in the MCP agent example

The examples use both Volcano Engine Ark and OpenAI-compatible DashScope/Qwen configurations.

## Environment Variables

Create a `.env` file in the project root when running the root examples:

```env
ARK_API_KEY=your_ark_api_key
Model=your_ark_model_name
DASHSCOPE_API_KEY=your_dashscope_api_key
LANG_SMITH_KEY=your_langsmith_key
```

For `mcp_mock/agent`, create a `.env` file in `mcp_mock` or make sure the same variables are available in your shell.

## Install Dependencies

```bash
go mod tidy
```

## Run Examples

This repository contains multiple demo files in the same `main` package. Several files define their own `main` function, so run a single example file at a time.

```bash
go run mock_2.go
go run mock_3.go
go run mock_4.go
```

Some files use helper entry functions such as `main_1` or `main_5`; rename the function to `main` before running that specific file.

## MCP Demo

Start the local MCP server first:

```bash
cd mcp_mock/server
go run .
```

Then start the agent in another terminal:

```bash
cd mcp_mock/agent
go run .
```

The MCP server exposes two tools:

- `string_transform`: uppercase, lowercase, character count, and reverse
- `time_util`: current time and date difference calculation

The agent connects to `http://localhost:8080/sse`, discovers the tools, and lets the ReAct agent call them when answering user questions.

## ADK Event Iteration

ADK runners return an `AsyncIterator[*AgentEvent]` instead of a single final string. Each call to `Next()` returns one event from the agent execution stream:

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

This is useful because one run may produce multiple events: intermediate agent outputs, final responses, tool actions, interruptions, or errors.

## Notes

- Do not commit real `.env` files or API keys.
- Some demo comments and strings may appear garbled if the source file was saved with a mismatched encoding. Save files as UTF-8 to keep Chinese text readable.
- `go test ./...` may need cleanup before it passes because the repository is organized as independent runnable demos rather than a single compiled application.

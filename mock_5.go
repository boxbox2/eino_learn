package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/joho/godotenv"
)
/*
构建一个贴近真实场景的审批工作流系统。场景是这样的:用户提交一个采购申请，系统先自动评估风险，
然后根据金额决定是否需要人工审批--小额直接通过，大额中断等待审批。整个过程有日志追踪、状态共享和中断恢复。
*/
type memCheckPointStore struct {
	m map[string][]byte
}

func (s *memCheckPointStore) Get(_ context.Context, id string) ([]byte, bool, error) {
	v, ok := s.m[id]
	return v, ok, nil
}

func (s *memCheckPointStore) Set(_ context.Context, id string, data []byte) error {
	s.m[id] = data
	return  nil 
}

type AssesParams struct {
	Amount      float64 `json:"amount" jsonschema:"description=采购金额（元）"`
	Description string  `json:"description" jsonschema: "description=采购描述"`
}

func assessRisk(ctx context.Context, params *AssesParams) (string, error) {
	riskLevel := "低风险"
	needApproval := false
	if params.Amount > 10000 {
		riskLevel = "高风险"
		needApproval = true
	} else if params.Amount > 5000 {
		riskLevel = "中风险"
		needApproval = true
	}
	adk.AddSessionValues(ctx, map[string]any{
		"amount":        params.Amount,
		"risk_level":    riskLevel,
		"need_approval": needApproval,
		"description":   params.Description,
	})
	return fmt.Sprintf("风险评估完成 | 金额: %.0f元 | 风险等级: %s | 需要审批: %v",
		params.Amount, riskLevel, needApproval), nil
}

type ApprovalParams struct {
	Action string `json:"action" jsonschema:"description=执行的审批动作: submit"`
}

func submitApproval(ctx context.Context, params *ApprovalParams) (string, error) {
	needApproval, _ := adk.GetSessionValue(ctx, "need_approval")
	amount, _ := adk.GetSessionValue(ctx, "amount")

	if need, ok := needApproval.(bool); ok && need {
		wasInterrupted, _, _ := compose.GetInterruptState[string](ctx)
		if !wasInterrupted {
			return "", compose.Interrupt(ctx,
				fmt.Sprintf("采购申请需要审批 | 金额: %.0f元 | 请审批人确认（approve/reject）", amount))
		}
		isTarget, hasData, decision := compose.GetResumeContext[string](ctx)
		if isTarget && hasData {
			if decision == "approve" {
				return "审批通过，采购申请已提交", nil
			}
			return "审批被拒绝，采购申请已取消", nil
		}
		return "", compose.Interrupt(ctx,
			fmt.Sprintf("采购申请需要审批 | 金额: %.0f元 | 请审批人确认（approve/reject）", amount))

	}

	return fmt.Sprintf("金额 %.0f 元低于审批阈值，自动通过", amount), nil
}

func main() {
	ctx := context.Background()
	if err := godotenv.Load(); err !=nil{
		log.Fatal(err)
	}
	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey: os.Getenv("ARK_API_KEY"),
		Model:  os.Getenv("Model"),
	})
	if err != nil {
		log.Fatal(err)
	}
	handler := callbacks.NewHandlerBuilder().OnStartFn(
		func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			if info.Component == adk.ComponentOfAgent {
				fmt.Printf("[%s] ▶ %s 开始\n", time.Now().Format("15:04:05"), info.Name)
			}
			return ctx
		}).OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
		if info.Component == adk.ComponentOfAgent {
			fmt.Printf("[%s] ■ %s 完成\n", time.Now().Format("15:04:05"), info.Name)
		}
		return ctx
	}).Build()

	assessTool, err := utils.InferTool("assess_risk", "评估采购申请的风险等级", assessRisk)
	if err != nil {
		log.Fatal(err)
	}
	approvalTool, err := utils.InferTool("submit_approval", "提交采购审批，大额采购需要人工审批", submitApproval)
	if err != nil {
		log.Fatal(err)
	}
	assessor, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "risk_assessor",
		Description: "评估采购申请的风险",
		Instruction: "你是风险评估师。收到采购申请后，使用 assess_risk 工具评估风险。输出评估结果即可。",
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{assessTool},
			},
		},
	})
	// Agent 2：审批处理器
	approver, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "approval_handler",
		Description: "处理采购审批流程",
		Instruction: "你是审批处理器。使用 submit_approval 工具提交采购审批。如果审批通过，输出确认信息；如果被拒绝，输出拒绝原因。",
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{approvalTool},
			},
		},
	})
	pipeline, _ := adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        "approval_pipeline",
		Description: "采购审批流水线",
		SubAgents:   []adk.Agent{assessor, approver},
	})
	store := &memCheckPointStore{m: make(map[string][]byte)}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           pipeline,
		CheckPointStore: store,
	})
	checkPointID := "purchase-001"
	fmt.Println("========== 提交采购申请 ==========")
	iter := runner.Query(ctx, "我需要采购一批服务器，预算 25000 元，用于部署新的AI推理服务",
		adk.WithCheckPointID(checkPointID),
		adk.WithCallbacks(handler),
	)
	var interruptID string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Fatal(event.Err)
		}
		if event.Action != nil && event.Action.Interrupted != nil {
			interruptID = event.Action.Interrupted.InterruptContexts[0].ID
			fmt.Printf("\n⏸️  流程中断: %v\n", event.Action.Interrupted.InterruptContexts[0].Info)
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			fmt.Printf("  [%s] %s\n", event.AgentName, event.Output.MessageOutput.Message.Content)
		}
	}
	if interruptID != "" {
		fmt.Println("\n========== 审批人确认 ==========")
		fmt.Println("审批决定: approve")
		resumeIter, err := runner.ResumeWithParams(ctx, checkPointID, &adk.ResumeParams{
			Targets: map[string]any{
				interruptID: "approve",
			},
		})
		if err != nil {
			log.Fatal(err)
		}
		       for {
          event, ok := resumeIter.Next()
          if !ok {
             break
          }
          if event.Err != nil {
             log.Fatal(event.Err)
          }
          if event.Output != nil && event.Output.MessageOutput != nil {
             fmt.Printf("  [%s] %s\n", event.AgentName, event.Output.MessageOutput.Message.Content)
          }
       }
	}
}

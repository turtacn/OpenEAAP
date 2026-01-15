// internal/api/cli/commands/agent_cmd.go
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/openeeap/openeeap/internal/app/dto"
	"github.com/openeeap/openeeap/internal/app/service"
	"github.com/openeeap/openeeap/pkg/errors"
)

// AgentCommand 创建 agent 命令
func AgentCommand(agentService service.AgentService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage agents",
		Long:  "Create, list, update, delete and test agents in OpenEAAP",
	}

	cmd.AddCommand(
		createAgentCommand(agentService),
		listAgentsCommand(agentService),
		getAgentCommand(agentService),
		updateAgentCommand(agentService),
		deleteAgentCommand(agentService),
		testAgentCommand(agentService),
	)

	return cmd
}

// createAgentCommand 创建 agent
func createAgentCommand(agentService service.AgentService) *cobra.Command {
	var (
		name        string
		description string
		runtimeType string
		configFile  string
		output      string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new agent",
		Long:  "Create a new agent with specified configuration",
		Example: `  # Create agent from config file
 openeeap agent create --name "my-agent" --file agent.yaml

 # Create agent with inline config
 openeeap agent create --name "my-agent" --runtime native --description "My first agent"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// 构建创建请求
			req := &dto.CreateAgentRequest{
				Name:        name,
				Description: description,
				RuntimeType: runtimeType,
			}

			// 从文件加载配置
			if configFile != "" {
				configData, err := os.ReadFile(configFile)
				if err != nil {
					return errors.Wrap(err, "failed to read config file")
				}

				var fileConfig dto.CreateAgentRequest
				if err := yaml.Unmarshal(configData, &fileConfig); err != nil {
					return errors.Wrap(err, "failed to parse config file")
				}

				// 合并配置（命令行参数优先）
				if req.Name == "" {
					req.Name = fileConfig.Name
				}
				if req.Description == "" {
					req.Description = fileConfig.Description
				}
				if req.RuntimeType == "" {
					req.RuntimeType = fileConfig.RuntimeType
				}
				req.Config = fileConfig.Config
				req.Tools = fileConfig.Tools
				req.Memory = fileConfig.Memory
				req.Constraints = fileConfig.Constraints
			}

			// 验证必填字段
			if req.Name == "" {
				return errors.NewBadRequest("agent name is required")
			}

			// 调用服务创建 agent
			resp, err := agentService.CreateAgent(ctx, req)
			if err != nil {
				return errors.Wrap(err, "failed to create agent")
			}

			// 格式化输出
			return printOutput(output, resp, func() {
				fmt.Printf("✅ Agent created successfully\n\n")
				fmt.Printf("ID:          %s\n", resp.ID)
				fmt.Printf("Name:        %s\n", resp.Name)
				fmt.Printf("Runtime:     %s\n", resp.RuntimeType)
				fmt.Printf("Status:      %s\n", resp.Status)
				fmt.Printf("Created At:  %s\n", resp.CreatedAt.Format(time.RFC3339))
			})
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Agent name (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Agent description")
	cmd.Flags().StringVarP(&runtimeType, "runtime", "r", "native", "Runtime type (native|langchain|autogpt)")
	cmd.Flags().StringVarP(&configFile, "file", "f", "", "Config file path (YAML)")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format (text|json|yaml)")

	return cmd
}

// listAgentsCommand 列出所有 agents
func listAgentsCommand(agentService service.AgentService) *cobra.Command {
	var (
		page     int
		pageSize int
		status   string
		runtime  string
		output   string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all agents",
		Long:  "List all agents with optional filtering",
		Example: `  # List all agents
 openeeap agent list

 # List active agents
 openeeap agent list --status active

 # List agents with specific runtime
 openeeap agent list --runtime native --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// 构建查询请求
			req := &dto.ListAgentsRequest{
				Page:     page,
				PageSize: pageSize,
				Status:   status,
				Runtime:  runtime,
			}

			// 调用服务查询
			resp, err := agentService.ListAgents(ctx, req)
			if err != nil {
				return errors.Wrap(err, "failed to list agents")
			}

			// 格式化输出
			return printOutput(output, resp, func() {
				if len(resp.Items) == 0 {
					fmt.Println("No agents found")
					return
				}

				// 使用 tabwriter 格式化表格
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
				fmt.Fprintln(w, "ID\tNAME\tRUNTIME\tSTATUS\tCREATED")
				fmt.Fprintln(w, "----\t----\t-------\t------\t-------")

				for _, agent := range resp.Items {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
						truncateString(agent.ID, 8),
						truncateString(agent.Name, 30),
						agent.RuntimeType,
						agent.Status,
						agent.CreatedAt.Format("2006-01-02"),
					)
				}

				w.Flush()
				fmt.Printf("\nTotal: %d agents (Page %d/%d)\n",
					resp.Total, resp.Page, (resp.Total+resp.PageSize-1)/resp.PageSize)
			})
		},
	}

	cmd.Flags().IntVarP(&page, "page", "p", 1, "Page number")
	cmd.Flags().IntVarP(&pageSize, "size", "s", 10, "Page size")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (active|inactive|error)")
	cmd.Flags().StringVar(&runtime, "runtime", "", "Filter by runtime type")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format (text|json|yaml)")

	return cmd
}

// getAgentCommand 获取单个 agent 详情
func getAgentCommand(agentService service.AgentService) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "get [agent-id]",
		Short: "Get agent details",
		Long:  "Get detailed information about a specific agent",
		Example: `  # Get agent by ID
 openeeap agent get abc123

 # Get agent with JSON output
 openeeap agent get abc123 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			agentID := args[0]

			// 调用服务查询
			resp, err := agentService.GetAgent(ctx, agentID)
			if err != nil {
				return errors.Wrap(err, "failed to get agent")
			}

			// 格式化输出
			return printOutput(output, resp, func() {
				fmt.Printf("Agent Details\n")
				fmt.Printf("=============\n\n")
				fmt.Printf("ID:              %s\n", resp.ID)
				fmt.Printf("Name:            %s\n", resp.Name)
				fmt.Printf("Description:     %s\n", resp.Description)
				fmt.Printf("Runtime Type:    %s\n", resp.RuntimeType)
				fmt.Printf("Status:          %s\n", resp.Status)
				fmt.Printf("Created At:      %s\n", resp.CreatedAt.Format(time.RFC3339))
				fmt.Printf("Updated At:      %s\n", resp.UpdatedAt.Format(time.RFC3339))

				if len(resp.Tools) > 0 {
					fmt.Printf("\nTools:\n")
					for _, tool := range resp.Tools {
						fmt.Printf("  - %s\n", tool)
					}
				}

				if resp.Config != nil {
					fmt.Printf("\nConfiguration:\n")
					configYAML, _ := yaml.Marshal(resp.Config)
					fmt.Printf("%s\n", string(configYAML))
				}
			})
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format (text|json|yaml)")

	return cmd
}

// updateAgentCommand 更新 agent
func updateAgentCommand(agentService service.AgentService) *cobra.Command {
	var (
		name        string
		description string
		configFile  string
		output      string
	)

	cmd := &cobra.Command{
		Use:   "update [agent-id]",
		Short: "Update an existing agent",
		Long:  "Update agent configuration and metadata",
		Example: `  # Update agent name
 openeeap agent update abc123 --name "new-name"

 # Update from config file
 openeeap agent update abc123 --file agent.yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			agentID := args[0]

			// 构建更新请求
			req := &dto.UpdateAgentRequest{
				Name:        name,
				Description: description,
			}

			// 从文件加载配置
			if configFile != "" {
				configData, err := os.ReadFile(configFile)
				if err != nil {
					return errors.Wrap(err, "failed to read config file")
				}

				var fileConfig dto.UpdateAgentRequest
				if err := yaml.Unmarshal(configData, &fileConfig); err != nil {
					return errors.Wrap(err, "failed to parse config file")
				}

				// 合并配置
				if req.Name == "" {
					req.Name = fileConfig.Name
				}
				if req.Description == "" {
					req.Description = fileConfig.Description
				}
				req.Config = fileConfig.Config
				req.Tools = fileConfig.Tools
			}

			// 调用服务更新
			resp, err := agentService.UpdateAgent(ctx, agentID, req)
			if err != nil {
				return errors.Wrap(err, "failed to update agent")
			}

			// 格式化输出
			return printOutput(output, resp, func() {
				fmt.Printf("✅ Agent updated successfully\n\n")
				fmt.Printf("ID:          %s\n", resp.ID)
				fmt.Printf("Name:        %s\n", resp.Name)
				fmt.Printf("Updated At:  %s\n", resp.UpdatedAt.Format(time.RFC3339))
			})
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "New agent name")
	cmd.Flags().StringVarP(&description, "description", "d", "", "New description")
	cmd.Flags().StringVarP(&configFile, "file", "f", "", "Config file path (YAML)")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format (text|json|yaml)")

	return cmd
}

// deleteAgentCommand 删除 agent
func deleteAgentCommand(agentService service.AgentService) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete [agent-id]",
		Short: "Delete an agent",
		Long:  "Delete an agent permanently",
		Example: `  # Delete agent (with confirmation)
 openeeap agent delete abc123

 # Force delete without confirmation
 openeeap agent delete abc123 --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			agentID := args[0]

			// 确认删除
			if !force {
				fmt.Printf("⚠️  Are you sure you want to delete agent '%s'? [y/N]: ", agentID)
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Cancelled")
					return nil
				}
			}

			// 调用服务删除
			if err := agentService.DeleteAgent(ctx, agentID); err != nil {
				return errors.Wrap(err, "failed to delete agent")
			}

			fmt.Printf("✅ Agent '%s' deleted successfully\n", agentID)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force delete without confirmation")

	return cmd
}

// testAgentCommand 测试 agent 执行
func testAgentCommand(agentService service.AgentService) *cobra.Command {
	var (
		input  string
		stream bool
		output string
	)

	cmd := &cobra.Command{
		Use:   "test [agent-id]",
		Short: "Test agent execution",
		Long:  "Send a test request to an agent and display the response",
		Example: `  # Test agent with input
 openeeap agent test abc123 --input "Hello, world!"

 # Test with streaming response
 openeeap agent test abc123 --input "Analyze this text" --stream`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			agentID := args[0]

			// 构建执行请求
			req := &dto.ExecuteAgentRequest{
				Input: input,
			}

			fmt.Printf("🚀 Testing agent '%s'...\n\n", agentID)

			if stream {
				// 流式执行
				respChan, errChan := agentService.ExecuteAgentStream(ctx, agentID, req)

				for {
					select {
					case chunk, ok := <-respChan:
						if !ok {
							fmt.Println("\n\n✅ Execution completed")
							return nil
						}
						fmt.Print(chunk.Content)

					case err := <-errChan:
						if err != nil {
							return errors.Wrap(err, "execution failed")
						}
						return nil

					case <-ctx.Done():
						return errors.NewInternal("execution cancelled")
					}
				}
			} else {
				// 同步执行
				resp, err := agentService.ExecuteAgent(ctx, agentID, req)
				if err != nil {
					return errors.Wrap(err, "execution failed")
				}

				// 格式化输出
				return printOutput(output, resp, func() {
					fmt.Printf("Response:\n%s\n\n", resp.Output)
					fmt.Printf("Status:      %s\n", resp.Status)
					fmt.Printf("Duration:    %dms\n", resp.Duration)
					if resp.TokensUsed > 0 {
						fmt.Printf("Tokens Used: %d\n", resp.TokensUsed)
					}
				})
			}
		},
	}

	cmd.Flags().StringVarP(&input, "input", "i", "", "Input text for agent (required)")
	cmd.Flags().BoolVarP(&stream, "stream", "s", false, "Stream response")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format (text|json|yaml)")
	cmd.MarkFlagRequired("input")

	return cmd
}

// printOutput 统一输出格式化
func printOutput(format string, data interface{}, textFunc func()) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(data)

	case "yaml":
		encoder := yaml.NewEncoder(os.Stdout)
		encoder.SetIndent(2)
		return encoder.Encode(data)

	case "text":
		textFunc()
		return nil

	default:
		return errors.NewBadRequest("unsupported output format: %s", format)
	}
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

//Personal.AI order the ending

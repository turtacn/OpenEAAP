// internal/api/cli/commands/model_cmd.go
package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/openeeap/openeeap/internal/app/dto"
	"github.com/openeeap/openeeap/internal/app/service"
	"github.com/openeeap/openeeap/pkg/errors"
	"github.com/spf13/cobra"
)

// ModelCommand 定义 model 根命令
type ModelCommand struct {
	modelService service.ModelService
}

// NewModelCommand 创建 model 命令
func NewModelCommand(modelService service.ModelService) *cobra.Command {
	mc := &ModelCommand{
		modelService: modelService,
	}

	cmd := &cobra.Command{
		Use:   "model",
		Short: "Manage AI models",
		Long:  `Manage AI models including registration, deployment, monitoring, and deletion`,
	}

	// 添加子命令
	cmd.AddCommand(mc.listCmd())
	cmd.AddCommand(mc.getCmd())
	cmd.AddCommand(mc.registerCmd())
	cmd.AddCommand(mc.deployCmd())
	cmd.AddCommand(mc.undeployCmd())
	cmd.AddCommand(mc.monitorCmd())
	cmd.AddCommand(mc.deleteCmd())
	cmd.AddCommand(mc.testCmd())

	return cmd
}

// listCmd 列出所有模型
func (mc *ModelCommand) listCmd() *cobra.Command {
	var (
		modelType  string
		status     string
		pageSize   int
		pageNumber int
		format     string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all models",
		Long:  `List all registered models with optional filtering`,
		Example: `  # List all models
 openeeap model list

 # List only LLM models
 openeeap model list --type llm

 # List deployed models
 openeeap model list --status deployed

 # Output in JSON format
 openeeap model list --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			req := &dto.ListModelsRequest{
				Type:       modelType,
				Status:     status,
				PageSize:   pageSize,
				PageNumber: pageNumber,
			}

			resp, err := mc.modelService.ListModels(ctx, req)
			if err != nil {
				return errors.Wrap(err, "failed to list models")
			}

			return mc.printModelList(resp, format)
		},
	}

	cmd.Flags().StringVar(&modelType, "type", "", "Filter by model type (llm, embedding, reranker)")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (registered, deploying, deployed, failed)")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "Number of items per page")
	cmd.Flags().IntVar(&pageNumber, "page", 1, "Page number")
	cmd.Flags().StringVarP(&format, "format", "f", "table", "Output format (table, json, yaml)")

	return cmd
}

// getCmd 获取模型详情
func (mc *ModelCommand) getCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "get [model-id]",
		Short: "Get model details",
		Long:  `Get detailed information about a specific model`,
		Args:  cobra.ExactArgs(1),
		Example: `  # Get model details
 openeeap model get model-123

 # Output in JSON format
 openeeap model get model-123 --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			modelID := args[0]
			model, err := mc.modelService.GetModel(ctx, modelID)
			if err != nil {
				return errors.Wrap(err, "failed to get model")
			}

			return mc.printModelDetail(model, format)
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "yaml", "Output format (yaml, json)")

	return cmd
}

// registerCmd 注册新模型
func (mc *ModelCommand) registerCmd() *cobra.Command {
	var (
		name        string
		modelType   string
		version     string
		endpoint    string
		apiKey      string
		configFile  string
		description string
	)

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new model",
		Long:  `Register a new AI model to the platform`,
		Example: `  # Register a model with basic info
 openeeap model register \
   --name "GPT-4" \
   --type llm \
   --version "gpt-4-0125-preview" \
   --endpoint "https://api.openai.com/v1"

 # Register with config file
 openeeap model register --config model-config.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			req := &dto.RegisterModelRequest{
				Name:        name,
				Type:        modelType,
				Version:     version,
				Endpoint:    endpoint,
				APIKey:      apiKey,
				Description: description,
			}

			// 如果提供了配置文件，从文件加载配置
			if configFile != "" {
				if err := loadConfigFromFile(configFile, req); err != nil {
					return errors.Wrap(err, "failed to load config from file")
				}
			}

			resp, err := mc.modelService.RegisterModel(ctx, req)
			if err != nil {
				return errors.Wrap(err, "failed to register model")
			}

			fmt.Printf("✅ Model registered successfully\n")
			fmt.Printf("   ID: %s\n", resp.ID)
			fmt.Printf("   Name: %s\n", resp.Name)
			fmt.Printf("   Type: %s\n", resp.Type)
			fmt.Printf("   Status: %s\n", resp.Status)

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Model name (required)")
	cmd.Flags().StringVar(&modelType, "type", "", "Model type: llm, embedding, reranker (required)")
	cmd.Flags().StringVar(&version, "version", "", "Model version (required)")
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "Model API endpoint (required)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key for authentication")
	cmd.Flags().StringVar(&configFile, "config", "", "Path to model config file (YAML)")
	cmd.Flags().StringVar(&description, "description", "", "Model description")

	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("type")

	return cmd
}

// deployCmd 部署模型
func (mc *ModelCommand) deployCmd() *cobra.Command {
	var (
		replicas  int
		gpuType   string
		gpuCount  int
		maxBatch  int
		maxTokens int
		waitReady bool
	)

	cmd := &cobra.Command{
		Use:   "deploy [model-id]",
		Short: "Deploy a model",
		Long:  `Deploy a registered model to inference service`,
		Args:  cobra.ExactArgs(1),
		Example: `  # Deploy with default settings
 openeeap model deploy model-123

 # Deploy with custom resources
 openeeap model deploy model-123 \
   --replicas 3 \
   --gpu-type A100 \
   --gpu-count 2 \
   --max-batch 64

 # Deploy and wait until ready
 openeeap model deploy model-123 --wait`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			modelID := args[0]
			req := &dto.DeployModelRequest{
				ModelID:   modelID,
				Replicas:  replicas,
				GPUType:   gpuType,
				GPUCount:  gpuCount,
				MaxBatch:  maxBatch,
				MaxTokens: maxTokens,
			}

			fmt.Printf("🚀 Deploying model %s...\n", modelID)
			resp, err := mc.modelService.DeployModel(ctx, req)
			if err != nil {
				return errors.Wrap(err, "failed to deploy model")
			}

			fmt.Printf("✅ Model deployment initiated\n")
			fmt.Printf("   Deployment ID: %s\n", resp.DeploymentID)
			fmt.Printf("   Status: %s\n", resp.Status)

			if waitReady {
				return mc.waitForDeployment(ctx, modelID)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&replicas, "replicas", 1, "Number of replicas")
	cmd.Flags().StringVar(&gpuType, "gpu-type", "", "GPU type (A100, V100, T4)")
	cmd.Flags().IntVar(&gpuCount, "gpu-count", 1, "Number of GPUs per replica")
	cmd.Flags().IntVar(&maxBatch, "max-batch", 32, "Maximum batch size")
	cmd.Flags().IntVar(&maxTokens, "max-tokens", 2048, "Maximum sequence length")
	cmd.Flags().BoolVar(&waitReady, "wait", false, "Wait until deployment is ready")

	return cmd
}

// undeployCmd 取消部署模型
func (mc *ModelCommand) undeployCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "undeploy [model-id]",
		Short: "Undeploy a model",
		Long:  `Stop a deployed model and release resources`,
		Args:  cobra.ExactArgs(1),
		Example: `  # Undeploy gracefully
 openeeap model undeploy model-123

 # Force undeploy immediately
 openeeap model undeploy model-123 --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			modelID := args[0]
			req := &dto.UndeployModelRequest{
				ModelID: modelID,
				Force:   force,
			}

			fmt.Printf("⏹️  Undeploying model %s...\n", modelID)
			if err := mc.modelService.UndeployModel(ctx, req); err != nil {
				return errors.Wrap(err, "failed to undeploy model")
			}

			fmt.Printf("✅ Model undeployed successfully\n")
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force undeploy without graceful shutdown")

	return cmd
}

// monitorCmd 监控模型状态和指标
func (mc *ModelCommand) monitorCmd() *cobra.Command {
	var (
		watch    bool
		interval int
	)

	cmd := &cobra.Command{
		Use:   "monitor [model-id]",
		Short: "Monitor model metrics",
		Long:  `Display real-time metrics and status of a deployed model`,
		Args:  cobra.ExactArgs(1),
		Example: `  # Show current metrics
 openeeap model monitor model-123

 # Watch metrics in real-time
 openeeap model monitor model-123 --watch

 # Custom refresh interval (seconds)
 openeeap model monitor model-123 --watch --interval 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			modelID := args[0]

			if watch {
				return mc.watchMetrics(ctx, modelID, interval)
			}

			return mc.showMetrics(ctx, modelID)
		},
	}

	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch metrics in real-time")
	cmd.Flags().IntVar(&interval, "interval", 3, "Refresh interval in seconds (with --watch)")

	return cmd
}

// deleteCmd 删除模型
func (mc *ModelCommand) deleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete [model-id]",
		Short: "Delete a model",
		Long:  `Delete a model from the platform (must be undeployed first)`,
		Args:  cobra.ExactArgs(1),
		Example: `  # Delete model (requires confirmation)
 openeeap model delete model-123

 # Force delete without confirmation
 openeeap model delete model-123 --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			modelID := args[0]

			// 确认删除
			if !force {
				fmt.Printf("⚠️  Are you sure you want to delete model %s? (yes/no): ", modelID)
				var confirm string
				fmt.Scanln(&confirm)
				if strings.ToLower(confirm) != "yes" {
					fmt.Println("❌ Deletion cancelled")
					return nil
				}
			}

			if err := mc.modelService.DeleteModel(ctx, modelID); err != nil {
				return errors.Wrap(err, "failed to delete model")
			}

			fmt.Printf("✅ Model deleted successfully\n")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force delete without confirmation")

	return cmd
}

// testCmd 测试模型推理
func (mc *ModelCommand) testCmd() *cobra.Command {
	var (
		prompt      string
		maxTokens   int
		temperature float64
		stream      bool
	)

	cmd := &cobra.Command{
		Use:   "test [model-id]",
		Short: "Test model inference",
		Long:  `Send a test prompt to a deployed model`,
		Args:  cobra.ExactArgs(1),
		Example: `  # Test with a simple prompt
 openeeap model test model-123 --prompt "Hello, world!"

 # Test with custom parameters
 openeeap model test model-123 \
   --prompt "Explain quantum computing" \
   --max-tokens 500 \
   --temperature 0.7

 # Test with streaming
 openeeap model test model-123 --prompt "Write a poem" --stream`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			modelID := args[0]
			req := &dto.InferenceRequest{
				ModelID:     modelID,
				Prompt:      prompt,
				MaxTokens:   maxTokens,
				Temperature: temperature,
				Stream:      stream,
			}

			fmt.Printf("🧪 Testing model %s...\n\n", modelID)

			if stream {
				return mc.testStreamInference(ctx, req)
			}

			return mc.testInference(ctx, req)
		},
	}

	cmd.Flags().StringVarP(&prompt, "prompt", "p", "", "Test prompt (required)")
	cmd.Flags().IntVar(&maxTokens, "max-tokens", 256, "Maximum tokens to generate")
	cmd.Flags().Float64Var(&temperature, "temperature", 0.7, "Sampling temperature")
	cmd.Flags().BoolVar(&stream, "stream", false, "Enable streaming output")

	cmd.MarkFlagRequired("prompt")

	return cmd
}

// Helper functions

// printModelList 打印模型列表
func (mc *ModelCommand) printModelList(resp *dto.ListModelsResponse, format string) error {
	if format == "json" {
		return printJSON(resp)
	}

	if format == "yaml" {
		return printYAML(resp)
	}

	// Table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tTYPE\tVERSION\tSTATUS\tREPLICAS\tCREATED")
	fmt.Fprintln(w, "--\t----\t----\t-------\t------\t--------\t-------")

	for _, model := range resp.Models {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
			model.ID,
			model.Name,
			model.Type,
			model.Version,
			colorizeStatus(model.Status),
			model.Replicas,
			model.CreatedAt.Format("2006-01-02"),
		)
	}

	w.Flush()

	fmt.Printf("\nTotal: %d models (Page %d/%d)\n",
		resp.Total,
		resp.PageNumber,
		(resp.Total+resp.PageSize-1)/resp.PageSize,
	)

	return nil
}

// printModelDetail 打印模型详情
func (mc *ModelCommand) printModelDetail(model *dto.ModelResponse, format string) error {
	if format == "json" {
		return printJSON(model)
	}

	return printYAML(model)
}

// waitForDeployment 等待部署完成
func (mc *ModelCommand) waitForDeployment(ctx context.Context, modelID string) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0

	for {
		select {
		case <-ctx.Done():
			return errors.New("deployment timeout")
		case <-ticker.C:
			model, err := mc.modelService.GetModel(ctx, modelID)
			if err != nil {
				return err
			}

			fmt.Printf("\r%s Waiting for deployment... Status: %s",
				spinner[i%len(spinner)],
				model.Status,
			)
			i++

			if model.Status == "deployed" {
				fmt.Printf("\n✅ Model deployed successfully\n")
				return nil
			}

			if model.Status == "failed" {
				fmt.Printf("\n❌ Deployment failed\n")
				return errors.New("deployment failed")
			}
		}
	}
}

// showMetrics 显示模型指标
func (mc *ModelCommand) showMetrics(ctx context.Context, modelID string) error {
	metrics, err := mc.modelService.GetModelMetrics(ctx, modelID)
	if err != nil {
		return errors.Wrap(err, "failed to get metrics")
	}

	fmt.Printf("📊 Model Metrics - %s\n\n", modelID)
	fmt.Printf("Status: %s\n", colorizeStatus(metrics.Status))
	fmt.Printf("Uptime: %s\n", formatDuration(metrics.Uptime))
	fmt.Printf("\n")

	fmt.Printf("Performance:\n")
	fmt.Printf("  QPS: %.2f req/s\n", metrics.QPS)
	fmt.Printf("  Latency (P50): %d ms\n", metrics.LatencyP50)
	fmt.Printf("  Latency (P95): %d ms\n", metrics.LatencyP95)
	fmt.Printf("  Latency (P99): %d ms\n", metrics.LatencyP99)
	fmt.Printf("\n")

	fmt.Printf("Resources:\n")
	fmt.Printf("  GPU Utilization: %.1f%%\n", metrics.GPUUtilization)
	fmt.Printf("  Memory Usage: %.1f%%\n", metrics.MemoryUsage)
	fmt.Printf("  Active Connections: %d\n", metrics.ActiveConnections)
	fmt.Printf("\n")

	fmt.Printf("Cache:\n")
	fmt.Printf("  Hit Rate: %.1f%%\n", metrics.CacheHitRate*100)
	fmt.Printf("  Total Requests: %d\n", metrics.TotalRequests)
	fmt.Printf("  Error Rate: %.2f%%\n", metrics.ErrorRate*100)

	return nil
}

// watchMetrics 实时监控指标
func (mc *ModelCommand) watchMetrics(ctx context.Context, modelID string, interval int) error {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	// 清屏
	fmt.Print("\033[H\033[2J")

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// 移动光标到起始位置
			fmt.Print("\033[H")

			if err := mc.showMetrics(ctx, modelID); err != nil {
				return err
			}

			fmt.Printf("\n[Press Ctrl+C to exit]\n")
		}
	}
}

// testInference 测试推理
func (mc *ModelCommand) testInference(ctx context.Context, req *dto.InferenceRequest) error {
	resp, err := mc.modelService.Inference(ctx, req)
	if err != nil {
		return errors.Wrap(err, "inference failed")
	}

	fmt.Printf("Response:\n%s\n\n", resp.Text)
	fmt.Printf("Tokens: %d\n", resp.TokensUsed)
	fmt.Printf("Latency: %d ms\n", resp.LatencyMs)

	return nil
}

// testStreamInference 测试流式推理
func (mc *ModelCommand) testStreamInference(ctx context.Context, req *dto.InferenceRequest) error {
	stream, err := mc.modelService.StreamInference(ctx, req)
	if err != nil {
		return errors.Wrap(err, "stream inference failed")
	}

	fmt.Printf("Response:\n")
	for chunk := range stream {
		if chunk.Error != nil {
			return chunk.Error
		}
		fmt.Print(chunk.Text)
	}
	fmt.Printf("\n")

	return nil
}

// colorizeStatus 为状态添加颜色
func colorizeStatus(status string) string {
	switch status {
	case "deployed":
		return fmt.Sprintf("\033[32m%s\033[0m", status) // Green
	case "deploying":
		return fmt.Sprintf("\033[33m%s\033[0m", status) // Yellow
	case "failed":
		return fmt.Sprintf("\033[31m%s\033[0m", status) // Red
	default:
		return status
	}
}

// formatDuration 格式化时间间隔
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

//Personal.AI order the ending

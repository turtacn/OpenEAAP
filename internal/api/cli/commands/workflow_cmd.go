// Package commands implements CLI commands for OpenEAAP
package commands

import (
	"context"
	"encoding/json"
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

// WorkflowCmd represents the workflow command
var WorkflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Manage workflows",
	Long:  `Create, run, pause, resume, and query workflows in OpenEAAP.`,
}

// workflowCreateCmd creates a new workflow
var workflowCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new workflow",
	Long:  `Create a new workflow from a configuration file or interactive input.`,
	RunE:  runWorkflowCreate,
}

// workflowListCmd lists all workflows
var workflowListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all workflows",
	Long:  `List all workflows with their status and metadata.`,
	RunE:  runWorkflowList,
}

// workflowGetCmd gets a workflow by ID
var workflowGetCmd = &cobra.Command{
	Use:   "get [workflow-id]",
	Short: "Get workflow details",
	Long:  `Get detailed information about a specific workflow.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowGet,
}

// workflowRunCmd runs a workflow
var workflowRunCmd = &cobra.Command{
	Use:   "run [workflow-id]",
	Short: "Run a workflow",
	Long:  `Execute a workflow with optional input parameters.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowRun,
}

// workflowPauseCmd pauses a running workflow
var workflowPauseCmd = &cobra.Command{
	Use:   "pause [execution-id]",
	Short: "Pause a running workflow",
	Long:  `Pause a currently running workflow execution.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowPause,
}

// workflowResumeCmd resumes a paused workflow
var workflowResumeCmd = &cobra.Command{
	Use:   "resume [execution-id]",
	Short: "Resume a paused workflow",
	Long:  `Resume a paused workflow execution from the last checkpoint.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowResume,
}

// workflowStatusCmd gets workflow execution status
var workflowStatusCmd = &cobra.Command{
	Use:   "status [execution-id]",
	Short: "Get workflow execution status",
	Long:  `Get the current status and progress of a workflow execution.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowStatus,
}

// workflowCancelCmd cancels a workflow execution
var workflowCancelCmd = &cobra.Command{
	Use:   "cancel [execution-id]",
	Short: "Cancel a workflow execution",
	Long:  `Cancel a running or paused workflow execution.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowCancel,
}

// workflowDeleteCmd deletes a workflow
var workflowDeleteCmd = &cobra.Command{
	Use:   "delete [workflow-id]",
	Short: "Delete a workflow",
	Long:  `Delete a workflow and all its execution history.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowDelete,
}

var (
	// Global flags
	workflowSvc service.WorkflowService

	// Create flags
	workflowFile        string
	workflowName        string
	workflowDescription string
	workflowInteractive bool

	// List flags
	workflowStatus string
	workflowLimit  int
	workflowOffset int

	// Run flags
	workflowInput     string
	workflowInputFile string
	workflowSync      bool
	workflowWatch     bool

	// Output flags
	outputFormat string
	outputFile   string
)

func init() {
	// Add subcommands
	WorkflowCmd.AddCommand(workflowCreateCmd)
	WorkflowCmd.AddCommand(workflowListCmd)
	WorkflowCmd.AddCommand(workflowGetCmd)
	WorkflowCmd.AddCommand(workflowRunCmd)
	WorkflowCmd.AddCommand(workflowPauseCmd)
	WorkflowCmd.AddCommand(workflowResumeCmd)
	WorkflowCmd.AddCommand(workflowStatusCmd)
	WorkflowCmd.AddCommand(workflowCancelCmd)
	WorkflowCmd.AddCommand(workflowDeleteCmd)

	// Create flags
	workflowCreateCmd.Flags().StringVarP(&workflowFile, "file", "f", "", "Workflow configuration file (YAML/JSON)")
	workflowCreateCmd.Flags().StringVarP(&workflowName, "name", "n", "", "Workflow name")
	workflowCreateCmd.Flags().StringVarP(&workflowDescription, "description", "d", "", "Workflow description")
	workflowCreateCmd.Flags().BoolVarP(&workflowInteractive, "interactive", "i", false, "Interactive mode")

	// List flags
	workflowListCmd.Flags().StringVarP(&workflowStatus, "status", "s", "", "Filter by status (running|paused|completed|failed)")
	workflowListCmd.Flags().IntVarP(&workflowLimit, "limit", "l", 20, "Maximum number of results")
	workflowListCmd.Flags().IntVarP(&workflowOffset, "offset", "o", 0, "Result offset")

	// Run flags
	workflowRunCmd.Flags().StringVarP(&workflowInput, "input", "i", "", "Input parameters (JSON string)")
	workflowRunCmd.Flags().StringVarP(&workflowInputFile, "input-file", "f", "", "Input parameters file (JSON/YAML)")
	workflowRunCmd.Flags().BoolVarP(&workflowSync, "sync", "s", false, "Wait for workflow completion")
	workflowRunCmd.Flags().BoolVarP(&workflowWatch, "watch", "w", false, "Watch workflow execution progress")

	// Global output flags
	WorkflowCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table|json|yaml)")
	WorkflowCmd.PersistentFlags().StringVar(&outputFile, "output-file", "", "Write output to file")
}

// SetWorkflowService sets the workflow service (called from main)
func SetWorkflowService(svc service.WorkflowService) {
	workflowSvc = svc
}

// runWorkflowCreate creates a new workflow
func runWorkflowCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	var req dto.CreateWorkflowRequest

	// Load from file
	if workflowFile != "" {
		data, err := os.ReadFile(workflowFile)
		if err != nil {
			return errors.Wrap(err, "failed to read workflow file")
		}

		if err := json.Unmarshal(data, &req); err != nil {
			return errors.Wrap(err, "failed to parse workflow file")
		}
	} else if workflowInteractive {
		// Interactive mode
		if err := promptWorkflowCreate(&req); err != nil {
			return err
		}
	} else {
		// Use flags
		if workflowName == "" {
			return errors.New("workflow name is required (use --name or --interactive)")
		}
		req.Name = workflowName
		req.Description = workflowDescription
	}

	// Validate request
	if req.Name == "" {
		return errors.New("workflow name cannot be empty")
	}

	// Create workflow
	resp, err := workflowSvc.CreateWorkflow(ctx, &req)
	if err != nil {
		return errors.Wrap(err, "failed to create workflow")
	}

	// Output result
	fmt.Printf("✅ Workflow created successfully\n")
	fmt.Printf("ID: %s\n", resp.ID)
	fmt.Printf("Name: %s\n", resp.Name)
	fmt.Printf("Status: %s\n", resp.Status)

	return nil
}

// runWorkflowList lists all workflows
func runWorkflowList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	req := &dto.ListWorkflowsRequest{
		Status: workflowStatus,
		Limit:  workflowLimit,
		Offset: workflowOffset,
	}

	resp, err := workflowSvc.ListWorkflows(ctx, req)
	if err != nil {
		return errors.Wrap(err, "failed to list workflows")
	}

	if len(resp.Workflows) == 0 {
		fmt.Println("No workflows found")
		return nil
	}

	// Output based on format
	switch outputFormat {
	case "json":
		return outputJSON(resp.Workflows)
	case "yaml":
		return outputYAML(resp.Workflows)
	default:
		return outputWorkflowsTable(resp.Workflows)
	}
}

// runWorkflowGet gets a workflow by ID
func runWorkflowGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	workflowID := args[0]

	resp, err := workflowSvc.GetWorkflow(ctx, workflowID)
	if err != nil {
		return errors.Wrap(err, "failed to get workflow")
	}

	// Output based on format
	switch outputFormat {
	case "json":
		return outputJSON(resp)
	case "yaml":
		return outputYAML(resp)
	default:
		return outputWorkflowDetails(resp)
	}
}

// runWorkflowRun runs a workflow
func runWorkflowRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	workflowID := args[0]

	req := &dto.RunWorkflowRequest{
		WorkflowID: workflowID,
	}

	// Parse input parameters
	if workflowInputFile != "" {
		data, err := os.ReadFile(workflowInputFile)
		if err != nil {
			return errors.Wrap(err, "failed to read input file")
		}
		var input map[string]interface{}
		if err := json.Unmarshal(data, &input); err != nil {
			return errors.Wrap(err, "failed to parse input file")
		}
		req.Input = input
	} else if workflowInput != "" {
		var input map[string]interface{}
		if err := json.Unmarshal([]byte(workflowInput), &input); err != nil {
			return errors.Wrap(err, "failed to parse input JSON")
		}
		req.Input = input
	}

	// Run workflow
	resp, err := workflowSvc.RunWorkflow(ctx, req)
	if err != nil {
		return errors.Wrap(err, "failed to run workflow")
	}

	fmt.Printf("✅ Workflow execution started\n")
	fmt.Printf("Execution ID: %s\n", resp.ExecutionID)
	fmt.Printf("Status: %s\n", resp.Status)

	// Wait for completion if sync mode
	if workflowSync || workflowWatch {
		return watchWorkflowExecution(ctx, resp.ExecutionID)
	}

	return nil
}

// runWorkflowPause pauses a running workflow
func runWorkflowPause(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	executionID := args[0]

	if err := workflowSvc.PauseWorkflow(ctx, executionID); err != nil {
		return errors.Wrap(err, "failed to pause workflow")
	}

	fmt.Printf("✅ Workflow execution paused: %s\n", executionID)
	return nil
}

// runWorkflowResume resumes a paused workflow
func runWorkflowResume(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	executionID := args[0]

	if err := workflowSvc.ResumeWorkflow(ctx, executionID); err != nil {
		return errors.Wrap(err, "failed to resume workflow")
	}

	fmt.Printf("✅ Workflow execution resumed: %s\n", executionID)
	return nil
}

// runWorkflowStatus gets workflow execution status
func runWorkflowStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	executionID := args[0]

	status, err := workflowSvc.GetExecutionStatus(ctx, executionID)
	if err != nil {
		return errors.Wrap(err, "failed to get workflow status")
	}

	// Output based on format
	switch outputFormat {
	case "json":
		return outputJSON(status)
	case "yaml":
		return outputYAML(status)
	default:
		return outputExecutionStatus(status)
	}
}

// runWorkflowCancel cancels a workflow execution
func runWorkflowCancel(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	executionID := args[0]

	if err := workflowSvc.CancelWorkflow(ctx, executionID); err != nil {
		return errors.Wrap(err, "failed to cancel workflow")
	}

	fmt.Printf("✅ Workflow execution cancelled: %s\n", executionID)
	return nil
}

// runWorkflowDelete deletes a workflow
func runWorkflowDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	workflowID := args[0]

	// Confirm deletion
	fmt.Printf("⚠️  Are you sure you want to delete workflow %s? (yes/no): ", workflowID)
	var confirm string
	fmt.Scanln(&confirm)

	if strings.ToLower(confirm) != "yes" {
		fmt.Println("Deletion cancelled")
		return nil
	}

	if err := workflowSvc.DeleteWorkflow(ctx, workflowID); err != nil {
		return errors.Wrap(err, "failed to delete workflow")
	}

	fmt.Printf("✅ Workflow deleted: %s\n", workflowID)
	return nil
}

// promptWorkflowCreate prompts for workflow creation in interactive mode
func promptWorkflowCreate(req *dto.CreateWorkflowRequest) error {
	fmt.Println("=== Create Workflow (Interactive Mode) ===")

	fmt.Print("Name: ")
	fmt.Scanln(&req.Name)

	fmt.Print("Description: ")
	fmt.Scanln(&req.Description)

	// TODO: Add more interactive prompts for steps, triggers, etc.

	return nil
}

// outputWorkflowsTable outputs workflows in table format
func outputWorkflowsTable(workflows []*dto.WorkflowResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSTATUS\tSTEPS\tCREATED")
	fmt.Fprintln(w, "----\t----\t------\t-----\t-------")

	for _, wf := range workflows {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
			wf.ID,
			wf.Name,
			wf.Status,
			len(wf.Steps),
			wf.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}

	return w.Flush()
}

// outputWorkflowDetails outputs workflow details
func outputWorkflowDetails(wf *dto.WorkflowResponse) error {
	fmt.Printf("ID:          %s\n", wf.ID)
	fmt.Printf("Name:        %s\n", wf.Name)
	fmt.Printf("Description: %s\n", wf.Description)
	fmt.Printf("Status:      %s\n", wf.Status)
	fmt.Printf("Steps:       %d\n", len(wf.Steps))
	fmt.Printf("Created:     %s\n", wf.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:     %s\n", wf.UpdatedAt.Format("2006-01-02 15:04:05"))

	if len(wf.Steps) > 0 {
		fmt.Println("\nSteps:")
		for i, step := range wf.Steps {
			fmt.Printf("  %d. %s (%s)\n", i+1, step.Name, step.Type)
		}
	}

	return nil
}

// outputExecutionStatus outputs execution status
func outputExecutionStatus(status *dto.ExecutionStatusResponse) error {
	fmt.Printf("Execution ID: %s\n", status.ExecutionID)
	fmt.Printf("Workflow ID:  %s\n", status.WorkflowID)
	fmt.Printf("Status:       %s\n", status.Status)
	fmt.Printf("Progress:     %.1f%%\n", status.Progress*100)
	fmt.Printf("Current Step: %s\n", status.CurrentStep)
	fmt.Printf("Started:      %s\n", status.StartedAt.Format("2006-01-02 15:04:05"))

	if status.CompletedAt != nil {
		fmt.Printf("Completed:    %s\n", status.CompletedAt.Format("2006-01-02 15:04:05"))
		duration := status.CompletedAt.Sub(status.StartedAt)
		fmt.Printf("Duration:     %s\n", duration.Round(time.Second))
	}

	if status.Error != "" {
		fmt.Printf("\n❌ Error: %s\n", status.Error)
	}

	if len(status.Steps) > 0 {
		fmt.Println("\nStep Status:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "STEP\tSTATUS\tDURATION")
		fmt.Fprintln(w, "----\t------\t--------")

		for _, step := range status.Steps {
			duration := ""
			if step.CompletedAt != nil && step.StartedAt != nil {
				duration = step.CompletedAt.Sub(*step.StartedAt).Round(time.Second).String()
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", step.Name, step.Status, duration)
		}
		w.Flush()
	}

	return nil
}

// watchWorkflowExecution watches workflow execution progress
func watchWorkflowExecution(ctx context.Context, executionID string) error {
	fmt.Println("\n⏳ Watching workflow execution...")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			status, err := workflowSvc.GetExecutionStatus(ctx, executionID)
			if err != nil {
				return errors.Wrap(err, "failed to get execution status")
			}

			// Clear line and print progress
			fmt.Printf("\r📊 Progress: %.1f%% | Status: %s | Step: %s",
				status.Progress*100, status.Status, status.CurrentStep)

			// Check if completed
			if status.Status == "completed" {
				fmt.Printf("\n\n✅ Workflow completed successfully\n")
				return outputExecutionStatus(status)
			} else if status.Status == "failed" {
				fmt.Printf("\n\n❌ Workflow failed\n")
				return outputExecutionStatus(status)
			} else if status.Status == "cancelled" {
				fmt.Printf("\n\n⚠️  Workflow cancelled\n")
				return outputExecutionStatus(status)
			}
		}
	}
}

// outputJSON outputs data in JSON format
func outputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// outputYAML outputs data in YAML format
func outputYAML(data interface{}) error {
	// TODO: Implement YAML output using yaml.v3
	return fmt.Errorf("YAML output not implemented")
}

//Personal.AI order the ending

// internal/api/cli/cobra.go
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/openeeap/openeeap/internal/api/cli/commands"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// 全局配置文件路径
	cfgFile string
	// 全局配置对象
	cfg *config.Config
	// 全局日志对象
	logger logging.Logger
	// 输出格式 (json|yaml|table)
	outputFormat string
	// 详细模式
	verbose bool
)

// rootCmd 根命令
var rootCmd = &cobra.Command{
	Use:   "openeeap",
	Short: "OpenEAAP - Enterprise AI Agent Platform",
	Long: `OpenEAAP CLI is a command-line tool for managing AI agents, workflows, models, and data.
   
It provides comprehensive capabilities for:
 - Agent lifecycle management (create, deploy, test, delete)
 - Workflow orchestration and execution
 - Model registry and deployment
 - Knowledge base management
 - System monitoring and troubleshooting`,
	Version: "1.0.0",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// 初始化配置
		if err := initConfig(); err != nil {
			return fmt.Errorf("failed to initialize config: %w", err)
		}

		// 初始化日志
		if err := initLogger(); err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}

		return nil
	},
}

// Execute 执行 CLI 命令
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// 全局持久化标志
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $HOME/.openeeap/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "output format (json|yaml|table)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	// 添加子命令组
	addSubCommands()

	// 设置帮助和用法模板
	rootCmd.SetHelpTemplate(helpTemplate)
	rootCmd.SetUsageTemplate(usageTemplate)
}

// addSubCommands 添加所有子命令
func addSubCommands() {
	// Agent 命令组
	rootCmd.AddCommand(commands.NewAgentCmd())

	// Workflow 命令组
	rootCmd.AddCommand(commands.NewWorkflowCmd())

	// Model 命令组
	rootCmd.AddCommand(commands.NewModelCmd())

	// Data 命令组 (知识库管理)
	rootCmd.AddCommand(commands.NewDataCmd())

	// System 命令组 (系统管理)
	rootCmd.AddCommand(commands.NewSystemCmd())

	// Config 命令 (配置管理)
	rootCmd.AddCommand(commands.NewConfigCmd())

	// Version 命令 (版本信息)
	rootCmd.AddCommand(newVersionCmd())

	// Completion 命令 (自动补全)
	rootCmd.AddCommand(newCompletionCmd())
}

// initConfig 初始化配置
func initConfig() error {
	if cfgFile != "" {
		// 使用指定的配置文件
		viper.SetConfigFile(cfgFile)
	} else {
		// 查找默认配置文件
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		configDir := filepath.Join(home, ".openeeap")
		viper.AddConfigPath(configDir)
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/openeeap")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// 读取环境变量
	viper.SetEnvPrefix("OPENEEAP")
	viper.AutomaticEnv()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		// 配置文件不存在时创建默认配置
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if err := createDefaultConfig(); err != nil {
				return fmt.Errorf("failed to create default config: %w", err)
			}
		} else {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// 解析配置到结构体
	cfg = &config.Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	return nil
}

// createDefaultConfig 创建默认配置文件
func createDefaultConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".openeeap")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configFile := filepath.Join(configDir, "config.yaml")
	defaultConfig := `# OpenEAAP CLI Configuration
server:
 host: localhost
 port: 8080
 api_version: v1
 timeout: 30s

auth:
 type: api_key
 api_key: ""  # Set your API key here

output:
 format: table
 color: true

log:
 level: info
 file: ""  # Empty means stdout
`

	if err := os.WriteFile(configFile, []byte(defaultConfig), 0644); err != nil {
		return err
	}

	viper.SetConfigFile(configFile)
	return viper.ReadInConfig()
}

// initLogger 初始化日志
func initLogger() error {
	logLevel := "info"
	if verbose {
		logLevel = "debug"
	}
	if cfg != nil && cfg.Log.Level != "" {
		logLevel = cfg.Log.Level
	}

	logConfig := logging.Config{
		Level:      logLevel,
		Format:     "text",
		Output:     "stdout",
		TimeFormat: "2006-01-02 15:04:05",
	}

	var err error
	logger, err = logging.NewLogger(logConfig)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	return nil
}

// GetConfig 获取全局配置
func GetConfig() *config.Config {
	return cfg
}

// GetLogger 获取全局日志对象
func GetLogger() logging.Logger {
	return logger
}

// GetOutputFormat 获取输出格式
func GetOutputFormat() string {
	return outputFormat
}

// IsVerbose 是否详细模式
func IsVerbose() bool {
	return verbose
}

// newVersionCmd 创建 version 命令
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("OpenEAAP CLI Version: %s\n", rootCmd.Version)
			fmt.Printf("Build Date: %s\n", "2026-01-15")
			fmt.Printf("Go Version: %s\n", "go1.22")
		},
	}
}

// newCompletionCmd 创建 completion 命令
func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `To load completions:

Bash:
 $ source <(openeeap completion bash)
 # To load completions for each session, execute once:
 # Linux:
 $ openeeap completion bash > /etc/bash_completion.d/openeeap
 # macOS:
 $ openeeap completion bash > /usr/local/etc/bash_completion.d/openeeap

Zsh:
 # If shell completion is not already enabled in your environment,
 # you will need to enable it. You can execute the following once:
 $ echo "autoload -U compinit; compinit" >> ~/.zshrc
 # To load completions for each session, execute once:
 $ openeeap completion zsh > "${fpath[1]}/_openeeap"
 # You will need to start a new shell for this setup to take effect.

Fish:
 $ openeeap completion fish | source
 # To load completions for each session, execute once:
 $ openeeap completion fish > ~/.config/fish/completions/openeeap.fish

PowerShell:
 PS> openeeap completion powershell | Out-String | Invoke-Expression
 # To load completions for every new session, run:
 PS> openeeap completion powershell > openeeap.ps1
 # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}
}

// helpTemplate 自定义帮助模板
const helpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`

// usageTemplate 自定义用法模板
const usageTemplate = `Usage:{{if .Runnable}}
 {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
 {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
 {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
 {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
 {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

//Personal.AI order the ending

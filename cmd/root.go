package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// root 命令的全局变量，可以在子命令中访问
var (
	baseURL      string
	insecureSkip bool
)

// rootCmd 代表了程序的基础命令，所有的子命令都将依附于它
var rootCmd = &cobra.Command{
	Use:   "gitlab-fork-cli",
	Short: "一个用于 GitLab 项目派生的 CLI 工具",
	Long: `gitlab-fork-cli 是一个命令行工具，
用于自动化从一个 GitLab 组派生项目到另一个 GitLab 组的操作。

例如:
  gitlab-fork-cli fork --source-group my-dev --source-project my-app --target-group my-prod --dev-token <token1> --prod-token <token2>`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute 为你的 root 命令添加所有子命令，并适当设置标志。
// 这是从 main() 调用的。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %s\n", err)
		os.Exit(1)
	}
}

func init() {
	// 定义全局标志 (flag)
	rootCmd.PersistentFlags().StringVarP(&baseURL, "base-url", "u", "https://aml-gitlab.alaudatech.net", "GitLab API 的基础 URL (e.g., 'https://gitlab.com')")
	rootCmd.PersistentFlags().BoolVarP(&insecureSkip, "insecure", "k", false, "跳过 TLS 证书验证 (⚠️ 慎用)")

	// 注册子命令 (在 fork.go 中定义)
	rootCmd.AddCommand(forkCmd)
	rootCmd.AddCommand(listProjectsCmd)
}

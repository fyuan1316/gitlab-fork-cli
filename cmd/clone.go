package cmd

import (
	"fmt"
	"github.com/fy1316/gitlab-fork-cli/pkg"
	"github.com/spf13/cobra"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// 定义 clone 命令的参数变量
var (
	fromRepoURL         string // 源 Git 仓库地址
	fromRef             string // 源仓库要克隆的分支或标签
	fromToken           string // 源仓库用于认证的个人访问令牌
	toRepoURL           string // 目的 Git 仓库地址
	toTag               string // push 到目的仓库的标签名称 (可选，省略时使用源标签名)
	toToken             string // 目的仓库用于认证的个人访问令牌
	outputDir           string // 克隆到的本地目录
	onTagExistsBehavior string // 处理标签已存在的行为
)

// cloneCmd 定义了 'clone' 子命令
var cloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "克隆 Git 仓库并推送到目标仓库",
	Long: `此命令用于从指定的源 Git 仓库克隆代码，然后推送到指定的目的 Git 仓库。
支持指定源分支或标签，并可提供个人访问令牌进行认证。
可以指定推送的目标标签，如果省略则尝试推送所有标签。`,
	Run: func(cmd *cobra.Command, args []string) {
		// 1. 参数校验
		if fromRepoURL == "" {
			log.Fatal("必须提供 --from-repo-url 参数。")
		}
		if toRepoURL == "" {
			log.Fatal("必须提供 --to-repo-url 参数。")
		}
		if fromRef == "" {
			log.Fatal("必须提供 --from-ref 参数（源分支或标签名）。")
		}
		if outputDir == "" {
			// 如果未指定 outputDir，则使用默认的临时目录
			// 在实际应用中，你可能希望生成一个更唯一的目录名
			// 使用当前时间戳作为随机数种子
			//rand.Seed(time.Now().UnixNano())
			source := rand.NewSource(time.Now().UnixNano())
			r := rand.New(source)
			// 生成一个随机数作为后缀
			randSuffix := strconv.Itoa(r.Intn(100000))
			outputDir = filepath.Join(os.TempDir(), "go-git-clone-push-temp-"+randSuffix)
			log.Printf("未指定 --output-dir，将使用随机临时目录: %s", outputDir)
		}

		// 2. 构造认证方式
		var fromAuth pkg.GitAuthMethod
		if fromToken != "" {
			fromAuth = &pkg.BasicAuthMethod{Username: "oauth2", Password: fromToken}
		}

		var toAuth pkg.GitAuthMethod
		if toToken != "" {
			toAuth = &pkg.BasicAuthMethod{Username: "oauth2", Password: toToken}
		}

		// 3. 构造操作选项
		opts := pkg.GitOperationOptions{
			FromRepoURL:         fromRepoURL,
			FromRef:             fromRef,
			FromAuth:            fromAuth,
			ToRepoURL:           toRepoURL,
			ToTag:               toTag,
			ToAuth:              toAuth,
			OutputDir:           outputDir,
			ProgressWriter:      os.Stdout, // 将进度输出到标准输出
			OnTagExistsBehavior: onTagExistsBehavior,
		}

		// 4. 执行核心操作
		err := pkg.PerformGitOperation(opts)
		if err != nil {
			log.Fatalf("Git 操作失败: %v", err)
		}

		fmt.Println("Git 仓库克隆和推送操作成功完成！")
	},
}

func init() {
	// 定义 clone 命令的本地标志
	cloneCmd.Flags().StringVarP(&fromRepoURL, "from-repo-url", "", "", "源 Git 仓库的 URL (必填)")
	cloneCmd.Flags().StringVarP(&fromRef, "from-ref", "", "", "源仓库要克隆的分支名称或标签名称 (必填)")
	cloneCmd.Flags().StringVarP(&fromToken, "from-token", "", "glpat-Uou_WTfqMyWn9wyZ_HNX", "源仓库用于认证的个人访问令牌 (可选)")
	cloneCmd.Flags().StringVarP(&toRepoURL, "to-repo-url", "", "", "目的 Git 仓库的 URL (必填)")
	cloneCmd.Flags().StringVarP(&toTag, "to-tag", "", "", "推送至目的仓库的标签名称 (可选，省略时使用源标签名)")
	cloneCmd.Flags().StringVarP(&toToken, "to-token", "", "glpat-5QL4aihz5PSymiALe1Uv", "目的仓库用于认证的个人访问令牌 (可选)")
	cloneCmd.Flags().StringVarP(&outputDir, "output-dir", "", "", "将仓库克隆到的本地目录 (可选，默认为临时目录)")
	cloneCmd.Flags().StringVarP(&onTagExistsBehavior, "on-tag-exists", "", "error", "处理目标标签已存在的行为：'error' (报错), 'skip' (跳过)")

	// 标记必填参数
	cloneCmd.MarkFlagRequired("from-repo-url")
	cloneCmd.MarkFlagRequired("from-ref")
	cloneCmd.MarkFlagRequired("to-repo-url")
}

package cmd

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// 定义 fork 命令的参数变量
var (
	sourceGroup   string
	sourceProject string
	targetGroup   string
	devToken      string
	prodToken     string
)

// newGitLabClient 封装了 GitLab 客户端的创建逻辑
func newGitLabClient(token, baseURL string, insecureSkipVerify bool) (*gitlab.Client, error) {
	var httpClient *http.Client
	if insecureSkipVerify {
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}

	client, err := gitlab.NewClient(
		token,
		gitlab.WithBaseURL(baseURL),
		gitlab.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("创建 GitLab 客户端失败: %w", err)
	}
	return client, nil
}

// findProjectInGroup 在指定组中查找项目并返回其 ID
func findProjectInGroup(client *gitlab.Client, groupID string, projectName string) (int, error) {
	listOptions := &gitlab.ListGroupProjectsOptions{}
	listOptions.PerPage = 100
	listOptions.IncludeSubGroups = gitlab.Ptr(true)

	// 循环遍历所有页，确保找到项目
	for {
		projects, resp, err := client.Groups.ListGroupProjects(groupID, listOptions)
		if err != nil {
			return -1, fmt.Errorf("列出组 '%s' 的项目失败: %w", groupID, err)
		}
		if resp.StatusCode != http.StatusOK {
			return -1, fmt.Errorf("列出组 '%s' 的项目失败，HTTP 状态码: %d", groupID, resp.StatusCode)
		}

		for _, p := range projects {
			if p.Name == projectName {
				fmt.Printf("✅ 找到源项目: %s (ID: %d) 在组 '%s'\n", p.NameWithNamespace, p.ID, groupID)
				return p.ID, nil
			}
		}

		// 如果没有下一页，则退出循环
		if listOptions.Page == 0 || resp.NextPage == 0 {
			break
		}
		listOptions.Page = resp.NextPage
	}

	return -1, fmt.Errorf("在组 '%s' 中未找到项目 '%s'", groupID, projectName)
}

// forkCmd 定义了 'fork' 子命令
var forkCmd = &cobra.Command{
	Use:   "fork",
	Short: "将一个 GitLab 项目派生到另一个组",
	Long: `此命令将指定的源项目从其当前组派生到目标组。
需要两个 GitLab 个人访问令牌：一个用于读取源项目，一个用于在目标组创建项目。`,
	Run: func(cmd *cobra.Command, args []string) {
		// 检查必填参数是否提供
		if sourceGroup == "" || sourceProject == "" || targetGroup == "" || devToken == "" || prodToken == "" {
			fmt.Println("❌ 错误: 缺少必要的命令行参数。请使用 --help 查看用法。")
			cmd.Help()
			os.Exit(1)
		}

		// 1. 使用 devToken 创建客户端，用于查询源项目
		fmt.Printf("ℹ️ 正在使用开发令牌创建客户端 (%s)...\n", baseURL)
		devGit, err := newGitLabClient(devToken, baseURL, insecureSkip)
		if err != nil {
			log.Fatalf("❌ %v", err)
		}

		// 2. 查找源项目 ID
		sourceProjectID, err := findProjectInGroup(devGit, sourceGroup, sourceProject)
		if err != nil {
			log.Fatalf("❌ %v", err)
		}

		// 3. 使用 prodToken 创建客户端，用于在目标组执行派生操作
		fmt.Printf("ℹ️ 正在使用生产令牌创建客户端 (%s)...\n", baseURL)
		prodGit, err := newGitLabClient(prodToken, baseURL, insecureSkip)
		if err != nil {
			log.Fatalf("❌ %v", err)
		}

		// 4. 执行派生操作
		fmt.Printf("🚀 正在将项目 '%s' (ID: %d) 从 '%s' 派生到目标组 '%s'...\n",
			sourceProject, sourceProjectID, sourceGroup, targetGroup)

		forkOptions := &gitlab.ForkProjectOptions{
			Namespace: gitlab.Ptr(targetGroup), // 确保派生到正确的组
		}

		newProject, resp, err := prodGit.Projects.ForkProject(sourceProjectID, forkOptions)
		if err != nil {
			if resp != nil && (resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusConflict) {
				log.Fatalf("❌ 派生项目失败。可能原因：目标组 '%s' 不存在，或生产令牌在目标组没有派生权限，或目标组已存在同名项目。HTTP 状态码: %d。原始错误: %v", targetGroup, resp.StatusCode, err)
			}
			log.Fatalf("❌ 派生项目失败: %v", err)
		}

		if resp.StatusCode != http.StatusCreated {
			log.Fatalf("❌ 派生项目失败，HTTP 状态码: %d", resp.StatusCode)
		}

		// 5. 打印新派生项目的信息
		fmt.Println("\n🎉 项目派生成功！新项目信息:")
		fmt.Printf("  ID: %d\n", newProject.ID)
		fmt.Printf("  名称: %s\n", newProject.Name)
		fmt.Printf("  带命名空间的全名: %s\n", newProject.NameWithNamespace)
		fmt.Printf("  Web URL: %s\n", newProject.WebURL)
		if newProject.ForkedFromProject != nil {
			fmt.Printf("  派生自: %s (ID: %d)\n", newProject.ForkedFromProject.NameWithNamespace, newProject.ForkedFromProject.ID)
		} else {
			fmt.Println("  派生自: (信息不可用或非派生项目)")
		}

		fmt.Println("\n✅ 操作完成。")
	},
}

func init() {
	// 定义 fork 命令的本地标志
	forkCmd.Flags().StringVarP(&sourceGroup, "source-group", "g", "", "源项目所在的 GitLab 组的路径或 ID (必填)")
	forkCmd.Flags().StringVarP(&sourceProject, "source-project", "p", "", "要派生（fork）的源项目的名称 (必填)")
	forkCmd.Flags().StringVarP(&targetGroup, "target-group", "t", "", "派生项目将要创建到的目标 GitLab 组的路径或 ID (必填)")
	forkCmd.Flags().StringVarP(&devToken, "dev-token", "d", "", "用于读取源项目的 GitLab 个人访问令牌 (必填)")
	forkCmd.Flags().StringVarP(&prodToken, "prod-token", "r", "", "用于在目标组创建（派生）项目的 GitLab 个人访问令牌 (必填)")

	// 标记这些标志为必填
	forkCmd.MarkFlagRequired("source-group")
	forkCmd.MarkFlagRequired("source-project")
	forkCmd.MarkFlagRequired("target-group")
	forkCmd.MarkFlagRequired("dev-token")
	forkCmd.MarkFlagRequired("prod-token")
}

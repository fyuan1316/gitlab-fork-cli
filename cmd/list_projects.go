package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// 定义 list-projects 命令的参数变量
var (
	listGroup      string
	listToken      string
	listVisibility string
)

// listProjectsCmd 定义了 'list-projects' 子命令
var listProjectsCmd = &cobra.Command{
	Use:   "list-projects",
	Short: "列出指定 GitLab 组下的所有项目",
	Long: `此命令用于列出指定 GitLab 组下的所有项目。
可以根据可见性 (public, private, internal) 进行筛选。

例如:
  gitlab-fork-cli list-projects --group my-dev --token <your_token>
  gitlab-fork-cli list-projects --group my-prod --token <your_token> --visibility public`,
	Run: func(cmd *cobra.Command, args []string) {
		// 检查必填参数
		if listGroup == "" || listToken == "" {
			fmt.Println("❌ 错误: 缺少必要的命令行参数 (--group 和 --token)。")
			cmd.Help()
			os.Exit(1)
		}

		// 验证 visibility 参数
		if listVisibility != "" {
			validVisibilities := map[string]struct{}{
				"public":   {},
				"private":  {},
				"internal": {},
			}
			if _, ok := validVisibilities[strings.ToLower(listVisibility)]; !ok {
				log.Fatalf("❌ 错误: 无效的可见性参数 '%s'。有效值: public, private, internal。", listVisibility)
			}
		}

		// 1. 创建 GitLab 客户端
		fmt.Printf("ℹ️ 正在创建 GitLab 客户端 (%s)...\n", baseURL)
		git, err := newGitLabClient(listToken, baseURL, insecureSkip)
		if err != nil {
			log.Fatalf("❌ %v", err)
		}

		// 2. 设置项目列表选项
		listOptions := &gitlab.ListGroupProjectsOptions{}
		listOptions.PerPage = 100
		listOptions.IncludeSubGroups = gitlab.Ptr(true)

		// 根据可见性参数设置筛选条件
		if listVisibility != "" {
			switch strings.ToLower(listVisibility) {
			case "public":
				listOptions.Visibility = gitlab.Ptr(gitlab.PublicVisibility)
			case "private":
				listOptions.Visibility = gitlab.Ptr(gitlab.PrivateVisibility)
			case "internal":
				listOptions.Visibility = gitlab.Ptr(gitlab.InternalVisibility)
			}
		}

		fmt.Printf("🚀 正在获取组 '%s' 下的项目 (可见性: %s)...\n", listGroup, func() string {
			if listVisibility == "" {
				return "所有"
			}
			return listVisibility
		}())

		// 3. 循环遍历所有页，获取项目列表
		allProjects := []*gitlab.Project{}
		for {
			projects, resp, err := git.Groups.ListGroupProjects(listGroup, listOptions)
			if err != nil {
				log.Fatalf("❌ 列出组 '%s' 的项目失败: %v", listGroup, err)
			}
			if resp.StatusCode != http.StatusOK {
				log.Fatalf("❌ 列出组 '%s' 的项目失败，HTTP 状态码: %d", listGroup, resp.StatusCode)
			}

			allProjects = append(allProjects, projects...)

			if resp.NextPage == 0 {
				break // 没有更多页了
			}
			listOptions.Page = resp.NextPage
		}

		// 4. 打印项目信息
		if len(allProjects) == 0 {
			fmt.Printf("ℹ️ 组 '%s' (可见性: %s) 下没有找到任何项目。\n", listGroup, func() string {
				if listVisibility == "" {
					return "所有"
				}
				return listVisibility
			}())
		} else {
			fmt.Printf("\n🎉 组 '%s' (可见性: %s) 下的项目列表 (%d 个):\n", listGroup, func() string {
				if listVisibility == "" {
					return "所有"
				}
				return listVisibility
			}(), len(allProjects))
			for i, p := range allProjects {
				fmt.Printf("  %d. %s (ID: %d, 路径: %s, 可见性: %s)\n",
					i+1, p.NameWithNamespace, p.ID, p.PathWithNamespace, p.Visibility)
			}
		}

		fmt.Println("\n✅ 操作完成。")
	},
}

func init() {
	// 定义 list-projects 命令的本地标志
	listProjectsCmd.Flags().StringVarP(&listGroup, "group", "g", "", "要列出项目的 GitLab 组的路径或 ID (必填)")
	listProjectsCmd.Flags().StringVarP(&listToken, "token", "t", "", "用于访问 GitLab API 的个人访问令牌 (必填)")
	listProjectsCmd.Flags().StringVarP(&listVisibility, "visibility", "v", "", "可选: 按可见性筛选项目 (public, private, internal)")

	// 标记这些标志为必填
	listProjectsCmd.MarkFlagRequired("group")
	listProjectsCmd.MarkFlagRequired("token")
}

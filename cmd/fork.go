package cmd

import (
	"crypto/tls"
	"fmt"
	"github.com/fy1316/gitlab-fork-cli/pkg/k8sutil"
	"log"
	"net/http"
	"strings"

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

const (
	GitlabSecretName = "aml-image-builder-secret"
	GitlabTokenKey   = "MODEL_REPO_GIT_TOKEN"
	amlModelsGroup   = "amlmodels"
)

func getModelGroupByNs(ns string) string {
	return ns + "/" + amlModelsGroup
}

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
		// 1. Check required command-line arguments
		if sourceGroup == "" || sourceProject == "" || targetGroup == "" || baseURL == "" {
			log.Fatal("❌ 错误: 缺少必要的命令行参数。请使用 --help 查看用法。")
		}

		// Get Kubernetes config once, for all K8s operations
		log.Println("ℹ️ 正在获取 Kubernetes 配置...")
		kubeRestConfig, err := k8sutil.GetKubeConfig()
		if err != nil {
			log.Fatalf("❌ 无法获取 Kubernetes 配置，无法检查命名空间或获取 Secret。错误: %v\n", err)
		}

		// 2. Check if sourceGroup (as Namespace) exists
		log.Printf("ℹ️ 正在检查源组 (Kubernetes 命名空间) '%s' 是否存在...\n", sourceGroup)
		sourceNsExists, err := k8sutil.CheckK8sNamespaceExists(kubeRestConfig, sourceGroup)
		if err != nil {
			log.Fatalf("❌ 检查源组命名空间失败。源组: %s, 错误: %v\n", sourceGroup, err)
		}
		if !sourceNsExists {
			log.Fatalf("❌ 源组对应的 Kubernetes 命名空间 '%s' 不存在。请确认该命名空间已被纳管。\n", sourceGroup)
		}

		// 3. Check if targetGroup (as Namespace) exists
		log.Printf("ℹ️ 正在检查目标组 (Kubernetes 命名空间) '%s' 是否存在...\n", targetGroup)
		targetNsExists, err := k8sutil.CheckK8sNamespaceExists(kubeRestConfig, targetGroup)
		if err != nil {
			log.Fatalf("❌ 检查目标组命名空间失败。目标组: %s, 错误: %v\n", targetGroup, err)
		}
		if !targetNsExists {
			log.Fatalf("❌ 目标组对应的 Kubernetes 命名空间 '%s' 不存在。请确认该命名空间已被纳管。\n", targetGroup)
		}

		// 4. Get devToken from Kubernetes Secret (sourceGroup as Namespace)
		log.Printf("ℹ️ 正在从 Kubernetes Secret 获取开发令牌...命名空间: %s, Secret名称: %s\n",
			sourceGroup, GitlabSecretName)
		devToken, err := k8sutil.GetSecretValue(kubeRestConfig, sourceGroup, GitlabSecretName, GitlabTokenKey)
		if err != nil {
			log.Fatalf("❌ 无法获取开发令牌。请确认输入的 source-group (%s) 对应的 Secret 存在且可访问。错误: %v\n",
				sourceGroup, err)
		}
		log.Println("✅ 成功获取开发令牌。")

		// 5. Create devGit client to query source project
		log.Printf("ℹ️ 正在使用开发令牌创建 GitLab 客户端...Base URL: %s\n", baseURL)
		devGit, err := newGitLabClient(devToken, baseURL, insecureSkip)
		if err != nil {
			log.Fatalf("❌ 创建 GitLab 开发客户端失败: %v\n", err)
		}

		// 6. Find source project ID
		log.Printf("ℹ️ 正在查找源项目 '%s' 是否存在于 GitLab 组 '%s'...\n", sourceProject, sourceGroup)
		sourceProjectID, err := findProjectInGroup(devGit, sourceGroup, sourceProject)
		if err != nil {
			log.Fatalf("❌ 源项目在 GitLab 组 '%s' 中未找到或查询失败。请确认项目名称和权限。错误: %v\n",
				sourceGroup, err)
		}
		log.Printf("✅ 源项目 '%s' 已在 GitLab 组 '%s' 中找到。ID: %d\n",
			sourceProject, sourceGroup, sourceProjectID)

		// 7. Get prodToken from Kubernetes Secret (targetGroup as Namespace)
		log.Printf("ℹ️ 正在从 Kubernetes Secret 获取生产令牌...命名空间: %s, Secret名称: %s\n",
			targetGroup, GitlabSecretName)
		prodToken, err := k8sutil.GetSecretValue(kubeRestConfig, targetGroup, GitlabSecretName, GitlabTokenKey)
		if err != nil {
			log.Fatalf("❌ 无法获取生产令牌。请确认输入的 target-group (%s) 对应的 Secret 存在且可访问。错误: %v\n",
				targetGroup, err)
		}
		log.Println("✅ 成功获取生产令牌。")

		// 8. Create prodGit client to perform fork operation in target group
		log.Printf("ℹ️ 正在使用生产令牌创建 GitLab 客户端...Base URL: %s\n", baseURL)
		prodGit, err := newGitLabClient(prodToken, baseURL, insecureSkip)
		if err != nil {
			log.Fatalf("❌ 创建 GitLab 生产客户端失败: %v\n", err)
		}

		// 9. Check if a project with the same name already exists in the target group
		log.Printf("ℹ️ 正在检查目标组 '%s' 中是否已存在同名项目 '%s'...\n", targetGroup, sourceProject)
		existingProjectID, err := findProjectInGroup(prodGit, getModelGroupByNs(targetGroup), sourceProject)
		if err == nil {
			log.Fatalf("❌ 目标组 '%s' 中已存在同名项目 '%s' (ID: %d)。请手动处理或更改目标项目名称。\n",
				targetGroup, sourceProject, existingProjectID)
		}
		// If the error is "project not found", it's expected and we can proceed.
		// Any other error means the check itself failed, and we should exit.
		if err != nil && !strings.Contains(err.Error(), "未找到项目") {
			log.Fatalf("❌ 检查目标组是否存在同名项目失败。目标组: %s, 项目: %s, 错误: %v\n",
				targetGroup, sourceProject, err)
		}
		log.Printf("✅ 目标组 '%s' 中未发现同名项目 '%s'，可以继续派生。\n", targetGroup, sourceProject)

		// 10. Perform the fork operation
		adminToken, err := k8sutil.GetSecretValue(kubeRestConfig, "kubeflow", GitlabSecretName, GitlabTokenKey)
		if err != nil {
			log.Fatalf("❌ 无法获取生产令牌。请确认输入的 target-group (%s) 对应的 Secret 存在且可访问。错误: %v\n",
				"kubeflow", err)
		}

		log.Println("✅ 成功获取生产令牌。")
		admindGit, err := newGitLabClient(adminToken, baseURL, insecureSkip)
		if err != nil {
			log.Fatalf("❌ 创建 GitLab 生产客户端失败: %v\n", err)
		}

		log.Printf("🚀 正在将项目 '%s' (ID: %d) 派生到目标组 '%s'...\n",
			sourceProject, sourceProjectID, targetGroup)

		forkOptions := &gitlab.ForkProjectOptions{
			Namespace: gitlab.Ptr(getModelGroupByNs(targetGroup)), // Ensure forking to the correct group
		}

		// Use prodGit for the fork operation as it has the necessary permissions for the target group
		newProject, resp, err := admindGit.Projects.ForkProject(sourceProjectID, forkOptions)
		if err != nil {
			if resp != nil {
				log.Printf("派生项目请求返回错误状态码。源项目: %s, 目标组: %s, HTTP状态码: %d, 原始错误: %v\n",
					sourceProject, targetGroup, resp.StatusCode, err)
				switch resp.StatusCode {
				case http.StatusNotFound:
					log.Fatal("❌ 派生项目失败: 可能原因 - 目标组不存在，或源项目不存在。")
				case http.StatusForbidden:
					log.Fatal("❌ 派生项目失败: 生产令牌在目标组没有足够的派生权限。")
				case http.StatusConflict:
					log.Fatal("❌ 派生项目失败: 目标组中已存在同名项目。") // This should ideally be caught by the pre-check
				default:
					log.Fatalf("❌ 派生项目失败: %v\n", err)
				}
			}
			log.Fatalf("❌ 派生项目请求失败: %v\n", err)
		}

		if resp.StatusCode != http.StatusCreated {
			log.Fatalf("❌ 派生项目失败，HTTP 状态码不是 201 Created，实际状态码: %d\n", resp.StatusCode)
		}

		// 11. Print information about the newly forked project
		log.Println("\n🎉 项目派生成功！新项目信息:")
		log.Printf("  ID: %d\n", newProject.ID)
		log.Printf("  名称: %s\n", newProject.Name)
		log.Printf("  带命名空间的全名: %s\n", newProject.PathWithNamespace)
		log.Printf("  Web URL: %s\n", newProject.WebURL)
		if newProject.ForkedFromProject != nil {
			log.Printf("  派生自: %s (ID: %d)\n", newProject.ForkedFromProject.NameWithNamespace, newProject.ForkedFromProject.ID)
		} else {
			log.Println("  派生自: (信息不可用或非派生项目)")
		}

		log.Println("\n✅ 操作完成。")
	},
}

func init() {
	// 定义 fork 命令的本地标志
	forkCmd.Flags().StringVarP(&sourceGroup, "source-group", "g", "", "项目开发所在的NS名称 (GitLab 组的名称)(必填)")
	forkCmd.Flags().StringVarP(&sourceProject, "source-project", "p", "", "平台项目的名称 (必填)")
	forkCmd.Flags().StringVarP(&targetGroup, "target-group", "t", "", "项目推理服务将要创建到的NS名称 (必填)")
	//forkCmd.Flags().StringVarP(&devToken, "dev-token", "d", "", "用于读取源项目的 GitLab 个人访问令牌 (必填)")
	//forkCmd.Flags().StringVarP(&prodToken, "prod-token", "r", "", "用于在目标组创建（派生）项目的 GitLab 个人访问令牌 (必填)")

	// 标记这些标志为必填
	forkCmd.MarkFlagRequired("source-group")
	forkCmd.MarkFlagRequired("source-project")
	forkCmd.MarkFlagRequired("target-group")
	//forkCmd.MarkFlagRequired("dev-token")
	//forkCmd.MarkFlagRequired("prod-token")
}

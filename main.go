//package main
//
//import (
//	"crypto/tls"
//	"fmt"
//	"log"
//	"net/http"
//
//	gitlab "gitlab.com/gitlab-org/api/client-go"
//)
//
//var BASE_URL = "https://aml-gitlab.alaudatech.net"
//
//func newGitLabClient(token, baseURL string, insecureSkipVerify bool) (*gitlab.Client, error) {
//	var httpClient *http.Client
//	if insecureSkipVerify {
//		httpClient = &http.Client{
//			Transport: &http.Transport{
//				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
//			},
//		}
//	}
//
//	client, err := gitlab.NewClient(
//		token,
//		gitlab.WithBaseURL(baseURL),
//		gitlab.WithHTTPClient(httpClient),
//	)
//	if err != nil {
//		return nil, fmt.Errorf("创建 GitLab 客户端失败: %w", err)
//	}
//	return client, nil
//}
//func main() {
//	gitlabAPIURL := fmt.Sprintf("%s/api/v4", BASE_URL)
//
//	// 配置开发环境相关参数
//	devToken := "glpat-Uou_WTfqMyWn9wyZ_HNX" // 用于访问源项目和源组的令牌
//	devGroup := "fy-dev"                     // 源项目所在的组
//	sourceProjectName := "iris"              // 要派生的源项目名称
//
//	// 配置生产环境相关参数 (目标环境)
//	prodToken := "glpat-5QL4aihz5PSymiALe1Uv" // 用于在目标组创建项目的令牌
//	targetGroup := "fy-prod"                  // 目标组
//
//	if targetGroup == "" {
//		log.Fatalf("目标组 (targetGroup) 是必须的。")
//	}
//
//	// 1. 使用 devToken 创建客户端，用于查询源项目
//	fmt.Printf("正在使用开发令牌创建客户端，API 基础 URL: %s\n", gitlabAPIURL)
//	devGit, err := newGitLabClient(devToken, gitlabAPIURL, true)
//	if err != nil {
//		log.Fatalf("%v", err) // 使用 %v 打印错误对象
//	}
//
//	// 2. 查找源项目
//	fmt.Printf("正在 '%s' 组中查找项目 '%s'...\n", devGroup, sourceProjectName)
//	listOptions := &gitlab.ListGroupProjectsOptions{}
//	listOptions.PerPage = 20
//	listOptions.Page = 1
//	listOptions.IncludeSubGroups = gitlab.Ptr(true)
//	projects, rsp, err := devGit.Groups.ListGroupProjects(devGroup, listOptions)
//	if err != nil {
//		log.Fatalf("列出 '%s' 组项目失败: %v", devGroup, err)
//	}
//	if rsp.StatusCode != http.StatusOK {
//		log.Fatalf("列出 '%s' 组项目失败，HTTP 状态码: %d", devGroup, rsp.StatusCode)
//	}
//
//	found := false
//	sourceProjectID := -1
//	for _, p := range projects {
//		if p.Name == sourceProjectName {
//			found = true
//			sourceProjectID = p.ID
//			fmt.Printf("找到源项目: %s (ID: %d)\n", p.NameWithNamespace, p.ID)
//			break // 找到即退出循环
//		}
//	}
//
//	if !found {
//		log.Fatalf("[命名空间=%s] 项目 '%s' 未找到。", devGroup, sourceProjectName)
//	}
//
//	// ---
//	// 针对 404 错误优化点 1 & 2：
//	// 问题：原始代码用 devToken 去 fork，但 devToken 可能在 targetGroup 没有创建项目的权限。
//	// 解决方案：使用 prodToken 创建一个新的客户端，专门用于在目标组执行派生操作。
//	// ---
//
//	// 3. 使用 prodToken 创建客户端，用于在目标组执行派生操作
//	fmt.Printf("正在使用生产令牌创建客户端，API 基础 URL: %s\n", gitlabAPIURL)
//	prodGit, err := newGitLabClient(prodToken, gitlabAPIURL, true)
//	if err != nil {
//		log.Fatalf("%v", err)
//	}
//
//	// 4. 执行派生操作
//	fmt.Printf("正在将项目 ID %d (源项目: %s) 派生到目标组 '%s'...\n", sourceProjectID, sourceProjectName, targetGroup)
//	forkOptions := &gitlab.ForkProjectOptions{
//		Namespace: gitlab.Ptr(targetGroup), // 确保派生到正确的组
//	}
//
//	newProject, resp, err := prodGit.Projects.ForkProject(sourceProjectID, forkOptions)
//	if err != nil {
//		// 进一步检查错误类型，提供更具体提示
//		if resp != nil && (resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden) {
//			log.Fatalf("派生项目失败。可能原因：目标组 '%s' 不存在，或生产令牌在目标组没有派生权限 (HTTP 状态码: %d)。原始错误: %v", targetGroup, resp.StatusCode, err)
//		}
//		log.Fatalf("派生项目失败: %v", err)
//	}
//
//	fmt.Printf("HTTP 响应状态: %s (状态码: %d)\n", resp.Status, resp.StatusCode)
//
//	if resp.StatusCode != http.StatusCreated { // 派生成功通常返回 201 Created
//		log.Fatalf("派生项目失败，HTTP 状态码: %d", resp.StatusCode)
//	}
//
//	// 5. 打印新派生项目的信息
//	fmt.Println("\n项目派生成功！新项目信息:")
//	fmt.Printf("  ID: %d\n", newProject.ID)
//	fmt.Printf("  名称: %s\n", newProject.Name)
//	fmt.Printf("  带命名空间的全名: %s\n", newProject.NameWithNamespace)
//	fmt.Printf("  Web URL: %s\n", newProject.WebURL)
//	if newProject.ForkedFromProject != nil {
//		fmt.Printf("  派生自: %s (ID: %d)\n", newProject.ForkedFromProject.NameWithNamespace, newProject.ForkedFromProject.ID)
//	} else {
//		fmt.Println("  派生自: (信息不可用或非派生项目)")
//	}
//
//	// 其他被注释掉的代码块可以根据需要取消注释并使用
//}

package main

import (
	"github.com/fy1316/gitlab-fork-cli/cmd"
)

func main() {
	cmd.Execute() // 执行 Cobra 根命令
}

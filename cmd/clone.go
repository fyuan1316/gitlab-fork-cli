package cmd

import (
	"fmt"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/go-git/go-git/v6/storage/memory"
	"github.com/spf13/cobra"
	"log"
	"os"
	"slices"
)

// 定义 clone 命令的参数变量
var (
	repoURL   string // Git 仓库地址
	gitRef    string // 要克隆的分支或标签
	gitToken  string // 用于认证的个人访问令牌
	outputDir string // 克隆到的本地目录
)

// cloneCmd 定义了 'clone' 子命令
var cloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "克隆一个 Git 仓库并下载模型",
	Long: `此命令用于从指定的 Git 仓库克隆代码。
支持指定分支或标签，并可提供个人访问令牌进行认证。
克隆完成后，将包含一个模型下载的占位符。`,
	Run: func(cmd *cobra.Command, args []string) {
		tag, branch := "1.0.0", "main"
		t, err := checkRemoteRefExistence(repoURL, "1.0.0", "main", "glpat-Uou_WTfqMyWn9wyZ_HNX")
		fmt.Printf("ref type= %v", t)

		cloneOptions := &git.CloneOptions{
			URL:             repoURL,
			Progress:        os.Stdout,
			InsecureSkipTLS: true, //
			Depth:           1,
		}

		if t == 1 {
			//refName := plumbing.NewTagReferenceName(tag)

		} else if t == 2 {
			//refName := plumbing.NewBranchReferenceName(branch)
			cloneOptions.SingleBranch = true
		}

		//url, directory, token := os.Args[1], os.Args[2], os.Args[3]
		directory := "/tmp/test1"
		// Clone the given repository to the given directory
		fmt.Printf("git clone %s %s", repoURL, directory)

		r, err := git.PlainClone(directory, &git.CloneOptions{
			// The intended use of a GitHub personal access token is in replace of your password
			// because access tokens can easily be revoked.
			// https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/
			Auth: &http.BasicAuth{
				Username: "oauth2",
				Password: gitToken,
			},
			InsecureSkipTLS: true,
			URL:             repoURL,
			Progress:        os.Stdout,
		})
		if err != nil {
			log.Fatalf("plainclone err: %v", err)
		}

		// ... retrieving the branch being pointed by HEAD
		ref, err := r.Head()
		if err != nil {
			log.Fatalf("plainclone err: %v", err)
		}
		// ... retrieving the commit object
		commit, err := r.CommitObject(ref.Hash())
		if err != nil {
			log.Fatalf("plainclone err: %v", err)
		}
		fmt.Println(commit)
	},
}

func checkRemoteRefExistence(repoURL, tag string, branch string, gitToken string) (int, error) {
	// Create the remote with repository URL
	rem := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{repoURL},
	})

	log.Print("Fetching tags...")

	// We can then use every Remote functions to retrieve wanted information
	refs, err := rem.List(&git.ListOptions{
		// Returns all references, including peeled references.
		PeelingOption:   git.AppendPeeled,
		InsecureSkipTLS: true,
		Auth: &http.BasicAuth{
			Username: "oauth2",
			Password: gitToken,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Filters the references list and only keeps tags
	var tags, branches []string
	for _, ref := range refs {
		if ref.Name().IsTag() {
			tags = append(tags, ref.Name().Short())
		}
		if ref.Name().IsBranch() {
			branches = append(branches, ref.Name().Short())
		}
	}
	if slices.Contains(tags, tag) {
		return 1, nil
	}
	if slices.Contains(branches, branch) {
		return 2, nil
	}
	return -1, nil
}

func init() {
	// 定义 clone 命令的本地标志
	cloneCmd.Flags().StringVarP(&repoURL, "repo-url", "f", "", "要克隆的 Git 仓库的 URL (必填)")
	cloneCmd.Flags().StringVarP(&gitRef, "ref", "r", "", "要克隆的分支名称或标签名称 (可选)")
	cloneCmd.Flags().StringVarP(&gitToken, "token", "t", "", "用于认证的个人访问令牌 (可选，对于私有仓库通常需要)")
	cloneCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "将仓库克隆到的本地目录 (可选，默认为仓库名称)")

	// 标记 repo-url 为必填
	cloneCmd.MarkFlagRequired("repo-url")
}

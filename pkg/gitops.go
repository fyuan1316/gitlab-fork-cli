package pkg

import (
	"errors"
	"fmt"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/transport"
	"github.com/go-git/go-git/v6/plumbing/transport/http" // 引入 HTTP 认证
	"github.com/go-git/go-git/v6/storage/memory"
	"io"
	"log"
	"slices"
	"strings"
)

// --- 认证接口定义 ---
// GitAuthMethod 定义了 Git 认证方法的接口
type GitAuthMethod interface {
	GetAuthMethod() transport.AuthMethod
}

// BasicAuthMethod 实现了 GitAuthMethod 接口，用于 HTTP Basic 认证
type BasicAuthMethod struct {
	Username string
	Password string
}

// GetAuthMethod 返回 HTTP Basic 认证方法
func (b *BasicAuthMethod) GetAuthMethod() transport.AuthMethod {
	return &http.BasicAuth{
		Username: b.Username,
		Password: b.Password,
	}
}

// --- 引用类型别名 ---
type RefType int

const (
	RefTypeUnknown RefType = iota // 未知类型
	RefTypeTag                    // 标签
	RefTypeBranch                 // 分支
)

// String 方法用于提高 RefType 的可读性
func (rt RefType) String() string {
	switch rt {
	case RefTypeTag:
		return "tag"
	case RefTypeBranch:
		return "branch"
	default:
		return "unknown"
	}
}

// --- 核心操作函数 ---

// GitOperationOptions 包含 Git 操作所需的所有参数
type GitOperationOptions struct {
	FromRepoURL         string
	FromRef             string // 源仓库分支或标签名
	FromAuth            GitAuthMethod
	ToRepoURL           string
	ToTag               string // 目标仓库标签名 (可选)
	ToAuth              GitAuthMethod
	OutputDir           string // 克隆到的本地目录
	ProgressWriter      io.Writer
	OnTagExistsBehavior string
}

// PerformGitOperation 执行克隆和推送的端到端 Git 操作
func PerformGitOperation(opts GitOperationOptions) error {
	// 1. 检查源仓库引用的类型（标签或分支）
	refType, err := checkRemoteRefExistence(opts.FromRepoURL, opts.FromRef, opts.FromAuth)
	if err != nil {
		return fmt.Errorf("检查源仓库引用 (%s) 失败: %w", opts.FromRef, err)
	}
	if refType == RefTypeUnknown {
		return fmt.Errorf("源仓库中未找到分支或标签: %s", opts.FromRef)
	}

	// 2. 配置克隆选项
	cloneOptions := &git.CloneOptions{
		URL:             opts.FromRepoURL,
		Progress:        opts.ProgressWriter,
		InsecureSkipTLS: true, // 生产环境请谨慎使用
		Depth:           1,    // 浅克隆，只获取最新提交
		SingleBranch:    true, // 只克隆指定的分支/标签
	}

	if opts.FromAuth != nil {
		cloneOptions.Auth = opts.FromAuth.GetAuthMethod()
	}

	// 根据引用类型设置克隆的目标引用
	if refType == RefTypeTag {
		cloneOptions.ReferenceName = plumbing.NewTagReferenceName(opts.FromRef)
		log.Printf("检测到源引用 '%s' 为标签，将克隆该标签。", opts.FromRef)
	} else if refType == RefTypeBranch {
		cloneOptions.ReferenceName = plumbing.NewBranchReferenceName(opts.FromRef)
		log.Printf("检测到源引用 '%s' 为分支，将克隆该分支。", opts.FromRef)
	}

	// 3. 执行克隆操作
	log.Printf("正在克隆仓库 %s 到 %s...", opts.FromRepoURL, opts.OutputDir)
	r, err := git.PlainClone(opts.OutputDir, cloneOptions) // false 表示非裸仓库
	if err != nil {
		if errors.Is(err, git.ErrRepositoryAlreadyExists) {
			log.Printf("目标目录 '%s' 已存在且是一个 Git 仓库，尝试打开而不是克隆。", opts.OutputDir)
			r, err = git.PlainOpen(opts.OutputDir)
			if err != nil {
				return fmt.Errorf("无法打开现有仓库 %s: %w", opts.OutputDir, err)
			}
			// 如果是打开现有仓库，我们应该先拉取，确保是最新的，或者提示用户
			log.Printf("警告: 目录 '%s' 已存在，克隆操作跳过。请确保它是所需状态。", opts.OutputDir)
			// 简单起见，这里假设如果目录存在且是仓库，我们就不再做拉取操作，直接进行下一步push。
			// 实际应用中可能需要更复杂的逻辑，比如先拉取或强制删除目录。
		} else {
			return fmt.Errorf("克隆失败: %w", err)
		}
	}
	log.Printf("仓库已成功克隆到 %s", opts.OutputDir)

	// 4. 配置目标远程仓库
	log.Printf("正在配置目标远程仓库 %s...", opts.ToRepoURL)
	targetRemoteConfig := &config.RemoteConfig{
		Name: "target", // 远程名称固定为 "target"
		URLs: []string{opts.ToRepoURL},
	}
	gitTarget, err := r.CreateRemote(targetRemoteConfig)
	if err != nil && !errors.Is(err, git.ErrRemoteExists) { // 如果远程已经存在，忽略错误
		return fmt.Errorf("创建远程仓库配置失败: %w", err)
	} else if errors.Is(err, git.ErrRemoteExists) {
		log.Printf("远程 '%s' 已存在，跳过创建。", targetRemoteConfig.Name)
		// 如果远程已存在，获取现有远程对象
		gitTarget, err = r.Remote(targetRemoteConfig.Name)
		if err != nil {
			return fmt.Errorf("无法获取已存在的远程 '%s': %w", targetRemoteConfig.Name, err)
		}
	}

	// 5. 配置推送选项
	pushOptions := &git.PushOptions{
		RemoteName:      "target",
		Progress:        opts.ProgressWriter,
		InsecureSkipTLS: true, // 生产环境请谨慎使用
	}
	if opts.ToAuth != nil {
		pushOptions.Auth = opts.ToAuth.GetAuthMethod()
	}

	// 设置推送的 RefSpecs
	if opts.ToTag != "" { // 如果指定了目标标签，则推送指定的标签
		// 获取本地克隆下来的 ref 对应的 commit hash
		localRef, err := r.Reference(plumbing.ReferenceName(fmt.Sprintf("refs/remotes/origin/%s", opts.FromRef)), false) // 如果是分支
		if refType == RefTypeTag {
			localRef, err = r.Reference(plumbing.NewTagReferenceName(opts.FromRef), false) // 如果是标签
		}
		if err != nil {
			return fmt.Errorf("无法获取本地引用 %s: %w", opts.FromRef, err)
		}

		// 推送本地 ref 的 hash 到目标标签
		pushOptions.RefSpecs = []config.RefSpec{
			config.RefSpec(fmt.Sprintf("%s:refs/tags/%s", localRef.Hash().String(), opts.ToTag)),
		}
		log.Printf("将本地提交 %s 推送到目标仓库标签 %s。", localRef.Hash().String(), opts.ToTag)
	} else { // 如果未指定目标标签，则推送所有标签
		pushOptions.RefSpecs = []config.RefSpec{
			config.RefSpec("refs/tags/*:refs/tags/*"), // 推送所有标签
		}
		log.Println("未指定目标标签，将推送所有本地标签到目标仓库。")
	}

	// 6. 执行推送操作
	log.Printf("正在推送内容到目标仓库 %s...", opts.ToRepoURL)
	err = gitTarget.Push(pushOptions)
	if err != nil {
		//if errors.Is(err, git.ErrRemoteExists) {
		//	// NoPushError 表示没有要推送的新内容，通常不是错误
		//	log.Printf("推送完成: 目标仓库已经最新，无需推送。")
		//	return nil
		//}

		// 目前虽然返回错误，但是推送是成功的
		// https://github.com/go-git/go-git/issues/1600
		if strings.Contains(err.Error(), "decode report-status: unknown channel unpack ok") {
			log.Println("内容已成功推送到目标仓库。")
			return nil
		}

		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			tag := opts.ToTag
			if tag == "" {
				tag = opts.FromRef
			}
			refType, err = checkRemoteRefExistence(opts.ToRepoURL, tag, opts.ToAuth)
			if err != nil {
				return fmt.Errorf("检查标签 '%s' 已存在于目标仓库 发生错误 %v。", tag, err)
			}
			if refType == RefTypeTag {
				switch opts.OnTagExistsBehavior {
				case "error":
					return fmt.Errorf("标签 '%s' 已存在于目标仓库，且配置为报错模式。", tag)
				case "skip":
					log.Printf("标签 '%s' 已存在于目标仓库，已跳过推送。", tag)
					return nil // 视为成功，不返回错误
				default:
					// 理论上不会发生，因为设置了默认值
					return fmt.Errorf("未知的 --on-tag-exists 行为: %s", opts.OnTagExistsBehavior)
				}
			}
		}
		return fmt.Errorf("推送失败: %w", err)
	}

	log.Println("内容已成功推送到目标仓库。")
	return nil
}

// checkRemoteRefExistence 检查远程仓库中是否存在指定的分支或标签
// 返回 1 表示是标签，2 表示是分支，-1 表示未找到或未知
func checkRemoteRefExistence(repoURL, refName string, auth GitAuthMethod) (RefType, error) {
	rem := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{repoURL},
	})

	log.Printf("正在从 %s 获取引用列表...", repoURL)

	listOptions := &git.ListOptions{
		PeelingOption:   git.AppendPeeled,
		InsecureSkipTLS: true,
	}
	if auth != nil {
		listOptions.Auth = auth.GetAuthMethod()
	}

	refs, err := rem.List(listOptions)
	if err != nil {
		return RefTypeUnknown, fmt.Errorf("列出远程引用失败: %w", err)
	}

	var tags, branches []string
	for _, ref := range refs {
		if ref.Name().IsTag() {
			tags = append(tags, ref.Name().Short())
		} else if ref.Name().IsBranch() { // 区分分支和标签
			branches = append(branches, ref.Name().Short())
		}
	}

	if slices.Contains(tags, refName) {
		log.Printf("引用 '%s' 存在于远程仓库并被识别为标签。", refName)
		return RefTypeTag, nil
	}
	if slices.Contains(branches, refName) {
		log.Printf("引用 '%s' 存在于远程仓库并被识别为分支。", refName)
		return RefTypeBranch, nil
	}

	log.Printf("引用 '%s' 在远程仓库中未被识别为标签或分支。", refName)
	return RefTypeUnknown, nil
}

package api

import (
	"context"
	"dailyDataPanel/internal/conf"
	"encoding/json"
	"fmt"
	"path/filepath"
)

// GitLabAPI 封装GitLab API操作
type GitLabAPI struct {
	client      *HTTPClient
	url         string
	accessToken string
	projectID   uint
	issueIID    uint
}

func NewGitLabAPI() *GitLabAPI {
	appConf := conf.GetAppConfig()
	return &GitLabAPI{
		client:      NewDefaultHTTPClient(),
		url:         appConf.Gitlab.URL,
		accessToken: appConf.Gitlab.AccessToken,
		projectID:   uint(appConf.Gitlab.ProjectID),
		issueIID:    uint(appConf.Gitlab.IssueIID),
	}
}

// CommentCreate 创建评论
func (g *GitLabAPI) CommentCreate(ctx context.Context, msg string) error {
	payload := map[string]string{
		"body": msg,
	}

	headers := map[string]string{
		"PRIVATE-TOKEN": g.accessToken,
	}

	url := g.url + fmt.Sprintf("/api/v4/projects/%d/issues/%d/notes", g.projectID, g.issueIID)

	_, err := g.client.PostJSON(ctx, url, payload, headers)
	if err != nil {
		return fmt.Errorf("创建评论失败: %w", err)
	}

	return nil
}

// UploadFile 上传文件并返回markdown字符串引用文本
func (g *GitLabAPI) UploadFile(ctx context.Context, filePath string) (string, error) {
	// 准备文件字段
	files := []FileField{
		{
			FieldName: "file",
			FilePath:  filePath,
			FileName:  filepath.Base(filePath),
		},
	}

	headers := map[string]string{
		"PRIVATE-TOKEN": g.accessToken,
	}

	url := g.url + fmt.Sprintf("/api/v4/projects/%d/uploads", g.projectID)

	resp, err := g.client.PostMultipart(ctx, url, nil, files, headers)
	if err != nil {
		return "", fmt.Errorf("上传文件失败: %w", err)
	}

	// 解析响应
	uploadResponse := struct {
		Alt      string `json:"alt"`
		URL      string `json:"url"`
		Markdown string `json:"markdown"`
	}{}

	if err := json.Unmarshal(resp.Body, &uploadResponse); err != nil {
		return "", fmt.Errorf("解析上传响应失败: %w", err)
	}

	return uploadResponse.Markdown, nil
}

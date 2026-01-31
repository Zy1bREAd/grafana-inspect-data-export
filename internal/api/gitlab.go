package api

import (
	"bytes"
	"context"
	"dailyDataPanel/internal/conf"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

// 主要是封装一个操作GITLAB API的Handler
type GitLabAPI struct {
	URL         string
	AccessToken string
	ProjectID   uint
	IssueIID    uint
}

func NewGitLabAPI() *GitLabAPI {
	appConf := conf.GetAppConfig()
	return &GitLabAPI{
		URL:         appConf.Gitlab.URL,
		AccessToken: appConf.Gitlab.AccessToken,
		ProjectID:   uint(appConf.Gitlab.ProjectID),
		IssueIID:    uint(appConf.Gitlab.IssueIID),
	}
}

// 创建评论
func (gitlab *GitLabAPI) CommentCreate(ctx context.Context, msg string) error {
	// 构造JSON形式作为Body
	payload := map[string]string{
		"body": msg,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	// 评论API接口地址
	commentCreateURL := gitlab.URL + fmt.Sprintf("/api/v4/projects/%d/issues/%d/notes", gitlab.ProjectID, gitlab.IssueIID)
	req, err := http.NewRequestWithContext(ctx, "POST", commentCreateURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", gitlab.AccessToken)
	// 设置请求头，携带JSON形式的POST请求体
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// 获取响应结果
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	// 状态码检查（成功是201）
	if resp.StatusCode > 299 || resp.StatusCode < 200 {
		return errors.New("request error: " + string(respBody))
	}

	return nil
}

// Gitlab 上传文件，并返回markdown字符串引用文本。
func (gitlab *GitLabAPI) UploadFile(ctx context.Context, filePath string) (string, error) {
	// 打开要上传的文件
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	// 创建 multipart/form-data 请求体
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加文件字段（字段名必须是 "file"）
	part, err := writer.CreateFormFile("file", filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return "", fmt.Errorf("failed to copy file into form: %w", err)
	}

	// 关闭 writer（必须调用，否则 boundary 不完整）
	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	apiURL := gitlab.URL + fmt.Sprintf("/api/v4/projects/%d/uploads", gitlab.ProjectID)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("PRIVATE-TOKEN", gitlab.AccessToken)
	// 设置请求头(需要导入文件路径)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 获取响应结果，提取markdown引用的字符串。
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	// 状态码检查（成功是201）
	if resp.StatusCode > 299 || resp.StatusCode < 200 {
		return "", errors.New("request error: " + string(respBody))
	}

	uploadResponse := struct {
		Alt      string `json:"alt"`
		URL      string `json:"url"`
		Markdown string `json:"markdown"`
	}{}
	err = json.Unmarshal(respBody, &uploadResponse)
	if err != nil {
		return "", err
	}
	return uploadResponse.Markdown, nil
}

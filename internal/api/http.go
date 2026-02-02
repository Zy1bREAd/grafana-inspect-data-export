package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

// HTTPClient 封装HTTP客户端，提供常用的请求方法
type HTTPClient struct {
	client  *http.Client
	timeout time.Duration
}

// RequestOptions HTTP请求选项
type RequestOptions struct {
	Headers      map[string]string
	QueryParams  map[string]string
	FormData     map[string]string
	Files        []FileField
	Body         []byte
	ExpectedCode int // 期望的状态码，默认200
}

// FileField 文件字段
type FileField struct {
	FieldName string
	FilePath  string
	FileName  string // 可选，默认使用文件名
}

// Response HTTP响应
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// NewHTTPClient 创建新的HTTP客户端
func NewHTTPClient(timeout time.Duration) *HTTPClient {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// NewDefaultHTTPClient 创建默认HTTP客户端
func NewDefaultHTTPClient() *HTTPClient {
	return NewHTTPClient(30 * time.Second)
}

// Get 发送GET请求
func (c *HTTPClient) Get(ctx context.Context, url string, options *RequestOptions) (*Response, error) {
	return c.doRequest(ctx, "GET", url, options)
}

// Post 发送POST请求
func (c *HTTPClient) Post(ctx context.Context, url string, options *RequestOptions) (*Response, error) {
	return c.doRequest(ctx, "POST", url, options)
}

// Put 发送PUT请求
func (c *HTTPClient) Put(ctx context.Context, url string, options *RequestOptions) (*Response, error) {
	return c.doRequest(ctx, "PUT", url, options)
}

// Delete 发送DELETE请求
func (c *HTTPClient) Delete(ctx context.Context, url string, options *RequestOptions) (*Response, error) {
	return c.doRequest(ctx, "DELETE", url, options)
}

// PostJSON 发送JSON格式的POST请求
func (c *HTTPClient) PostJSON(ctx context.Context, url string, data any, headers map[string]string) (*Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	options := &RequestOptions{
		Headers: headers,
		Body:    jsonData,
	}

	if options.Headers == nil {
		options.Headers = make(map[string]string)
	}
	options.Headers["Content-Type"] = "application/json"

	return c.Post(ctx, url, options)
}

// PostFormData 发送表单数据的POST请求
func (c *HTTPClient) PostFormData(ctx context.Context, url string, formData map[string]string, headers map[string]string) (*Response, error) {
	options := &RequestOptions{
		Headers:  headers,
		FormData: formData,
	}

	return c.Post(ctx, url, options)
}

// PostMultipart 发送multipart/form-data格式的POST请求（支持文件上传）
func (c *HTTPClient) PostMultipart(ctx context.Context, url string, formData map[string]string, files []FileField, headers map[string]string) (*Response, error) {
	options := &RequestOptions{
		Headers:  headers,
		FormData: formData,
		Files:    files,
	}

	return c.Post(ctx, url, options)
}

// doRequest 执行HTTP请求的核心方法
func (c *HTTPClient) doRequest(ctx context.Context, method, url string, options *RequestOptions) (*Response, error) {
	var body io.Reader
	var contentType string
	var err error

	// 处理请求体
	if options != nil {
		if len(options.Files) > 0 {
			// 处理文件上传
			body, contentType, err = c.createMultipartBody(options.FormData, options.Files)
			if err != nil {
				return nil, err
			}
		} else if len(options.FormData) > 0 {
			// 处理表单数据
			formData := c.createFormDataBody(options.FormData)
			body = formData
			contentType = "application/x-www-form-urlencoded"
		} else if len(options.Body) > 0 {
			// 处理原始请求体
			body = bytes.NewReader(options.Body)
		}
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	// 设置请求头
	if options != nil && options.Headers != nil {
		for key, value := range options.Headers {
			req.Header.Set(key, value)
		}
	}

	// 设置Content-Type（如果未设置且有内容类型）
	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	response := &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       respBody,
	}

	// 检查状态码
	expectedCode := http.StatusOK
	if options != nil && options.ExpectedCode > 0 {
		expectedCode = options.ExpectedCode
	}

	if resp.StatusCode != expectedCode && !isSuccessCode(resp.StatusCode) {
		return response, &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}

	return response, nil
}

// createFormDataBody 创建表单请求体
func (c *HTTPClient) createFormDataBody(formData map[string]string) *bytes.Buffer {
	body := &bytes.Buffer{}
	for key, value := range formData {
		body.WriteString(key)
		body.WriteString("=")
		body.WriteString(value)
		body.WriteString("&")
	}
	// 移除最后的&
	if body.Len() > 0 {
		body.Truncate(body.Len() - 1)
	}
	return body
}

// createMultipartBody 创建multipart请求体（支持文件上传）
func (c *HTTPClient) createMultipartBody(formData map[string]string, files []FileField) (io.Reader, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加表单字段
	for key, value := range formData {
		err := writer.WriteField(key, value)
		if err != nil {
			return nil, "", err
		}
	}

	// 添加文件字段
	for _, file := range files {
		err := c.addFileToMultipart(writer, file)
		if err != nil {
			return nil, "", err
		}
	}

	// 关闭writer（必须调用，否则boundary不完整）
	err := writer.Close()
	if err != nil {
		return nil, "", err
	}

	return body, writer.FormDataContentType(), nil
}

// addFileToMultipart 添加文件到multipart请求体
func (c *HTTPClient) addFileToMultipart(writer *multipart.Writer, file FileField) error {
	fileObj, err := os.Open(file.FilePath)
	if err != nil {
		return err
	}
	defer fileObj.Close()

	fileName := file.FileName
	if fileName == "" {
		fileName = extractFileName(file.FilePath)
	}

	part, err := writer.CreateFormFile(file.FieldName, fileName)
	if err != nil {
		return err
	}

	_, err = io.Copy(part, fileObj)
	return err
}

// HTTPError HTTP错误
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return e.Message
}

// isSuccessCode 检查状态码是否表示成功
func isSuccessCode(code int) bool {
	return code >= 200 && code < 300
}

// extractFileName 从文件路径中提取文件名
func extractFileName(filePath string) string {
	for i := len(filePath) - 1; i >= 0; i-- {
		if filePath[i] == '/' || filePath[i] == '\\' {
			return filePath[i+1:]
		}
	}
	return filePath
}

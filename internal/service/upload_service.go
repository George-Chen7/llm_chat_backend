package service

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"backend/internal/llm"
	"backend/internal/store"
)

// UploadFileResult 上传结果。
type UploadFileResult struct {
	AttachmentID int
	MimeType     string
	URLOrPath    string
}

// SaveLocalFile 保存本地文件并返回公开路径。
func SaveLocalFile(reader io.Reader, filename string) (string, string, error) {
	if err := os.MkdirAll("uploads", 0o755); err != nil {
		return "", "", err
	}
	name := "upload.bin"
	if filename != "" {
		name = filepath.Base(filename)
	}
	storeName := strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + name
	storePath := filepath.Join("uploads", storeName)
	publicPath := "/uploads/" + storeName

	out, err := os.Create(storePath)
	if err != nil {
		return "", "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, reader); err != nil {
		return "", "", err
	}
	return storePath, publicPath, nil
}

// UploadAndRecord 上传并记录附件。
func UploadAndRecord(ctx context.Context, userID int, filename, mimeType string, reader io.Reader) (UploadFileResult, error) {
	if reader == nil {
		return UploadFileResult{}, errors.New("missing file")
	}

	llmModel := "unknown"
	if client := llm.Get(); client != nil && client.Model() != "" {
		llmModel = client.Model()
	}

	uploadConvID, err := store.GetOrCreateUploadConversation(ctx, userID, llmModel)
	if err != nil {
		return UploadFileResult{}, err
	}
	uploadMsgID, err := store.CreateUploadMessage(ctx, uploadConvID)
	if err != nil {
		return UploadFileResult{}, err
	}

	_, publicPath, err := SaveLocalFile(reader, filename)
	if err != nil {
		return UploadFileResult{}, err
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	attachID, err := store.CreateAttachment(ctx, uploadMsgID, "FILE", mimeType, "LOCAL", publicPath, nil)
	if err != nil {
		return UploadFileResult{}, err
	}

	return UploadFileResult{
		AttachmentID: attachID,
		MimeType:     mimeType,
		URLOrPath:    publicPath,
	}, nil
}

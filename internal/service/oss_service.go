package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"
	"time"

	"backend/internal/config"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

var (
	ossClient *oss.Client
	ossConfig config.OSSConfig

	// ErrOSSNotReady OSS client not initialized.
	ErrOSSNotReady = errors.New("oss client not initialized")
)

// InitOSS initializes the OSS client if config is enabled.
func InitOSS(cfg config.OSSConfig) error {
	ossConfig = cfg
	if !cfg.Enabled() {
		ossClient = nil
		return nil
	}
	if cfg.Region == "" || cfg.Endpoint == "" {
		return fmt.Errorf("oss config missing region or endpoint")
	}

	provider := credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.AccessKeySecret, cfg.SecurityToken)
	clientCfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(provider).
		WithRegion(cfg.Region).
		WithEndpoint(cfg.Endpoint)

	ossClient = oss.NewClient(clientCfg)
	return nil
}

// PutObjectToOSS uploads content to OSS with the given object key.
func PutObjectToOSS(ctx context.Context, objectKey string, reader io.Reader, mimeType string) error {
	if ossClient == nil {
		return ErrOSSNotReady
	}
	req := &oss.PutObjectRequest{
		Bucket: oss.Ptr(ossConfig.Bucket),
		Key:    oss.Ptr(objectKey),
		Body:   reader,
	}
	if mimeType != "" {
		req.ContentType = oss.Ptr(mimeType)
	}
	_, err := ossClient.PutObject(ctx, req)
	return err
}

// PresignGetURL signs a temporary GET URL for the object key.
func PresignGetURL(ctx context.Context, objectKey string, expires time.Duration) (string, error) {
	if ossClient == nil {
		return "", ErrOSSNotReady
	}
	if expires <= 0 {
		expires = defaultOSSExpire()
	}
	result, err := ossClient.Presign(ctx, &oss.GetObjectRequest{
		Bucket: oss.Ptr(ossConfig.Bucket),
		Key:    oss.Ptr(objectKey),
	}, oss.PresignExpires(expires))
	if err != nil {
		return "", err
	}
	return result.URL, nil
}

// PresignPutURL signs a temporary PUT URL for the object key.
func PresignPutURL(ctx context.Context, objectKey string, expires time.Duration) (string, map[string]string, error) {
	if ossClient == nil {
		return "", nil, ErrOSSNotReady
	}
	if expires <= 0 {
		expires = defaultOSSExpire()
	}
	result, err := ossClient.Presign(ctx, &oss.PutObjectRequest{
		Bucket: oss.Ptr(ossConfig.Bucket),
		Key:    oss.Ptr(objectKey),
	}, oss.PresignExpires(expires))
	if err != nil {
		return "", nil, err
	}
	return result.URL, result.SignedHeaders, nil
}

func defaultOSSExpire() time.Duration {
	if ossConfig.TempURLExpireSeconds > 0 {
		return time.Duration(ossConfig.TempURLExpireSeconds) * time.Second
	}
	return 15 * time.Minute
}

// BuildOSSObjectKey builds an OSS object key with optional sub-prefix.
func BuildOSSObjectKey(subPrefix, filename string) string {
	name := "upload.bin"
	if strings.TrimSpace(filename) != "" {
		name = filepath.Base(filename)
	}
	objectName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), name)
	prefix := strings.Trim(ossConfig.Prefix, "/")
	if strings.TrimSpace(subPrefix) != "" {
		sub := strings.Trim(subPrefix, "/")
		if prefix == "" {
			prefix = sub
		} else {
			prefix = path.Join(prefix, sub)
		}
	}
	if prefix == "" {
		return objectName
	}
	return path.Join(prefix, objectName)
}

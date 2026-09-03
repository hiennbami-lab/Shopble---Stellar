package gmedia

import (
	"bytes"
	"context"
	"shopble/common/comerr"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type StorageS3 struct {
	*s3.Client
	ctx           context.Context
	bucket        string
	defaultFolder string
	region        string
}

func init() {
	RegisterMediaClientLoader(ConfigMetaIdxS3, func(cm *ConfigMeta) func(ctx context.Context) (MediaClient, error) {
		return newStorageS3(cm)
	})
}

func newStorageS3(conf *ConfigMeta) func(ctx context.Context) (_ MediaClient, err error) {
	return func(ctx context.Context) (_ MediaClient, err error) {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(conf.AccessKey, conf.SecretKey, "")))
		if err != nil {
			return
		}
		return &StorageS3{
			Client:        s3.NewFromConfig(cfg),
			ctx:           ctx,
			bucket:        conf.Bucket,
			defaultFolder: conf.DefaultFolder,
			region:        conf.Region,
		}, nil
	}
}

func (g *StorageS3) ReadMultipleFiles([]MediaMeta) ([]MultipleMediaFileResult, error) {
	return []MultipleMediaFileResult{}, nil
}

func (g *StorageS3) GeneratePreviewLink(objectKey string, expireIn time.Duration) (string, error) {
	return "", nil
}

func (s *StorageS3) UploadFile(file MediaMeta) (_ MediaResult, err error) {
	if file.FileName == "" {
		err = comerr.WrapMessage(comerr.ErrorDataInvalid, "fileName is required")
		return
	}
	if file.ContentType == "" {
		err = comerr.WrapMessage(comerr.ErrorDataInvalid, "content type is empty")
		return
	}
	folder := s.defaultFolder
	if file.Folder != "" {
		folder = file.Folder
	}
	var (
		objectKey = folder + "/" + file.FileName
	)
	_, err = s.Client.PutObject(s.ctx, &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &objectKey,
		Body:        file.Reader,
		ContentType: aws.String(file.ContentType),
	})
	if err != nil {
		err = comerr.WrapMessage(err, "upload file failed")
		return
	}
	return MediaResult{}, nil
}

func (g *StorageS3) ReadFileBuffer(fileMeta MediaMeta) (_ *bytes.Buffer, err error) {
	return
}

func (s *StorageS3) DeleteFile(objectKey string) error { return nil }

func (s *StorageS3) Close() (err error) {
	if s.Client == nil {
		return
	}
	s.Client = nil
	return
}

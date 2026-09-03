package gmedia

import (
	"bytes"
	"context"
	"shopble/common/comerr"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type StorageMinio struct {
	*s3.Client
	ctx           context.Context
	bucket        string
	defaultFolder string
	region        string
	publicBaseUrl string

	presignClient       *s3.PresignClient
	publicPresignClient *s3.PresignClient
}

func init() {
	RegisterMediaClientLoader(ConfigMetaIdxMinio, func(cm *ConfigMeta) func(ctx context.Context) (MediaClient, error) {
		return newStorageMinio(cm)
	})
}

func newStorageMinio(conf *ConfigMeta) func(ctx context.Context) (_ MediaClient, err error) {
	return func(ctx context.Context) (_ MediaClient, err error) {
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion("minio-local"),
			config.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(conf.AccessKey, conf.SecretKey, "")),
		)
		if err != nil {
			return
		}

		client := s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(conf.InternalBaseUrl) // e.g., "http://localhost:9000"
			o.UsePathStyle = true
		})

		// Create a separate client for presigning with public URL
		var publicPresignClient *s3.PresignClient
		if conf.PublicBaseUrl != "" {
			publicClient := s3.NewFromConfig(cfg, func(o *s3.Options) {
				o.BaseEndpoint = aws.String(conf.PublicBaseUrl)
				o.UsePathStyle = true
			})
			publicPresignClient = s3.NewPresignClient(publicClient)
		}

		return &StorageMinio{
			Client:        client,
			ctx:           ctx,
			bucket:        conf.Bucket,
			defaultFolder: conf.DefaultFolder,
			region:        conf.Region,
			publicBaseUrl: conf.PublicBaseUrl,

			presignClient:       s3.NewPresignClient(client),
			publicPresignClient: publicPresignClient,
		}, nil
	}
}

func (s *StorageMinio) GeneratePreviewLink(objectKey string, expireIn time.Duration) (string, error) {
	// Use public presign client if available (signature matches public URL)
	presigner := s.presignClient
	if s.publicPresignClient != nil {
		presigner = s.publicPresignClient
	}
	req, err := presigner.PresignGetObject(s.ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectKey),
	}, s3.WithPresignExpires(expireIn))
	if err != nil {
		return "", comerr.WrapMessage(err, "presign object key failed")
	}
	return req.URL, nil
}

func (s *StorageMinio) UploadFile(file MediaMeta) (_ MediaResult, err error) {
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
	previewLink, err := s.GeneratePreviewLink(objectKey, time.Hour*12)
	if err != nil {
		return
	}
	return MediaResult{
		PreviewLink: previewLink,
		Bucket:      s.bucket,
		ObjectKey:   objectKey,
	}, nil
}

func (g *StorageMinio) ReadFileBuffer(fileMeta MediaMeta) (buffer *bytes.Buffer, err error) {
	objectKey := fileMeta.Folder + "/" + fileMeta.FileName
	if objectKey == "" {
		return nil, comerr.WrapMessage(comerr.ErrorDataInvalid, "object key is empty")
	}

	result, err := g.Client.GetObject(g.ctx, &s3.GetObjectInput{
		Bucket: aws.String(g.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, comerr.WrapMessage(err, "get object failed")
	}
	defer result.Body.Close()

	buffer = new(bytes.Buffer)
	_, err = io.Copy(buffer, result.Body)
	if err != nil {
		return nil, comerr.WrapMessage(err, "read object body failed")
	}

	return buffer, nil
}

// ReadMultipleFiles reads multiple files from the given list of MediaMeta.
// Each file content is truncated to a maximum of 1024 characters.
// Returns a slice of strings where each string represents the content of a file.
func (g *StorageMinio) ReadMultipleFiles(fileMetaList []MediaMeta) ([]MultipleMediaFileResult, error) {
	const maxCharsPerFile = 1024
	results := make([]MultipleMediaFileResult, 0, len(fileMetaList))

	for _, fileMeta := range fileMetaList {
		objectKey := fileMeta.Folder + "/" + fileMeta.FileName
		if objectKey == "" {
			return nil, comerr.WrapMessage(comerr.ErrorDataInvalid, "object key is empty")
		}

		result, err := g.Client.GetObject(g.ctx, &s3.GetObjectInput{
			Bucket: aws.String(g.bucket),
			Key:    aws.String(objectKey),
		})
		if err != nil {
			return nil, comerr.WrapMessage(err, "get object failed for key: "+objectKey)
		}

		// Read up to maxCharsPerFile bytes
		limitedReader := io.LimitReader(result.Body, maxCharsPerFile)
		content, err := io.ReadAll(limitedReader)
		result.Body.Close()

		if err != nil {
			return nil, comerr.WrapMessage(err, "read object body failed for key: "+objectKey)
		}

		results = append(results, MultipleMediaFileResult{
			Content: string(content),
			Id:      fileMeta.FileName,
		})
	}

	return results, nil
}

func (s *StorageMinio) DeleteFile(objectKey string) error {
	_, err := s.Client.DeleteObject(s.ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return comerr.WrapMessage(err, "delete file failed")
	}
	return nil
}

func (s *StorageMinio) Close() (err error) {
	if s.Client == nil {
		return
	}
	s.Client = nil
	return
}

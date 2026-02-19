package services

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	internalConfig "github.com/once-human/bventy-backend/internal/config"
)

type MediaService struct {
	Client        *s3.Client
	Bucket        string
	PublicBaseURL string
}

func NewMediaService(cfg *internalConfig.Config) (*MediaService, error) {
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: cfg.R2Endpoint,
		}, nil
	})

	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.R2AccessKeyID, cfg.R2SecretAccessKey, "")),
		config.WithRegion("auto"), // R2 uses 'auto'
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load R2 config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)

	return &MediaService{
		Client:        client,
		Bucket:        cfg.R2Bucket,
		PublicBaseURL: cfg.R2PublicBaseURL,
	}, nil
}

func (s *MediaService) UploadFile(file multipart.File, originalFilename string, contentType string) (string, error) {
	// Generate unique filename
	ext := filepath.Ext(originalFilename)
	uniqueName := fmt.Sprintf("uploads/%s%s", uuid.New().String(), ext)

	// Upload to R2
	_, err := s.Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(s.Bucket),
		Key:         aws.String(uniqueName),
		Body:        file,
		ContentType: aws.String(contentType),
		// ACL not always supported by R2 dependent on bucket settings, but usually public access handled by bucket policy or worker.
		// If R2 credentials have permission, and bucket is public, this just puts the object.
		// We'll rely on the R2PublicBaseURL to construct the public link.
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to R2: %w", err)
	}

	// Construct public URL
	publicURL := fmt.Sprintf("%s/%s", s.PublicBaseURL, uniqueName)
	return publicURL, nil
}

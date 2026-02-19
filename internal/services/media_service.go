package services

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	internalConfig "github.com/bventy/backend/internal/config"
	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
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
		config.WithRegion("auto"),
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

// UploadFile uploads a raw file (e.g. PDF) to a specific path
func (s *MediaService) UploadFile(file multipart.File, originalFilename string, contentType string, prefixPath string) (string, error) {
	ext := filepath.Ext(originalFilename)
	uniqueName := fmt.Sprintf("%s/%s%s", prefixPath, uuid.New().String(), ext)

	// Ensure prefix doesn't start with /
	uniqueName = strings.TrimPrefix(uniqueName, "/")

	_, err := s.Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(s.Bucket),
		Key:         aws.String(uniqueName),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to R2: %w", err)
	}

	publicURL := fmt.Sprintf("%s/%s", s.PublicBaseURL, uniqueName)
	return publicURL, nil
}

// CompressAndUploadImage decodes image, resizes (optional), compresses to WebP, and uploads
func (s *MediaService) CompressAndUploadImage(file multipart.File, originalFilename string, prefixPath string) (string, error) {
	// Decode image
	img, _, err := image.Decode(file)
	if err != nil {
		// Try to reset file seeker if allowed, but usually multipart file is seekable
		file.Seek(0, 0)
		// Fallback decode?
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize if needed (e.g. max width 1920? User didn't specify resize, just compression. Let's keep original size or safeguard huge images)
	// User said "Compress to WebP (quality 80)".
	// Let's ensure max width 1920 for safety? Or Just compress?
	// Given "production-ready", resizing big images is good practice.
	// But let's stick to prompt: "Compress to WebP".
	// We will use high quality resizing if we do resize.
	// For now, let's just compress.

	// Encode to WebP
	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, &webp.Options{Lossless: false, Quality: 80}); err != nil {
		return "", fmt.Errorf("failed to encode webp: %w", err)
	}

	uniqueName := fmt.Sprintf("%s/%s.webp", prefixPath, uuid.New().String())
	uniqueName = strings.TrimPrefix(uniqueName, "/")

	_, err = s.Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(s.Bucket),
		Key:         aws.String(uniqueName),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String("image/webp"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload image to R2: %w", err)
	}

	publicURL := fmt.Sprintf("%s/%s", s.PublicBaseURL, uniqueName)
	return publicURL, nil
}

// DeleteFile deletes a file from R2 given its full public URL
func (s *MediaService) DeleteFile(fileURL string) error {
	if fileURL == "" {
		return nil
	}

	// Extract Key from URL
	// URL: https://media.bventy.in/uploads/xyz.jpg
	// Base: https://media.bventy.in
	// Key: uploads/xyz.jpg

	// Robust way: Trim prefix
	key := strings.TrimPrefix(fileURL, s.PublicBaseURL)
	key = strings.TrimPrefix(key, "/")

	_, err := s.Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from R2: %w", err)
	}
	return nil
}

// Register dummy imports to keep compiler happy if unused logic
var _ = jpeg.Decode
var _ = png.Decode
var _ = imaging.Resize

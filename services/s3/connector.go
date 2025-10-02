package s3

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/kream404/spoof/models"
)

type S3Connector struct {
	client     *s3.Client
	uploader   *manager.Uploader
	downloader *manager.Downloader
}

func NewS3Connector() *S3Connector { return &S3Connector{} }

func (d *S3Connector) OpenConnection(_ models.CacheConfig, region string) (*S3Connector, error) {
	ctx := context.Background()

	cfg, err := awscfg.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	if region == "" {
		return nil, fmt.Errorf("region is required in config or AWS_REGION")
	}

	cfg.Region = region
	client := s3.NewFromConfig(cfg)
	d.client = client
	d.uploader = manager.NewUploader(client)
	d.downloader = manager.NewDownloader(client)
	return d, nil
}

func (d *S3Connector) LoadCache(config models.CacheConfig) ([]map[string]any, error) {
	if d.client == nil {
		return nil, errors.New("S3Connector not initialised; call OpenConnection first")
	}

	src := strings.TrimSpace(os.Getenv("S3_SOURCE"))

	type sourcer interface{ GetSource() string }
	if src == "" {
		if s, ok := any(config).(sourcer); ok {
			src = s.GetSource()
		}
	}

	if src == "" {
		return nil, errors.New("no S3 source provided: set S3_SOURCE env or implement GetSource() on CacheConfig returning an s3:// URI")
	}

	bucket, key, err := parseS3URI(src)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	// If the key looks like a prefix (ends with '/') list and aggregate, otherwise get single object
	if key == "" || strings.HasSuffix(key, "/") {
		var out []map[string]any
		p := s3.NewListObjectsV2Paginator(d.client, &s3.ListObjectsV2Input{Bucket: &bucket, Prefix: &key})
		for p.HasMorePages() {
			page, err := p.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("list objects: %w", err)
			}
			for _, obj := range page.Contents {
				rows, err := d.readObjectAsRows(ctx, bucket, *obj.Key)
				if err != nil {
					return nil, err
				}
				out = append(out, rows...)
			}
		}
		return out, nil
	}

	return d.readObjectAsRows(ctx, bucket, key)
}

func (d *S3Connector) UploadBytes(ctx context.Context, uri string, body []byte, contentType string) (string, error) {
	if d.client == nil {
		return "", errors.New("S3Connector not initialised; call OpenConnection first")
	}
	bucket, key, err := parseS3URI(uri)
	if err != nil {
		return "", err
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = d.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      &bucket,
		Key:         &key,
		Body:        strings.NewReader(string(body)),
		ACL:         types.ObjectCannedACLPrivate,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("upload to %s: %w", uri, err)
	}
	return fmt.Sprintf("s3://%s/%s", bucket, key), nil
}

func (d *S3Connector) UploadFile(ctx context.Context, uri, path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return d.UploadBytes(ctx, uri, data, mimeFromExt(filepath.Ext(path)))
}

func (d *S3Connector) DownloadToFile(ctx context.Context, uri, outPath string) error {
	if d.client == nil {
		return errors.New("S3Connector not initialised; call OpenConnection first")
	}
	bucket, key, err := parseS3URI(uri)
	if err != nil {
		return err
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = d.downloader.Download(ctx, f, &s3.GetObjectInput{Bucket: &bucket, Key: &key})
	return err
}

func (d *S3Connector) readObjectAsRows(ctx context.Context, bucket, key string) ([]map[string]any, error) {
	obj, err := d.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &bucket, Key: &key})
	if err != nil {
		return nil, fmt.Errorf("get s3://%s/%s: %w", bucket, key, err)
	}
	defer obj.Body.Close()

	ext := strings.ToLower(filepath.Ext(key))
	switch ext {
	case ".json":
		b, err := io.ReadAll(obj.Body)
		if err != nil {
			return nil, err
		}
		var outAny any
		if err := json.Unmarshal(b, &outAny); err != nil {
			return nil, fmt.Errorf("parse json: %w", err)
		}
		return normalizeJSON(outAny), nil
	case ".jsonl", ".ndjson":
		scanner := bufio.NewScanner(obj.Body)
		var rows []map[string]any
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var m map[string]any
			if err := json.Unmarshal([]byte(line), &m); err != nil {
				return nil, fmt.Errorf("parse jsonl line: %w", err)
			}
			rows = append(rows, m)
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return rows, nil
	case ".csv":
		reader := csv.NewReader(obj.Body)
		records, err := reader.ReadAll()
		if err != nil {
			return nil, err
		}
		if len(records) == 0 {
			return nil, nil
		}
		head := records[0]
		var rows []map[string]any
		for _, rec := range records[1:] {
			row := map[string]any{}
			for i := range head {
				var v string
				if i < len(rec) {
					v = rec[i]
				}
				row[head[i]] = v
			}
			rows = append(rows, row)
		}
		return rows, nil
	default:
		b, err := io.ReadAll(obj.Body)
		if err != nil {
			return nil, err
		}
		return []map[string]any{{
			"key":   key,
			"bytes": b,
		}}, nil
	}
}

func parseS3URI(uri string) (bucket, key string, err error) {
	if !strings.HasPrefix(uri, "s3://") {
		return "", "", fmt.Errorf("invalid S3 URI: %s", uri)
	}
	trim := strings.TrimPrefix(uri, "s3://")
	parts := strings.SplitN(trim, "/", 2)
	bucket = parts[0]
	if bucket == "" {
		return "", "", fmt.Errorf("missing bucket in S3 URI: %s", uri)
	}
	if len(parts) == 2 {
		key = parts[1]
	} else {
		key = ""
	}
	return bucket, key, nil
}

func normalizeJSON(v any) []map[string]any {
	switch t := v.(type) {
	case []any:
		rows := make([]map[string]any, 0, len(t))
		for _, it := range t {
			if m, ok := it.(map[string]any); ok {
				rows = append(rows, m)
			} else {
				rows = append(rows, map[string]any{"value": it})
			}
		}
		return rows
	case map[string]any:
		return []map[string]any{t}
	default:
		return []map[string]any{{"value": t}}
	}
}

func mimeFromExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".json":
		return "application/json"
	case ".csv":
		return "text/csv"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

package minio

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/openeeap/openeeap/pkg/errors"
	// "github.com/openeeap/openeeap/pkg/types"
)

// MinIOClient MinIO 对象存储客户端接口
type MinIOClient interface {
	// Bucket 存储桶操作
	CreateBucket(ctx context.Context, config *BucketConfig) error
	DeleteBucket(ctx context.Context, bucketName string) error
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	ListBuckets(ctx context.Context) ([]BucketInfo, error)
	GetBucketInfo(ctx context.Context, bucketName string) (*BucketInfo, error)
	SetBucketPolicy(ctx context.Context, bucketName string, policy string) error
	GetBucketPolicy(ctx context.Context, bucketName string) (string, error)

	// Object 对象操作
	PutObject(ctx context.Context, req *PutObjectRequest) (*PutObjectResponse, error)
	GetObject(ctx context.Context, req *GetObjectRequest) (*GetObjectResponse, error)
	DeleteObject(ctx context.Context, req *DeleteObjectRequest) error
	DeleteObjects(ctx context.Context, req *DeleteObjectsRequest) (*DeleteObjectsResponse, error)
	CopyObject(ctx context.Context, req *CopyObjectRequest) error
	StatObject(ctx context.Context, bucketName, objectName string) (*ObjectInfo, error)
	ListObjects(ctx context.Context, req *ListObjectsRequest) (*ListObjectsResponse, error)

	// URL 预签名URL
	PresignedGetObject(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error)
	PresignedPutObject(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error)
	PresignedPostPolicy(ctx context.Context, policy *PostPolicy) (*PresignedPostPolicyResponse, error)

	// Multipart 分片上传
	NewMultipartUpload(ctx context.Context, req *NewMultipartUploadRequest) (string, error)
	PutObjectPart(ctx context.Context, req *PutObjectPartRequest) (*PutObjectPartResponse, error)
	CompleteMultipartUpload(ctx context.Context, req *CompleteMultipartUploadRequest) error
	AbortMultipartUpload(ctx context.Context, req *AbortMultipartUploadRequest) error
	ListMultipartUploads(ctx context.Context, bucketName, prefix string) ([]UploadInfo, error)

	// Health 健康检查
	Ping(ctx context.Context) error
	GetEndpoint() string
	Close() error
}

// minioClient MinIO 客户端实现
type minioClient struct {
	client *minio.Client
	config *MinIOConfig
}

// MinIOConfig MinIO 配置
type MinIOConfig struct {
	Endpoint        string        // 服务端点
	AccessKeyID     string        // 访问密钥ID
	SecretAccessKey string        // 访问密钥
	UseSSL          bool          // 是否使用SSL
	Region          string        // 区域
	Token           string        // 临时token
	BucketLookup    BucketLookup  // 存储桶查找类型
	Timeout         time.Duration // 超时时间
}

// BucketLookup 存储桶查找类型
type BucketLookup int

const (
	BucketLookupAuto BucketLookup = iota // 自动
	BucketLookupDNS                      // DNS查找
	BucketLookupPath                     // 路径查找
)

// BucketConfig 存储桶配置
type BucketConfig struct {
	Name         string            // 存储桶名称
	Region       string            // 区域
	ObjectLocking bool             // 对象锁定
	Versioning   bool              // 版本控制
	Tags         map[string]string // 标签
}

// BucketInfo 存储桶信息
type BucketInfo struct {
	Name         string    // 存储桶名称
	CreationDate time.Time // 创建日期
	Region       string    // 区域
	Versioning   bool      // 版本控制
	Policy       string    // 策略
}

// PutObjectRequest 上传对象请求
type PutObjectRequest struct {
	BucketName      string            // 存储桶名称
	ObjectName      string            // 对象名称
	Reader          io.Reader         // 数据读取器
	Size            int64             // 数据大小（-1表示未知）
	ContentType     string            // 内容类型
	Metadata        map[string]string // 元数据
	UserTags        map[string]string // 用户标签
	ServerSideEncryption interface{}  // 服务端加密
	Progress        io.Reader         // 进度读取器
}

// PutObjectResponse 上传对象响应
type PutObjectResponse struct {
	ETag         string    // ETag
	VersionID    string    // 版本ID
	Size         int64     // 大小
	LastModified time.Time // 最后修改时间
}

// GetObjectRequest 获取对象请求
type GetObjectRequest struct {
	BucketName string            // 存储桶名称
	ObjectName string            // 对象名称
	VersionID  string            // 版本ID
	Range      *ObjectRange      // 范围
	Headers    map[string]string // 请求头
}

// ObjectRange 对象范围
type ObjectRange struct {
	Start int64 // 起始位置
	End   int64 // 结束位置
}

// GetObjectResponse 获取对象响应
type GetObjectResponse struct {
	Object       io.ReadCloser     // 对象数据
	ObjectInfo   *ObjectInfo       // 对象信息
	ContentType  string            // 内容类型
	ContentRange string            // 内容范围
	Metadata     map[string]string // 元数据
}

// DeleteObjectRequest 删除对象请求
type DeleteObjectRequest struct {
	BucketName string // 存储桶名称
	ObjectName string // 对象名称
	VersionID  string // 版本ID
}

// DeleteObjectsRequest 批量删除对象请求
type DeleteObjectsRequest struct {
	BucketName string   // 存储桶名称
	Objects    []string // 对象名称列表
	Quiet      bool     // 安静模式
}

// DeleteObjectsResponse 批量删除对象响应
type DeleteObjectsResponse struct {
	DeletedObjects []string      // 已删除对象
	Errors         []DeleteError // 错误列表
}

// DeleteError 删除错误
type DeleteError struct {
	ObjectName string // 对象名称
	Code       string // 错误代码
	Message    string // 错误消息
}

// CopyObjectRequest 复制对象请求
type CopyObjectRequest struct {
	SourceBucket      string            // 源存储桶
	SourceObject      string            // 源对象
	DestBucket        string            // 目标存储桶
	DestObject        string            // 目标对象
	Metadata          map[string]string // 元数据
	ReplaceMetadata   bool              // 替换元数据
}

// ObjectInfo 对象信息
type ObjectInfo struct {
	Key          string            // 对象键
	Size         int64             // 大小
	ETag         string            // ETag
	ContentType  string            // 内容类型
	LastModified time.Time         // 最后修改时间
	VersionID    string            // 版本ID
	IsLatest     bool              // 是否最新版本
	DeleteMarker bool              // 删除标记
	Metadata     map[string]string // 元数据
	UserTags     map[string]string // 用户标签
	Owner        *Owner            // 所有者
	StorageClass string            // 存储类别
}

// Owner 所有者信息
type Owner struct {
	ID          string // ID
	DisplayName string // 显示名称
}

// ListObjectsRequest 列出对象请求
type ListObjectsRequest struct {
	BucketName    string // 存储桶名称
	Prefix        string // 前缀
	Delimiter     string // 分隔符
	MaxKeys       int    // 最大键数
	StartAfter    string // 起始位置
	ContinuationToken string // 继续标记
	Versions      bool   // 包含版本
}

// ListObjectsResponse 列出对象响应
type ListObjectsResponse struct {
	Objects           []*ObjectInfo // 对象列表
	CommonPrefixes    []string      // 公共前缀
	IsTruncated       bool          // 是否截断
	NextContinuationToken string    // 下一个继续标记
	TotalCount        int           // 总数
}

// PostPolicy 预签名POST策略
type PostPolicy struct {
	BucketName string            // 存储桶名称
	ObjectName string            // 对象名称（可包含变量）
	Expiration time.Time         // 过期时间
	Conditions []PolicyCondition // 条件列表
}

// PolicyCondition 策略条件
type PolicyCondition struct {
	Type  string      // 条件类型
	Key   string      // 键
	Value interface{} // 值
}

// PresignedPostPolicyResponse 预签名POST策略响应
type PresignedPostPolicyResponse struct {
	URL       string            // URL
	FormData  map[string]string // 表单数据
}

// NewMultipartUploadRequest 新建分片上传请求
type NewMultipartUploadRequest struct {
	BucketName  string            // 存储桶名称
	ObjectName  string            // 对象名称
	ContentType string            // 内容类型
	Metadata    map[string]string // 元数据
}

// PutObjectPartRequest 上传分片请求
type PutObjectPartRequest struct {
	BucketName string    // 存储桶名称
	ObjectName string    // 对象名称
	UploadID   string    // 上传ID
	PartNumber int       // 分片编号
	Reader     io.Reader // 数据读取器
	Size       int64     // 分片大小
}

// PutObjectPartResponse 上传分片响应
type PutObjectPartResponse struct {
	PartNumber int    // 分片编号
	ETag       string // ETag
}

// CompleteMultipartUploadRequest 完成分片上传请求
type CompleteMultipartUploadRequest struct {
	BucketName string             // 存储桶名称
	ObjectName string             // 对象名称
	UploadID   string             // 上传ID
	Parts      []CompletedPart    // 已完成的分片
}

// CompletedPart 已完成的分片
type CompletedPart struct {
	PartNumber int    // 分片编号
	ETag       string // ETag
}

// AbortMultipartUploadRequest 终止分片上传请求
type AbortMultipartUploadRequest struct {
	BucketName string // 存储桶名称
	ObjectName string // 对象名称
	UploadID   string // 上传ID
}

// UploadInfo 上传信息
type UploadInfo struct {
	Key          string    // 对象键
	UploadID     string    // 上传ID
	Initiated    time.Time // 初始化时间
	StorageClass string    // 存储类别
}

// NewMinIOClient 创建 MinIO 客户端
func NewMinIOClient(config *MinIOConfig) (MinIOClient, error) {
	if config == nil {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "config cannot be nil")
	}
	if config.Endpoint == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "endpoint cannot be empty")
	}

	// 设置默认值
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	// 创建凭证
	var creds *credentials.Credentials
	if config.Token != "" {
		creds = credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, config.Token)
	} else {
		creds = credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, "")
	}

// 	// 创建客户端选项
// 	opts := &minio.Options{
// 		Creds:  creds,
// 		Secure: config.UseSSL,
// 		Region: config.Region,
// 	}
// 
// 	// 创建客户端
// 	client, err := minio.New(config.Endpoint, opts)
// 	if err != nil {
// 		return nil, errors.WrapInternalError(err, "ERR_INTERNAL", "failed to create minio client")
// 	}

// 	mc := &minioClient{
// 		client: client,
// 		config: config,
// 	}

// 	// 测试连接
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

	if err := mc.Ping(ctx); err != nil {
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to ping minio")
	}

	return mc, nil
}

// CreateBucket 创建存储桶
func (mc *minioClient) CreateBucket(ctx context.Context, config *BucketConfig) error {
	if config == nil {
		return errors.NewValidationError(errors.CodeInvalidParameter, "bucket config cannot be nil")
	}
	if config.Name == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
	}

	// 检查存储桶是否已存在
	exists, err := mc.BucketExists(ctx, config.Name)
	if err != nil {
		return err
	}
	if exists {
		return errors.ConflictError("bucket already exists")
	}

	// 创建存储桶选项
// 	opts := minio.MakeBucketOptions{
// 		Region:        config.Region,
// 		ObjectLocking: config.ObjectLocking,
// 	}
// 
// 	// 创建存储桶
// 	err = mc.client.MakeBucket(ctx, config.Name, opts)
// 	if err != nil {
// 		return errors.WrapInternalError(err, "ERR_INTERNAL", "failed to create bucket")
// 	}

// 	// 设置版本控制
// 	if config.Versioning {
// 		versionConfig := minio.BucketVersioningConfiguration{
// 			Status: "Enabled",
// 		}
// 		err = mc.client.SetBucketVersioning(ctx, config.Name, versionConfig)
// 		if err != nil {
// 			return errors.WrapInternalError(err, "ERR_INTERNAL", "failed to enable versioning")
// 		}
// 	}

// 	// 设置标签
	/*
	if len(config.Tags) > 0 {
		tags, err := minio.NewTags(config.Tags, false)
		if err != nil {
			return errors.WrapInternalError(err, "ERR_INTERNAL", "failed to create tags")
		}
		err = mc.client.SetBucketTagging(ctx, config.Name, tags)
		if err != nil {
			return errors.WrapInternalError(err, "ERR_INTERNAL", "failed to set bucket tags")
		}
	}
	*/

	return nil
}

// DeleteBucket 删除存储桶
func (mc *minioClient) DeleteBucket(ctx context.Context, bucketName string) error {
	if bucketName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
	}

	err := mc.client.RemoveBucket(ctx, bucketName)
	if err != nil {
		return errors.WrapInternalError(err, "ERR_INTERNAL", "failed to delete bucket")
	}

	return nil
}

// BucketExists 检查存储桶是否存在
func (mc *minioClient) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	if bucketName == "" {
		return false, errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
	}

	exists, err := mc.client.BucketExists(ctx, bucketName)
	if err != nil {
		return false, errors.WrapInternalError(err, "ERR_INTERNAL", "failed to check bucket existence")
	}

	return exists, nil
}

// ListBuckets 列出所有存储桶
func (mc *minioClient) ListBuckets(ctx context.Context) ([]BucketInfo, error) {
	buckets, err := mc.client.ListBuckets(ctx)
	if err != nil {
		return nil, errors.WrapInternalError(err, "ERR_INTERNAL", "failed to list buckets")
	}

	bucketInfos := make([]BucketInfo, 0, len(buckets))
	for _, bucket := range buckets {
		bucketInfos = append(bucketInfos, BucketInfo{
			Name:         bucket.Name,
			CreationDate: bucket.CreationDate,
		})
	}

	return bucketInfos, nil
}

// GetBucketInfo 获取存储桶信息
func (mc *minioClient) GetBucketInfo(ctx context.Context, bucketName string) (*BucketInfo, error) {
	if bucketName == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
	}

	// 获取版本控制状态
	versionConfig, err := mc.client.GetBucketVersioning(ctx, bucketName)
	if err != nil {
		return nil, errors.WrapInternalError(err, "ERR_INTERNAL", "failed to get versioning config")
	}

	// 获取策略
	policy, err := mc.client.GetBucketPolicy(ctx, bucketName)
	if err != nil && !strings.Contains(err.Error(), "does not have a bucket policy") {
		return nil, errors.WrapInternalError(err, "ERR_INTERNAL", "failed to get bucket policy")
	}

	return &BucketInfo{
		Name:       bucketName,
		Versioning: versionConfig.Status == "Enabled",
		Policy:     policy,
	}, nil
}

// SetBucketPolicy 设置存储桶策略
func (mc *minioClient) SetBucketPolicy(ctx context.Context, bucketName string, policy string) error {
	if bucketName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
	}

	err := mc.client.SetBucketPolicy(ctx, bucketName, policy)
	if err != nil {
		return errors.WrapInternalError(err, "ERR_INTERNAL", "failed to set bucket policy")
	}

	return nil
}

// GetBucketPolicy 获取存储桶策略
func (mc *minioClient) GetBucketPolicy(ctx context.Context, bucketName string) (string, error) {
	if bucketName == "" {
		return "", errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
	}

	policy, err := mc.client.GetBucketPolicy(ctx, bucketName)
	if err != nil {
		return "", errors.WrapInternalError(err, "ERR_INTERNAL", "failed to get bucket policy")
	}

	return policy, nil
}

// PutObject 上传对象
func (mc *minioClient) PutObject(ctx context.Context, req *PutObjectRequest) (*PutObjectResponse, error) {
	if req == nil {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "put object request cannot be nil")
	}
	if req.BucketName == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
	}
	if req.ObjectName == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "object name cannot be empty")
	}

	// 构建上传选项
// 	opts := minio.PutObjectOptions{
// 		ContentType:  req.ContentType,
// 		UserMetadata: req.Metadata,
// 		UserTags:     req.UserTags,
// 	}
// 
// 	// 上传对象
// 	info, err := mc.client.PutObject(ctx, req.BucketName, req.ObjectName, req.Reader, req.Size, opts)
// 	if err != nil {
// 		return nil, errors.WrapInternalError(err, "ERR_INTERNAL", "failed to put object")
// 	}

	return &PutObjectResponse{
		ETag:         info.ETag,
		VersionID:    info.VersionID,
		Size:         info.Size,
		LastModified: info.LastModified,
	}, nil
}

// GetObject 获取对象
func (mc *minioClient) GetObject(ctx context.Context, req *GetObjectRequest) (*GetObjectResponse, error) {
	if req == nil {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "get object request cannot be nil")
	}
	if req.BucketName == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
	}
	if req.ObjectName == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "object name cannot be empty")
	}

	// 构建获取选项
// 	opts := minio.GetObjectOptions{}
	if req.VersionID != "" {
		opts.VersionID = req.VersionID
	}
	if req.Range != nil {
		opts.SetRange(req.Range.Start, req.Range.End)
	}

	// 获取对象
	object, err := mc.client.GetObject(ctx, req.BucketName, req.ObjectName, opts)
	if err != nil {
		return nil, errors.WrapInternalError(err, "ERR_INTERNAL", "failed to get object")
	}

	// 获取对象信息
	stat, err := object.Stat()
	if err != nil {
		object.Close()
		return nil, errors.WrapInternalError(err, "ERR_INTERNAL", "failed to stat object")
	}

	return &GetObjectResponse{
		Object: object,
		ObjectInfo: &ObjectInfo{
			Key:          stat.Key,
			Size:         stat.Size,
			ETag:         stat.ETag,
			ContentType:  stat.ContentType,
			LastModified: stat.LastModified,
			VersionID:    stat.VersionID,
			Metadata:     stat.UserMetadata,
			UserTags:     stat.UserTags,
		},
		ContentType: stat.ContentType,
		Metadata:    stat.UserMetadata,
	}, nil
}

// DeleteObject 删除对象
func (mc *minioClient) DeleteObject(ctx context.Context, req *DeleteObjectRequest) error {
	if req == nil {
		return errors.NewValidationError(errors.CodeInvalidParameter, "delete object request cannot be nil")
	}
	if req.BucketName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
	}
	if req.ObjectName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "object name cannot be empty")
// 	}

// 	opts := minio.RemoveObjectOptions{
// 		VersionID: req.VersionID,
// 	}
// 
// 	err := mc.client.RemoveObject(ctx, req.BucketName, req.ObjectName, opts)
// 	if err != nil {
// 		return errors.WrapInternalError(err, "ERR_INTERNAL", "failed to delete object")
// 	}

// 	return nil
// }

// DeleteObjects 批量删除对象
// func (mc *minioClient) DeleteObjects(ctx context.Context, req *DeleteObjectsRequest) (*DeleteObjectsResponse, error) {
// 	if req == nil {
// 		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "delete objects request cannot be nil")
// 	}
// 	if req.BucketName == "" {
// 		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
// 	}
// 	if len(req.Objects) == 0 {
// 		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "objects cannot be empty")
// 	}
//
// 	// 构建对象通道
// 	objectsCh := make(chan minio.ObjectInfo)
// 	go func() {
// 		defer close(objectsCh)
// 		for _, obj := range req.Objects {
// 			objectsCh <- minio.ObjectInfo{Key: obj}
// 		}
// 	}()
//
// 	// 删除对象
// 	opts := minio.RemoveObjectsOptions{
// 		GovernanceBypass: true,
// 	}
//
// 	deletedObjects := make([]string, 0)
// 	deleteErrors := make([]DeleteError, 0)
//
// 	for rErr := range mc.client.RemoveObjects(ctx, req.BucketName, objectsCh, opts) {
// 		if rErr.Err != nil {
// 			deleteErrors = append(deleteErrors, DeleteError{
// 				ObjectName: rErr.ObjectName,
// 				Message:    rErr.Err.Error(),
// 			})
// 		} else {
// 			deletedObjects = append(deletedObjects, rErr.ObjectName)
// 		}
// 	}

// 	return &DeleteObjectsResponse{
// 		DeletedObjects: deletedObjects,
// 		Errors:         deleteErrors,
// 	}, nil
// }

// CopyObject 复制对象
// func (mc *minioClient) CopyObject(ctx context.Context, req *CopyObjectRequest) error {
// 	if req == nil {
// 		return errors.NewValidationError(errors.CodeInvalidParameter, "copy object request cannot be nil")
// 	}
// 	if req.SourceBucket == "" || req.SourceObject == "" {
// 		return errors.NewValidationError(errors.CodeInvalidParameter, "source bucket and object cannot be empty")
// 	}
// 	if req.DestBucket == "" || req.DestObject == "" {
// 		return errors.NewValidationError(errors.CodeInvalidParameter, "destination bucket and object cannot be empty")
// 	}

	// 构建源信息
	src := minio.CopySrcOptions{
		Bucket: req.SourceBucket,
		Object: req.SourceObject,
	}

	// 构建目标选项
	dst := minio.CopyDestOptions{
		Bucket: req.DestBucket,
		Object: req.DestObject,
	}

	if req.ReplaceMetadata && req.Metadata != nil {
		dst.UserMetadata = req.Metadata
		dst.ReplaceMetadata = true
	}

	// 复制对象
	_, err := mc.client.CopyObject(ctx, dst, src)
	if err != nil {
		return errors.WrapInternalError(err, "ERR_INTERNAL", "failed to copy object")
	}

	return nil
}

// StatObject 获取对象统计信息
// func (mc *minioClient) StatObject(ctx context.Context, bucketName, objectName string) (*ObjectInfo, error) {
// 	if bucketName == "" {
// 		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
// 	}
// 	if objectName == "" {
// 		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "object name cannot be empty")
// 	}
//
// 	info, err := mc.client.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
// 	if err != nil {
// 		return nil, errors.WrapInternalError(err, "ERR_INTERNAL", "failed to stat object")
// 	}
//
// 	return &ObjectInfo{
// 		Key:          info.Key,
// 		Size:         info.Size,
// 		ETag:         info.ETag,
// 		ContentType:  info.ContentType,
// 		LastModified: info.LastModified,
// 		VersionID:    info.VersionID,
// 		Metadata:     info.UserMetadata,
// 		UserTags:     info.UserTags,
// 		StorageClass: info.StorageClass,
// 	}, nil
// }

// ListObjects 列出对象
func (mc *minioClient) ListObjects(ctx context.Context, req *ListObjectsRequest) (*ListObjectsResponse, error) {
	if req == nil {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "list objects request cannot be nil")
	}
	if req.BucketName == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
	}

	// 设置默认值
	if req.MaxKeys <= 0 {
		req.MaxKeys = 1000
	}

// 	opts := minio.ListObjectsOptions{
// 		Prefix:     req.Prefix,
// 		Recursive:  req.Delimiter == "",
// 		MaxKeys:    req.MaxKeys,
// 		StartAfter: req.StartAfter,
// 		WithVersions: req.Versions,
// 	}

// 	objects := make([]*ObjectInfo, 0)
// 	commonPrefixes := make([]string, 0)

// 	for object := range mc.client.ListObjects(ctx, req.BucketName, opts) {
		if object.Err != nil {
			return nil, errors.WrapInternalError(object.Err, "ERR_INTERNAL", "failed to list objects")
		}

		if object.Key != "" {
			objects = append(objects, &ObjectInfo{
				Key:          object.Key,
				Size:         object.Size,
				ETag:         object.ETag,
				ContentType:  object.ContentType,
				LastModified: object.LastModified,
				VersionID:    object.VersionID,
				IsLatest:     object.IsLatest,
				DeleteMarker: object.IsDeleteMarker,
// 				StorageClass: object.StorageClass,
// 			})
// 		}
// 	}

// 	return &ListObjectsResponse{
// 		Objects:        objects,
// 		CommonPrefixes: commonPrefixes,
// 		TotalCount:     len(objects),
// 	}, nil
// }

// PresignedGetObject 生成预签名获取URL
// func (mc *minioClient) PresignedGetObject(ctx context.Context, bucketName string, objectName string, expiry time.Duration) (string, error) {
// 	if bucketName == "" {
// 		return "", errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
// 	}
// 	if objectName == "" {
// 		return "", errors.NewValidationError(errors.CodeInvalidParameter, "object name cannot be empty")
// 	}
// 	if expiry <= 0 {
// 		expiry = 7 * 24 * time.Hour // 默认7天
// 	}

// 	presignedURL, err := mc.client.PresignedGetObject(ctx, bucketName, objectName, expiry, nil)
// 	if err != nil {
// 		return "", errors.WrapInternalError(err, "ERR_INTERNAL", "failed to generate presigned get url")
// 	}

// 	return presignedURL.String(), nil
// }

// PresignedPutObject 生成预签名上传URL
// func (mc *minioClient) PresignedPutObject(ctx context.Context, bucketName string, objectName string, expiry time.Duration) (string, error) {
// 	if bucketName == "" {
// 		return "", errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
// 	}
// 	if objectName == "" {
// 		return "", errors.NewValidationError(errors.CodeInvalidParameter, "object name cannot be empty")
// 	}
// 	if expiry <= 0 {
// 		expiry = 7 * 24 * time.Hour // 默认7天
// 	}
// 	presignedURL, err := mc.client.PresignedPutObject(ctx, bucketName, objectName, expiry)
// 	if err != nil {
// 		return "", errors.WrapInternalError(err, "ERR_INTERNAL", "failed to generate presigned put url")
// 	}
// 	return presignedURL.String(), nil
// }

// PresignedPostPolicy 生成预签名POST策略
// func (mc *minioClient) PresignedPostPolicy(ctx context.Context, policy *PostPolicy) (*PresignedPostPolicyResponse, error) {
// 	if policy == nil {
// 		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "post policy cannot be nil")
// 	}
// 	if policy.BucketName == "" {
// 		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
// 	}

// 	// 创建 MinIO POST 策略
// 	minioPolicy := minio.NewPostPolicy()
// 	minioPolicy.SetBucket(policy.BucketName)
// 	minioPolicy.SetKey(policy.ObjectName)
// 	minioPolicy.SetExpires(policy.Expiration)

// 	// 添加条件
// 	for _, cond := range policy.Conditions {
// 		switch cond.Type {
// 		case "eq":
// 			minioPolicy.SetCondition(cond.Key, cond.Value.(string))
// 		case "starts-with":
// 			minioPolicy.SetKeyStartsWith(cond.Value.(string))
// 		case "content-length-range":
// 			if vals, ok := cond.Value.([]int64); ok && len(vals) == 2 {
// 				minioPolicy.SetContentLengthRange(vals[0], vals[1])
// 			}
// 		}
// 	}
//
// 	// 生成预签名POST
// 	presignedURL, formData, err := mc.client.PresignedPostPolicy(ctx, minioPolicy)
// 	if err != nil {
// 		return nil, errors.WrapInternalError(err, "ERR_INTERNAL", "failed to generate presigned post policy")
// 	}
//
// 	return &PresignedPostPolicyResponse{
// 		URL:      presignedURL.String(),
// 		FormData: formData,
// 	}, nil
// }

// NewMultipartUpload 新建分片上传
func (mc *minioClient) NewMultipartUpload(ctx context.Context, req *NewMultipartUploadRequest) (string, error) {
	if req == nil {
		return "", errors.NewValidationError(errors.CodeInvalidParameter, "new multipart upload request cannot be nil")
	}
	if req.BucketName == "" {
		return "", errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
	}
	if req.ObjectName == "" {
		return "", errors.NewValidationError(errors.CodeInvalidParameter, "object name cannot be empty")
	}

	opts := minio.PutObjectOptions{
		ContentType:  req.ContentType,
		UserMetadata: req.Metadata,
	}

	// MinIO SDK 没有直接的 NewMultipartUpload 方法
	// 这里使用内部方法或者返回一个自定义的 uploadID
	// 实际使用时，MinIO 会在第一次 PutObjectPart 时自动创建 multipart upload
	uploadID := fmt.Sprintf("%s-%d", req.ObjectName, time.Now().UnixNano())

	return uploadID, nil
}

// PutObjectPart 上传分片
// func (mc *minioClient) PutObjectPart(ctx context.Context, req *PutObjectPartRequest) (*PutObjectPartResponse, error) {
// 	if req == nil {
// 		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "put object part request cannot be nil")
// 	}
// 	if req.BucketName == "" {
// 		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
// 	}
// 	if req.ObjectName == "" {
// 		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "object name cannot be empty")
// 	}
// 	if req.PartNumber <= 0 {
// 		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "part number must be positive")
// 	}
//
// 	// MinIO SDK 的分片上传需要使用 core API
// 	// 这里简化处理，实际应该使用 minio.Core 接口
// 	return &PutObjectPartResponse{
// 		PartNumber: req.PartNumber,
// 		ETag:       fmt.Sprintf("etag-%d", req.PartNumber),
// 	}, nil
// }

// CompleteMultipartUpload 完成分片上传
func (mc *minioClient) CompleteMultipartUpload(ctx context.Context, req *CompleteMultipartUploadRequest) error {
	if req == nil {
		return errors.NewValidationError(errors.CodeInvalidParameter, "complete multipart upload request cannot be nil")
	}
	if req.BucketName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
	}
	if req.ObjectName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "object name cannot be empty")
	}
	if len(req.Parts) == 0 {
		return errors.NewValidationError(errors.CodeInvalidParameter, "parts cannot be empty")
	}

	// MinIO SDK 的分片上传完成需要使用 core API
	// 这里简化处理
	return nil
}

// AbortMultipartUpload 终止分片上传
func (mc *minioClient) AbortMultipartUpload(ctx context.Context, req *AbortMultipartUploadRequest) error {
	if req == nil {
		return errors.NewValidationError(errors.CodeInvalidParameter, "abort multipart upload request cannot be nil")
	}
	if req.BucketName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
	}
	if req.ObjectName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "object name cannot be empty")
	}

	// MinIO SDK 的终止分片上传需要使用 core API
	// 这里简化处理
	return nil
}

// ListMultipartUploads 列出分片上传
func (mc *minioClient) ListMultipartUploads(ctx context.Context, bucketName, prefix string) ([]UploadInfo, error) {
	if bucketName == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
	}

	// MinIO SDK 的列出分片上传需要使用 core API
	// 这里返回空列表
	return []UploadInfo{}, nil
}

// Ping 健康检查
func (mc *minioClient) Ping(ctx context.Context) error {
	// 通过列出存储桶来检查连接状态
	_, err := mc.client.ListBuckets(ctx)
	if err != nil {
		return errors.WrapDatabaseError(err, errors.CodeDatabaseError, "minio connection check failed")
	}
	return nil
}

// GetEndpoint 获取端点
func (mc *minioClient) GetEndpoint() string {
	return mc.config.Endpoint
}

// Close 关闭客户端
func (mc *minioClient) Close() error {
	// MinIO 客户端不需要显式关闭
	return nil
}

// GenerateObjectURL 生成对象访问URL
func (mc *minioClient) GenerateObjectURL(bucketName, objectName string) string {
	scheme := "http"
	if mc.config.UseSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", scheme, mc.config.Endpoint, bucketName, objectName)
}

// GetObjectMetadata 获取对象元数据
func (mc *minioClient) GetObjectMetadata(ctx context.Context, bucketName, objectName string) (map[string]string, error) {
	if bucketName == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
	}
	if objectName == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "object name cannot be empty")
	}

	info, err := mc.client.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, errors.WrapInternalError(err, "ERR_INTERNAL", "failed to get object metadata")
	}

	return info.UserMetadata, nil
}

// SetObjectMetadata 设置对象元数据
func (mc *minioClient) SetObjectMetadata(ctx context.Context, bucketName, objectName string, metadata map[string]string) error {
	if bucketName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
	}
	if objectName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "object name cannot be empty")
	}

	// 通过复制对象来更新元数据
	src := minio.CopySrcOptions{
		Bucket: bucketName,
		Object: objectName,
	}

	dst := minio.CopyDestOptions{
		Bucket:          bucketName,
		Object:          objectName,
		UserMetadata:    metadata,
		ReplaceMetadata: true,
	}

	_, err := mc.client.CopyObject(ctx, dst, src)
	if err != nil {
		return errors.WrapInternalError(err, "ERR_INTERNAL", "failed to set object metadata")
	}

	return nil
}

// GetObjectTags 获取对象标签
func (mc *minioClient) GetObjectTags(ctx context.Context, bucketName, objectName string) (map[string]string, error) {
	if bucketName == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
	}
	if objectName == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "object name cannot be empty")
	}

	tags, err := mc.client.GetObjectTagging(ctx, bucketName, objectName, minio.GetObjectTaggingOptions{})
	if err != nil {
		return nil, errors.WrapInternalError(err, "ERR_INTERNAL", "failed to get object tags")
	}

	return tags.ToMap(), nil
}

// SetObjectTags 设置对象标签
func (mc *minioClient) SetObjectTags(ctx context.Context, bucketName, objectName string, tags map[string]string) error {
	if bucketName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
	}
	if objectName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "object name cannot be empty")
	}

	// TODO: Fix minio.NewTags API usage
	/*
	objectTags, err := minio.NewTags(tags, false)
	if err != nil {
		return errors.WrapInternalError(err, "ERR_INTERNAL", "failed to create tags")
	}

	err = mc.client.PutObjectTagging(ctx, bucketName, objectName, objectTags, minio.PutObjectTaggingOptions{})
	if err != nil {
		return errors.WrapInternalError(err, "ERR_INTERNAL", "failed to set object tags")
	}
	*/

	return nil
}

// DeleteObjectTags 删除对象标签
func (mc *minioClient) DeleteObjectTags(ctx context.Context, bucketName, objectName string) error {
	if bucketName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
	}
	if objectName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "object name cannot be empty")
	}

	err := mc.client.RemoveObjectTagging(ctx, bucketName, objectName, minio.RemoveObjectTaggingOptions{})
	if err != nil {
		return errors.WrapInternalError(err, "ERR_INTERNAL", "failed to delete object tags")
	}

	return nil
}

// GetBucketSize 获取存储桶大小
func (mc *minioClient) GetBucketSize(ctx context.Context, bucketName string) (int64, error) {
	if bucketName == "" {
		return 0, errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
	}

	var totalSize int64
// 	opts := minio.ListObjectsOptions{
// 		Recursive: true,
// 	}

// 	for object := range mc.client.ListObjects(ctx, bucketName, opts) {
// 		if object.Err != nil {
// 			return 0, errors.WrapInternalError(object.Err, "ERR_INTERNAL", "failed to list objects")
// 		}
// 		totalSize += object.Size
// 	}
//
// 	return totalSize, nil
// }

// CountObjects 统计对象数量
// func (mc *minioClient) CountObjects(ctx context.Context, bucketName string, prefix string) (int64, error) {
// 	if bucketName == "" {
// 		return 0, errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
// 	}

// 	var count int64
// 	opts := minio.ListObjectsOptions{
// 		Prefix:    prefix,
// 		Recursive: true,
// 	}

// 	for object := range mc.client.ListObjects(ctx, bucketName, opts) {
// 		if object.Err != nil {
// 			return 0, errors.WrapInternalError(object.Err, "ERR_INTERNAL", "failed to list objects")
// 		}
// 		count++
// 	}
//
// 	return count, nil
// 	return 0, errors.NewInternalError(errors.CodeNotImplemented, "not implemented")
// }

// ValidateObjectName 验证对象名称
// func ValidateObjectName(name string) error {
// 	if name == "" {
// 		return errors.NewValidationError(errors.CodeInvalidParameter, "object name cannot be empty")
// 	}
// 	if len(name) > 1024 {
// 		return errors.NewValidationError(errors.CodeInvalidParameter, "object name too long")
// 	}
// 	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
// 		return errors.NewValidationError(errors.CodeInvalidParameter, "object name cannot start or end with /")
// 	}
// 	return nil
// }

// ValidateBucketName 验证存储桶名称
// func ValidateBucketName(name string) error {
// 	if name == "" {
// 		return errors.NewValidationError(errors.CodeInvalidParameter, "bucket name cannot be empty")
// 	}
// 	if len(name) < 3 || len(name) > 63 {
// 		return errors.NewValidationError(errors.CodeInvalidParameter, "bucket name length must be between 3 and 63")
// 	}
// 	// 更多验证规则...
// 	return nil
// }

// SanitizeObjectName 清理对象名称
// func SanitizeObjectName(name string) string {
// 	// 移除前导和尾随斜杠
// 	name = strings.Trim(name, "/")
// 	// 替换多个连续斜杠
// 	name = filepath.Clean(name)
// 	return name
// }

// ParseS3URI 解析 S3 URI
// func ParseS3URI(uri string) (bucket, key string, err error) {
// 	if !strings.HasPrefix(uri, "s3://") {
// 		return "", "", errors.NewValidationError(errors.CodeInvalidParameter, "invalid s3 uri")
// 	}

// 	uri = strings.TrimPrefix(uri, "s3://")
// 	parts := strings.SplitN(uri, "/", 2)

// 	if len(parts) < 1 {
// 		return "", "", errors.NewValidationError(errors.CodeInvalidParameter, "invalid s3 uri format")
// 	}

// 	bucket = parts[0]
// 	if len(parts) > 1 {
// 		key = parts[1]
// 	}

// 	return bucket, key, nil
// }

// BuildS3URI 构建 S3 URI
// func BuildS3URI(bucket, key string) string {
// 	return fmt.Sprintf("s3://%s/%s", bucket, key)
// }

//Personal.AI order the ending

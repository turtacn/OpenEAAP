package storage

import (
	"context"
	"fmt"
	"io"
	"time"
)

// ObjectStore 对象存储接口
// 抽象对象存储操作，支持多种对象存储实现
type ObjectStore interface {
	// Bucket 存储桶操作
	CreateBucket(ctx context.Context, config *BucketConfig) error
	DeleteBucket(ctx context.Context, name string) error
	BucketExists(ctx context.Context, name string) (bool, error)
	ListBuckets(ctx context.Context) ([]*BucketInfo, error)
	GetBucketInfo(ctx context.Context, name string) (*BucketInfo, error)
	SetBucketPolicy(ctx context.Context, name string, policy *BucketPolicy) error
	GetBucketPolicy(ctx context.Context, name string) (*BucketPolicy, error)

	// Object 对象操作
	PutObject(ctx context.Context, req *PutObjectRequest) (*ObjectInfo, error)
	GetObject(ctx context.Context, req *GetObjectRequest) (io.ReadCloser, error)
	HeadObject(ctx context.Context, bucket, key string) (*ObjectInfo, error)
	DeleteObject(ctx context.Context, bucket, key string) error
	DeleteObjects(ctx context.Context, bucket string, keys []string) (*BatchDeleteResult, error)
	CopyObject(ctx context.Context, req *CopyObjectRequest) (*ObjectInfo, error)
	MoveObject(ctx context.Context, req *MoveObjectRequest) error
	ListObjects(ctx context.Context, req *ListObjectsRequest) (*ListObjectsResult, error)

	// URL 操作
	GetPresignedURL(ctx context.Context, req *PresignedURLRequest) (*PresignedURL, error)
	GetPublicURL(bucket, key string) string

	// Metadata 元数据操作
	GetObjectMetadata(ctx context.Context, bucket, key string) (map[string]string, error)
	SetObjectMetadata(ctx context.Context, bucket, key string, metadata map[string]string) error

	// Tags 标签操作
	GetObjectTags(ctx context.Context, bucket, key string) (map[string]string, error)
	SetObjectTags(ctx context.Context, bucket, key string, tags map[string]string) error
	DeleteObjectTags(ctx context.Context, bucket, key string) error

	// Multipart 分片上传
	InitiateMultipartUpload(ctx context.Context, req *InitiateMultipartRequest) (*MultipartUpload, error)
	UploadPart(ctx context.Context, req *UploadPartRequest) (*PartInfo, error)
	CompleteMultipartUpload(ctx context.Context, req *CompleteMultipartRequest) (*ObjectInfo, error)
	AbortMultipartUpload(ctx context.Context, req *AbortMultipartRequest) error
	ListParts(ctx context.Context, req *ListPartsRequest) (*ListPartsResult, error)
	ListMultipartUploads(ctx context.Context, bucket, prefix string) ([]*MultipartUpload, error)

	// Versioning 版本控制
	SetBucketVersioning(ctx context.Context, bucket string, enabled bool) error
	GetBucketVersioning(ctx context.Context, bucket string) (bool, error)
	ListObjectVersions(ctx context.Context, req *ListObjectVersionsRequest) (*ListObjectVersionsResult, error)
	DeleteObjectVersion(ctx context.Context, bucket, key, versionID string) error

	// Lifecycle 生命周期
	SetBucketLifecycle(ctx context.Context, bucket string, rules []*LifecycleRule) error
	GetBucketLifecycle(ctx context.Context, bucket string) ([]*LifecycleRule, error)
	DeleteBucketLifecycle(ctx context.Context, bucket string) error

	// CORS 跨域
	SetBucketCORS(ctx context.Context, bucket string, rules []*CORSRule) error
	GetBucketCORS(ctx context.Context, bucket string) ([]*CORSRule, error)
	DeleteBucketCORS(ctx context.Context, bucket string) error

	// Stats 统计
	GetBucketSize(ctx context.Context, bucket string) (*BucketStats, error)
	CountObjects(ctx context.Context, bucket, prefix string) (int64, error)

	// Health 健康检查
	Ping(ctx context.Context) error
	Close() error
}

// BucketConfig 存储桶配置
type BucketConfig struct {
	Name              string            // 存储桶名称
	Region            string            // 区域
	ACL               ACLType           // 访问控制列表
	Versioning        bool              // 版本控制
	ObjectLocking     bool              // 对象锁定
	Encryption        *EncryptionConfig // 加密配置
	Tags              map[string]string // 标签
	StorageClass      StorageClass      // 存储类别
	ReplicationConfig *ReplicationConfig // 复制配置
}

// BucketInfo 存储桶信息
type BucketInfo struct {
	Name         string            // 存储桶名称
	Region       string            // 区域
	CreatedAt    time.Time         // 创建时间
	Owner        *Owner            // 所有者
	ACL          ACLType           // 访问控制
	Versioning   bool              // 版本控制
	Encryption   *EncryptionConfig // 加密配置
	Tags         map[string]string // 标签
	StorageClass StorageClass      // 存储类别
	ObjectCount  int64             // 对象数量
	TotalSize    int64             // 总大小
}

// BucketPolicy 存储桶策略
type BucketPolicy struct {
	Version    string            // 版本
	Statements []*PolicyStatement // 策略声明
}

// PolicyStatement 策略声明
type PolicyStatement struct {
	Effect     PolicyEffect       // 效果
	Principal  *Principal         // 主体
	Actions    []string           // 动作
	Resources  []string           // 资源
	Conditions []*PolicyCondition // 条件
}

// PolicyEffect 策略效果
type PolicyEffect string

const (
	PolicyEffectAllow PolicyEffect = "Allow" // 允许
	PolicyEffectDeny  PolicyEffect = "Deny"  // 拒绝
)

// Principal 主体
type Principal struct {
	Type  PrincipalType // 类型
	Value []string      // 值
}

// PrincipalType 主体类型
type PrincipalType string

const (
	PrincipalTypeAWS           PrincipalType = "AWS"           // AWS
	PrincipalTypeCanonicalUser PrincipalType = "CanonicalUser" // 规范用户
	PrincipalTypeFederated     PrincipalType = "Federated"     // 联合
	PrincipalTypeService       PrincipalType = "Service"       // 服务
	PrincipalTypeAll           PrincipalType = "*"             // 所有
)

// PolicyCondition 策略条件
type PolicyCondition struct {
	Operator string                 // 操作符
	Key      string                 // 键
	Values   []string               // 值
	Context  map[string]interface{} // 上下文
}

// PutObjectRequest 上传对象请求
type PutObjectRequest struct {
	Bucket           string            // 存储桶
	Key              string            // 对象键
	Body             io.Reader         // 数据体
	Size             int64             // 大小（-1表示未知）
	ContentType      string            // 内容类型
	ContentEncoding  string            // 内容编码
	ContentLanguage  string            // 内容语言
	ContentMD5       string            // 内容MD5
	CacheControl     string            // 缓存控制
	ContentDisposition string          // 内容配置
	Expires          *time.Time        // 过期时间
	Metadata         map[string]string // 元数据
	Tags             map[string]string // 标签
	ACL              ACLType           // 访问控制
	StorageClass     StorageClass      // 存储类别
	Encryption       *EncryptionConfig // 加密配置
	ProgressCallback ProgressCallback  // 进度回调
}

// GetObjectRequest 获取对象请求
type GetObjectRequest struct {
	Bucket               string            // 存储桶
	Key                  string            // 对象键
	VersionID            string            // 版本ID
	Range                *Range            // 范围
	IfMatch              string            // 如果匹配
	IfNoneMatch          string            // 如果不匹配
	IfModifiedSince      *time.Time        // 如果修改自
	IfUnmodifiedSince    *time.Time        // 如果未修改自
	ResponseContentType  string            // 响应内容类型
	ResponseCacheControl string            // 响应缓存控制
	Headers              map[string]string // 请求头
	ProgressCallback     ProgressCallback  // 进度回调
}

// Range 范围
type Range struct {
	Start int64 // 起始位置
	End   int64 // 结束位置
}

// String 返回范围字符串表示
func (r *Range) String() string {
	if r.End > 0 {
		return fmt.Sprintf("bytes=%d-%d", r.Start, r.End)
	}
	return fmt.Sprintf("bytes=%d-", r.Start)
}

// CopyObjectRequest 复制对象请求
type CopyObjectRequest struct {
	SourceBucket      string            // 源存储桶
	SourceKey         string            // 源对象键
	SourceVersionID   string            // 源版本ID
	DestBucket        string            // 目标存储桶
	DestKey           string            // 目标对象键
	Metadata          map[string]string // 元数据
	MetadataDirective MetadataDirective // 元数据指令
	Tags              map[string]string // 标签
	TaggingDirective  TaggingDirective  // 标签指令
	ACL               ACLType           // 访问控制
	StorageClass      StorageClass      // 存储类别
}

// MoveObjectRequest 移动对象请求
type MoveObjectRequest struct {
	SourceBucket string // 源存储桶
	SourceKey    string // 源对象键
	DestBucket   string // 目标存储桶
	DestKey      string // 目标对象键
}

// MetadataDirective 元数据指令
type MetadataDirective string

const (
	MetadataDirectiveCopy    MetadataDirective = "COPY"    // 复制
	MetadataDirectiveReplace MetadataDirective = "REPLACE" // 替换
)

// TaggingDirective 标签指令
type TaggingDirective string

const (
	TaggingDirectiveCopy    TaggingDirective = "COPY"    // 复制
	TaggingDirectiveReplace TaggingDirective = "REPLACE" // 替换
)

// ListObjectsRequest 列出对象请求
type ListObjectsRequest struct {
	Bucket            string // 存储桶
	Prefix            string // 前缀
	Delimiter         string // 分隔符
	MaxKeys           int    // 最大键数
	StartAfter        string // 起始位置
	ContinuationToken string // 继续标记
	EncodingType      string // 编码类型
	FetchOwner        bool   // 获取所有者
}

// ListObjectsResult 列出对象结果
type ListObjectsResult struct {
	Objects               []*ObjectInfo // 对象列表
	CommonPrefixes        []string      // 公共前缀
	IsTruncated           bool          // 是否截断
	NextContinuationToken string        // 下一个继续标记
	KeyCount              int           // 键数量
}

// ObjectInfo 对象信息
type ObjectInfo struct {
	Bucket           string            // 存储桶
	Key              string            // 对象键
	VersionID        string            // 版本ID
	IsLatest         bool              // 是否最新版本
	Size             int64             // 大小
	ETag             string            // ETag
	ContentType      string            // 内容类型
	ContentEncoding  string            // 内容编码
	ContentLanguage  string            // 内容语言
	LastModified     time.Time         // 最后修改时间
	Expires          *time.Time        // 过期时间
	CacheControl     string            // 缓存控制
	StorageClass     StorageClass      // 存储类别
	Metadata         map[string]string // 元数据
	Tags             map[string]string // 标签
	Owner            *Owner            // 所有者
	ACL              ACLType           // 访问控制
	ServerSideEncryption string        // 服务端加密
	DeleteMarker     bool              // 删除标记
}

// Owner 所有者
type Owner struct {
	ID          string // ID
	DisplayName string // 显示名称
}

// BatchDeleteResult 批量删除结果
type BatchDeleteResult struct {
	Deleted []string       // 已删除
	Errors  []*DeleteError // 错误列表
}

// DeleteError 删除错误
type DeleteError struct {
	Key     string // 键
	Code    string // 错误代码
	Message string // 错误消息
}

// PresignedURLRequest 预签名URL请求
type PresignedURLRequest struct {
	Bucket     string            // 存储桶
	Key        string            // 对象键
	Method     HTTPMethod        // HTTP方法
	Expiration time.Duration     // 过期时间
	Headers    map[string]string // 请求头
	Params     map[string]string // 查询参数
}

// HTTPMethod HTTP方法
type HTTPMethod string

const (
	HTTPMethodGET    HTTPMethod = "GET"    // GET
	HTTPMethodPUT    HTTPMethod = "PUT"    // PUT
	HTTPMethodDELETE HTTPMethod = "DELETE" // DELETE
	HTTPMethodPOST   HTTPMethod = "POST"   // POST
	HTTPMethodHEAD   HTTPMethod = "HEAD"   // HEAD
)

// PresignedURL 预签名URL
type PresignedURL struct {
	URL        string            // URL
	Method     HTTPMethod        // 方法
	Headers    map[string]string // 头部
	Expiration time.Time         // 过期时间
}

// InitiateMultipartRequest 初始化分片上传请求
type InitiateMultipartRequest struct {
	Bucket      string            // 存储桶
	Key         string            // 对象键
	ContentType string            // 内容类型
	Metadata    map[string]string // 元数据
	Tags        map[string]string // 标签
	ACL         ACLType           // 访问控制
	StorageClass StorageClass     // 存储类别
	Encryption  *EncryptionConfig // 加密配置
}

// MultipartUpload 分片上传
type MultipartUpload struct {
	UploadID     string            // 上传ID
	Bucket       string            // 存储桶
	Key          string            // 对象键
	Initiated    time.Time         // 初始化时间
	StorageClass StorageClass      // 存储类别
	Owner        *Owner            // 所有者
	Metadata     map[string]string // 元数据
}

// UploadPartRequest 上传分片请求
type UploadPartRequest struct {
	Bucket           string           // 存储桶
	Key              string           // 对象键
	UploadID         string           // 上传ID
	PartNumber       int              // 分片编号
	Body             io.Reader        // 数据体
	Size             int64            // 大小
	ContentMD5       string           // 内容MD5
	ProgressCallback ProgressCallback // 进度回调
}

// PartInfo 分片信息
type PartInfo struct {
	PartNumber   int       // 分片编号
	ETag         string    // ETag
	Size         int64     // 大小
	LastModified time.Time // 最后修改时间
}

// CompleteMultipartRequest 完成分片上传请求
type CompleteMultipartRequest struct {
	Bucket   string      // 存储桶
	Key      string      // 对象键
	UploadID string      // 上传ID
	Parts    []*PartInfo // 分片列表
}

// AbortMultipartRequest 终止分片上传请求
type AbortMultipartRequest struct {
	Bucket   string // 存储桶
	Key      string // 对象键
	UploadID string // 上传ID
}

// ListPartsRequest 列出分片请求
type ListPartsRequest struct {
	Bucket           string // 存储桶
	Key              string // 对象键
	UploadID         string // 上传ID
	MaxParts         int    // 最大分片数
	PartNumberMarker int    // 分片编号标记
}

// ListPartsResult 列出分片结果
type ListPartsResult struct {
	Parts                []*PartInfo // 分片列表
	IsTruncated          bool        // 是否截断
	NextPartNumberMarker int         // 下一个分片编号标记
	MaxParts             int         // 最大分片数
	PartCount            int         // 分片数量
}

// ListObjectVersionsRequest 列出对象版本请求
type ListObjectVersionsRequest struct {
	Bucket          string // 存储桶
	Prefix          string // 前缀
	Delimiter       string // 分隔符
	MaxKeys         int    // 最大键数
	KeyMarker       string // 键标记
	VersionIDMarker string // 版本ID标记
}

// ListObjectVersionsResult 列出对象版本结果
type ListObjectVersionsResult struct {
	Versions           []*ObjectInfo // 版本列表
	DeleteMarkers      []*ObjectInfo // 删除标记列表
	CommonPrefixes     []string      // 公共前缀
	IsTruncated        bool          // 是否截断
	NextKeyMarker      string        // 下一个键标记
	NextVersionIDMarker string       // 下一个版本ID标记
}

// LifecycleRule 生命周期规则
type LifecycleRule struct {
	ID                             string                  // ID
	Status                         RuleStatus              // 状态
	Filter                         *LifecycleFilter        // 过滤器
	Transitions                    []*Transition           // 转换规则
	Expiration                     *Expiration             // 过期规则
	NoncurrentVersionTransitions   []*NoncurrentTransition // 非当前版本转换
	NoncurrentVersionExpiration    *NoncurrentExpiration   // 非当前版本过期
	AbortIncompleteMultipartUpload *AbortIncompleteMultipart // 终止未完成分片上传
}

// RuleStatus 规则状态
type RuleStatus string

const (
	RuleStatusEnabled  RuleStatus = "Enabled"  // 启用
	RuleStatusDisabled RuleStatus = "Disabled" // 禁用
)

// LifecycleFilter 生命周期过滤器
type LifecycleFilter struct {
	Prefix                string            // 前缀
	Tags                  map[string]string // 标签
	ObjectSizeGreaterThan int64             // 对象大小大于
	ObjectSizeLessThan    int64             // 对象大小小于
	And                   *LifecycleFilterAnd // AND条件
}

// LifecycleFilterAnd AND条件
type LifecycleFilterAnd struct {
	Prefix                string            // 前缀
	Tags                  map[string]string // 标签
	ObjectSizeGreaterThan int64             // 对象大小大于
	ObjectSizeLessThan    int64             // 对象大小小于
}

// Transition 转换规则
type Transition struct {
	Days         int          // 天数
	Date         *time.Time   // 日期
	StorageClass StorageClass // 存储类别
}

// Expiration 过期规则
type Expiration struct {
	Days                      int        // 天数
	Date                      *time.Time // 日期
	ExpiredObjectDeleteMarker bool       // 过期对象删除标记
}

// NoncurrentTransition 非当前版本转换
type NoncurrentTransition struct {
	NoncurrentDays int          // 非当前天数
	StorageClass   StorageClass // 存储类别
}

// NoncurrentExpiration 非当前版本过期
type NoncurrentExpiration struct {
	NoncurrentDays int // 非当前天数
}

// AbortIncompleteMultipart 终止未完成分片上传
type AbortIncompleteMultipart struct {
	DaysAfterInitiation int // 初始化后天数
}

// CORSRule CORS规则
type CORSRule struct {
	ID             string   // ID
	AllowedOrigins []string // 允许的源
	AllowedMethods []string // 允许的方法
	AllowedHeaders []string // 允许的头
	ExposeHeaders  []string // 暴露的头
	MaxAgeSeconds  int      // 最大年龄（秒）
}

// ACLType 访问控制类型
type ACLType string

const (
	ACLPrivate           ACLType = "private"            // 私有
	ACLPublicRead        ACLType = "public-read"        // 公共读
	ACLPublicReadWrite   ACLType = "public-read-write"  // 公共读写
	ACLAuthenticatedRead ACLType = "authenticated-read" // 认证读
	ACLBucketOwnerRead   ACLType = "bucket-owner-read"  // 存储桶所有者读
	ACLBucketOwnerFull   ACLType = "bucket-owner-full-control" // 存储桶所有者完全控制
)

// StorageClass 存储类别
type StorageClass string

const (
	StorageClassStandard          StorageClass = "STANDARD"            // 标准
	StorageClassReducedRedundancy StorageClass = "REDUCED_REDUNDANCY"  // 降低冗余
	StorageClassGlacier           StorageClass = "GLACIER"             // 冰川
	StorageClassStandardIA        StorageClass = "STANDARD_IA"         // 标准不频繁访问
	StorageClassOnezoneIA         StorageClass = "ONEZONE_IA"          // 单区不频繁访问
	StorageClassIntelligentTiering StorageClass = "INTELLIGENT_TIERING" // 智能分层
	StorageClassDeepArchive       StorageClass = "DEEP_ARCHIVE"        // 深度归档
	StorageClassGlacierIR         StorageClass = "GLACIER_IR"          // 冰川即时检索
)

// EncryptionConfig 加密配置
type EncryptionConfig struct {
	Type      EncryptionType // 类型
	Algorithm string         // 算法
	KeyID     string         // 密钥ID
	Key       string         // 密钥
}

// EncryptionType 加密类型
type EncryptionType string

const (
	EncryptionTypeNone EncryptionType = "none" // 无
	EncryptionTypeAES  EncryptionType = "AES256" // AES256
	EncryptionTypeKMS  EncryptionType = "aws:kms" // KMS
)

// ReplicationConfig 复制配置
type ReplicationConfig struct {
	Role  string              // 角色
	Rules []*ReplicationRule  // 规则列表
}

// ReplicationRule 复制规则
type ReplicationRule struct {
	ID          string               // ID
	Status      RuleStatus           // 状态
	Priority    int                  // 优先级
	Filter      *ReplicationFilter   // 过滤器
	Destination *ReplicationDestination // 目标
}

// ReplicationFilter 复制过滤器
type ReplicationFilter struct {
	Prefix string            // 前缀
	Tags   map[string]string // 标签
}

// ReplicationDestination 复制目标
type ReplicationDestination struct {
	Bucket       string       // 存储桶
	Account      string       // 账户
	StorageClass StorageClass // 存储类别
}

// BucketStats 存储桶统计
type BucketStats struct {
	ObjectCount int64     // 对象数量
	TotalSize   int64     // 总大小
	LastUpdated time.Time // 最后更新时间
}

// ProgressCallback 进度回调
type ProgressCallback func(transferred, total int64)

// ObjectStoreFactory 对象存储工厂接口
type ObjectStoreFactory interface {
	// Create 创建对象存储实例
	Create(config map[string]interface{}) (ObjectStore, error)
	// Type 返回存储类型
	Type() string
}

// ObjectStoreType 对象存储类型
type ObjectStoreType string

const (
	ObjectStoreTypeMinIO    ObjectStoreType = "minio"    // MinIO
	ObjectStoreTypeS3       ObjectStoreType = "s3"       // AWS S3
	ObjectStoreTypeOSS      ObjectStoreType = "oss"      // 阿里云OSS
	ObjectStoreTypeCOS      ObjectStoreType = "cos"      // 腾讯云COS
	ObjectStoreTypeOBS      ObjectStoreType = "obs"      // 华为云OBS
	ObjectStoreTypeGCS      ObjectStoreType = "gcs"      // Google Cloud Storage
	ObjectStoreTypeAzure    ObjectStoreType = "azure"    // Azure Blob Storage
	ObjectStoreTypeBackblaze ObjectStoreType = "backblaze" // Backblaze B2
)

//Personal.AI order the ending

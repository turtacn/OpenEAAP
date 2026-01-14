package vector

import (
	"context"
	"time"
)

// VectorStore 向量存储接口
// 抽象向量数据库操作，支持多种向量数据库实现
type VectorStore interface {
	// Collection 集合管理
	CreateCollection(ctx context.Context, config *CollectionConfig) error
	DeleteCollection(ctx context.Context, name string) error
	CollectionExists(ctx context.Context, name string) (bool, error)
	GetCollectionInfo(ctx context.Context, name string) (*CollectionInfo, error)
	ListCollections(ctx context.Context) ([]string, error)

	// Vector 向量操作
	Insert(ctx context.Context, req *InsertRequest) (*InsertResponse, error)
	BatchInsert(ctx context.Context, req *BatchInsertRequest) (*BatchInsertResponse, error)
	Update(ctx context.Context, req *UpdateRequest) error
	Delete(ctx context.Context, req *DeleteRequest) error

	// Search 检索操作
	Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error)
	HybridSearch(ctx context.Context, req *HybridSearchRequest) (*SearchResponse, error)
	BatchSearch(ctx context.Context, req *BatchSearchRequest) (*BatchSearchResponse, error)

	// Index 索引管理
	CreateIndex(ctx context.Context, req *IndexRequest) error
	DeleteIndex(ctx context.Context, collection, field string) error
	GetIndexInfo(ctx context.Context, collection, field string) (*IndexInfo, error)

	// Lifecycle 生命周期
	LoadCollection(ctx context.Context, name string) error
	ReleaseCollection(ctx context.Context, name string) error
	Flush(ctx context.Context, collections ...string) error

	// Stats 统计信息
	GetStats(ctx context.Context, collection string) (*CollectionStats, error)

	// Health 健康检查
	Ping(ctx context.Context) error
	Close() error
}

// CollectionConfig 集合配置
type CollectionConfig struct {
	Name               string                 // 集合名称
	Description        string                 // 描述
	Schema             *Schema                // 数据模式
	ShardNum           int32                  // 分片数量
	ReplicationFactor  int32                  // 副本数量
	ConsistencyLevel   ConsistencyLevel       // 一致性级别
	Properties         map[string]string      // 自定义属性
	TTL                *time.Duration         // 数据过期时间
}

// Schema 数据模式
type Schema struct {
	Fields              []*Field // 字段定义
	EnableDynamicField  bool     // 是否启用动态字段
	PrimaryField        string   // 主键字段名
	VectorFields        []string // 向量字段名列表
}

// Field 字段定义
type Field struct {
	Name         string    // 字段名
	DataType     DataType  // 数据类型
	IsPrimary    bool      // 是否主键
	AutoID       bool      // 是否自动生成ID
	Dimension    int       // 向量维度（仅向量类型）
	MaxLength    int       // 最大长度（仅字符串类型）
	Description  string    // 描述
	DefaultValue interface{} // 默认值
	Indexed      bool      // 是否索引
	Nullable     bool      // 是否可为空
}

// DataType 数据类型
type DataType int

const (
	DataTypeUnknown DataType = iota
	DataTypeInt8
	DataTypeInt16
	DataTypeInt32
	DataTypeInt64
	DataTypeFloat
	DataTypeDouble
	DataTypeString
	DataTypeBool
	DataTypeJSON
	DataTypeFloatVector
	DataTypeBinaryVector
	DataTypeSparseVector
	DataTypeArray
)

// String 返回数据类型字符串表示
func (dt DataType) String() string {
	switch dt {
	case DataTypeInt8:
		return "int8"
	case DataTypeInt16:
		return "int16"
	case DataTypeInt32:
		return "int32"
	case DataTypeInt64:
		return "int64"
	case DataTypeFloat:
		return "float"
	case DataTypeDouble:
		return "double"
	case DataTypeString:
		return "string"
	case DataTypeBool:
		return "bool"
	case DataTypeJSON:
		return "json"
	case DataTypeFloatVector:
		return "float_vector"
	case DataTypeBinaryVector:
		return "binary_vector"
	case DataTypeSparseVector:
		return "sparse_vector"
	case DataTypeArray:
		return "array"
	default:
		return "unknown"
	}
}

// InsertRequest 插入请求
type InsertRequest struct {
	Collection string                   // 集合名称
	Partition  string                   // 分区名称（可选）
	Data       []map[string]interface{} // 数据记录
}

// InsertResponse 插入响应
type InsertResponse struct {
	IDs          []interface{} // 插入的ID列表
	InsertCount  int64         // 插入数量
	InsertCost   time.Duration // 插入耗时
}

// BatchInsertRequest 批量插入请求
type BatchInsertRequest struct {
	Collection string                   // 集合名称
	Partition  string                   // 分区名称（可选）
	Data       []map[string]interface{} // 数据记录
	BatchSize  int                      // 批次大小
}

// BatchInsertResponse 批量插入响应
type BatchInsertResponse struct {
	IDs         []interface{} // 插入的ID列表
	InsertCount int64         // 插入数量
	FailCount   int64         // 失败数量
	TotalCost   time.Duration // 总耗时
}

// UpdateRequest 更新请求
type UpdateRequest struct {
	Collection string                   // 集合名称
	Filter     string                   // 过滤条件
	Data       map[string]interface{}   // 更新数据
}

// DeleteRequest 删除请求
type DeleteRequest struct {
	Collection string        // 集合名称
	IDs        []interface{} // ID列表
	Filter     string        // 过滤条件（与IDs二选一）
}

// SearchRequest 检索请求
type SearchRequest struct {
	Collection       string                 // 集合名称
	Partition        string                 // 分区名称（可选）
	Vectors          [][]float32            // 查询向量
	VectorField      string                 // 向量字段名
	MetricType       MetricType             // 度量类型
	TopK             int                    // 返回前K个结果
	Filter           string                 // 过滤表达式
	OutputFields     []string               // 输出字段
	SearchParams     map[string]interface{} // 搜索参数
	Offset           int                    // 偏移量
	Limit            int                    // 限制数量
	ConsistencyLevel ConsistencyLevel       // 一致性级别
	Timeout          time.Duration          // 超时时间
}

// HybridSearchRequest 混合检索请求
type HybridSearchRequest struct {
	Collection       string                 // 集合名称
	VectorSearches   []*VectorSearchParams  // 多个向量搜索
	ScalarFilter     string                 // 标量过滤条件
	Reranker         *RerankerParams        // 重排序参数
	TopK             int                    // 返回前K个结果
	OutputFields     []string               // 输出字段
	ConsistencyLevel ConsistencyLevel       // 一致性级别
}

// VectorSearchParams 向量搜索参数
type VectorSearchParams struct {
	Vectors      [][]float32            // 查询向量
	VectorField  string                 // 向量字段名
	MetricType   MetricType             // 度量类型
	TopK         int                    // 该搜索返回前K个结果
	Filter       string                 // 过滤表达式
	SearchParams map[string]interface{} // 搜索参数
	Weight       float32                // 权重（用于结果融合）
}

// RerankerParams 重排序参数
type RerankerParams struct {
	Type   RerankerType           // 重排序类型
	Params map[string]interface{} // 重排序参数
}

// RerankerType 重排序类型
type RerankerType string

const (
	RerankerTypeRRF        RerankerType = "rrf"        // Reciprocal Rank Fusion
	RerankerTypeWeighted   RerankerType = "weighted"   // 加权融合
	RerankerTypeCrossEncoder RerankerType = "cross_encoder" // 交叉编码器
)

// BatchSearchRequest 批量检索请求
type BatchSearchRequest struct {
	Searches []*SearchRequest // 多个检索请求
}

// SearchResponse 检索响应
type SearchResponse struct {
	Results    []*SearchResult // 检索结果
	SearchCost time.Duration   // 检索耗时
	TotalCount int64           // 总结果数
}

// BatchSearchResponse 批量检索响应
type BatchSearchResponse struct {
	Responses  []*SearchResponse // 多个检索响应
	TotalCost  time.Duration     // 总耗时
}

// SearchResult 检索结果
type SearchResult struct {
	ID       interface{}            // 向量ID
	Score    float32                // 相似度分数
	Distance float32                // 距离
	Fields   map[string]interface{} // 字段数据
	Metadata map[string]interface{} // 元数据
}

// IndexRequest 索引请求
type IndexRequest struct {
	Collection string                 // 集合名称
	Field      string                 // 字段名
	IndexType  IndexType              // 索引类型
	MetricType MetricType             // 度量类型
	Params     map[string]interface{} // 索引参数
}

// IndexType 索引类型
type IndexType string

const (
	IndexTypeFlat       IndexType = "FLAT"        // 暴力检索
	IndexTypeIVFFlat    IndexType = "IVF_FLAT"    // IVF索引
	IndexTypeIVFSQ8     IndexType = "IVF_SQ8"     // IVF量化索引
	IndexTypeIVFPQ      IndexType = "IVF_PQ"      // IVF乘积量化
	IndexTypeHNSW       IndexType = "HNSW"        // HNSW图索引
	IndexTypeANNOY      IndexType = "ANNOY"       // ANNOY索引
	IndexTypeDiskANN    IndexType = "DISKANN"     // DiskANN索引
	IndexTypeAutoIndex  IndexType = "AUTOINDEX"   // 自动选择索引
)

// MetricType 度量类型
type MetricType string

const (
	MetricTypeL2               MetricType = "L2"               // 欧氏距离
	MetricTypeIP               MetricType = "IP"               // 内积
	MetricTypeCosine           MetricType = "COSINE"           // 余弦相似度
	MetricTypeHamming          MetricType = "HAMMING"          // 汉明距离
	MetricTypeJaccard          MetricType = "JACCARD"          // Jaccard距离
	MetricTypeTanimoto         MetricType = "TANIMOTO"         // Tanimoto系数
	MetricTypeSuperstructure   MetricType = "SUPERSTRUCTURE"   // 超结构
	MetricTypeSubstructure     MetricType = "SUBSTRUCTURE"     // 子结构
)

// ConsistencyLevel 一致性级别
type ConsistencyLevel int

const (
	ConsistencyLevelStrong     ConsistencyLevel = iota // 强一致性
	ConsistencyLevelBounded                            // 有界一致性
	ConsistencyLevelSession                            // 会话一致性
	ConsistencyLevelEventual                           // 最终一致性
)

// String 返回一致性级别字符串表示
func (cl ConsistencyLevel) String() string {
	switch cl {
	case ConsistencyLevelStrong:
		return "strong"
	case ConsistencyLevelBounded:
		return "bounded"
	case ConsistencyLevelSession:
		return "session"
	case ConsistencyLevelEventual:
		return "eventual"
	default:
		return "unknown"
	}
}

// CollectionInfo 集合信息
type CollectionInfo struct {
	Name              string           // 集合名称
	Description       string           // 描述
	Schema            *Schema          // 数据模式
	ShardNum          int32            // 分片数量
	ReplicationFactor int32            // 副本数量
	ConsistencyLevel  ConsistencyLevel // 一致性级别
	CreatedAt         time.Time        // 创建时间
	UpdatedAt         time.Time        // 更新时间
	IsLoaded          bool             // 是否已加载
	Properties        map[string]string // 自定义属性
}

// IndexInfo 索引信息
type IndexInfo struct {
	Field      string                 // 字段名
	IndexType  IndexType              // 索引类型
	MetricType MetricType             // 度量类型
	Params     map[string]interface{} // 索引参数
	State      IndexState             // 索引状态
	CreatedAt  time.Time              // 创建时间
}

// IndexState 索引状态
type IndexState string

const (
	IndexStateBuilding IndexState = "building" // 构建中
	IndexStateFinished IndexState = "finished" // 已完成
	IndexStateFailed   IndexState = "failed"   // 失败
)

// CollectionStats 集合统计信息
type CollectionStats struct {
	RowCount      int64     // 行数
	DataSize      int64     // 数据大小（字节）
	IndexedRows   int64     // 已索引行数
	MemorySize    int64     // 内存占用（字节）
	DiskSize      int64     // 磁盘占用（字节）
	LastUpdated   time.Time // 最后更新时间
}

// VectorStoreFactory 向量存储工厂接口
type VectorStoreFactory interface {
	// Create 创建向量存储实例
	Create(config map[string]interface{}) (VectorStore, error)
	// Type 返回存储类型
	Type() string
}

// VectorStoreType 向量存储类型
type VectorStoreType string

const (
	VectorStoreTypeMilvus   VectorStoreType = "milvus"   // Milvus
	VectorStoreTypeQdrant   VectorStoreType = "qdrant"   // Qdrant
	VectorStoreTypeWeaviate VectorStoreType = "weaviate" // Weaviate
	VectorStoreTypePinecone VectorStoreType = "pinecone" // Pinecone
	VectorStoreTypeChroma   VectorStoreType = "chroma"   // ChromaDB
)

// VectorMetadata 向量元数据
type VectorMetadata struct {
	ID          interface{}            // 向量ID
	Vector      []float32              // 向量数据
	Fields      map[string]interface{} // 字段数据
	Metadata    map[string]interface{} // 额外元数据
	CreatedAt   time.Time              // 创建时间
	UpdatedAt   time.Time              // 更新时间
}

// SearchOptions 检索选项
type SearchOptions struct {
	EnableCache      bool          // 启用缓存
	CacheTTL         time.Duration // 缓存过期时间
	EnablePrefetch   bool          // 启用预取
	EnableRerank     bool          // 启用重排序
	RerankModel      string        // 重排序模型
	MinScore         float32       // 最小分数阈值
	MaxDistance      float32       // 最大距离阈值
	DiversityFactor  float32       // 多样性因子
	GroupBy          string        // 分组字段
	GroupLimit       int           // 每组限制数量
}

// BulkOperation 批量操作接口
type BulkOperation interface {
	// Add 添加操作
	Add(op Operation)
	// Execute 执行批量操作
	Execute(ctx context.Context) (*BulkOperationResult, error)
	// Size 返回操作数量
	Size() int
	// Clear 清空操作
	Clear()
}

// Operation 操作接口
type Operation interface {
	// Type 返回操作类型
	Type() OperationType
	// Validate 验证操作
	Validate() error
}

// OperationType 操作类型
type OperationType string

const (
	OperationTypeInsert OperationType = "insert" // 插入
	OperationTypeUpdate OperationType = "update" // 更新
	OperationTypeDelete OperationType = "delete" // 删除
)

// BulkOperationResult 批量操作结果
type BulkOperationResult struct {
	SuccessCount int64         // 成功数量
	FailCount    int64         // 失败数量
	Errors       []error       // 错误列表
	Duration     time.Duration // 执行时长
}

// Snapshot 快照接口
type Snapshot interface {
	// ID 快照ID
	ID() string
	// Collection 集合名称
	Collection() string
	// CreatedAt 创建时间
	CreatedAt() time.Time
	// Restore 恢复快照
	Restore(ctx context.Context) error
	// Delete 删除快照
	Delete(ctx context.Context) error
}

// SnapshotManager 快照管理器接口
type SnapshotManager interface {
	// Create 创建快照
	Create(ctx context.Context, collection string) (Snapshot, error)
	// List 列出快照
	List(ctx context.Context, collection string) ([]Snapshot, error)
	// Get 获取快照
	Get(ctx context.Context, id string) (Snapshot, error)
	// Delete 删除快照
	Delete(ctx context.Context, id string) error
}

// Monitor 监控接口
type Monitor interface {
	// RecordInsert 记录插入操作
	RecordInsert(collection string, count int64, duration time.Duration)
	// RecordSearch 记录检索操作
	RecordSearch(collection string, topK int, duration time.Duration)
	// RecordError 记录错误
	RecordError(collection string, operation string, err error)
	// GetMetrics 获取指标
	GetMetrics(collection string) *Metrics
}

// Metrics 指标数据
type Metrics struct {
	InsertCount   int64         // 插入次数
	InsertLatency time.Duration // 插入延迟
	SearchCount   int64         // 检索次数
	SearchLatency time.Duration // 检索延迟
	ErrorCount    int64         // 错误次数
	QPS           float64       // 每秒查询数
	UpdatedAt     time.Time     // 更新时间
}

//Personal.AI order the ending

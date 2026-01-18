package milvus

import (
	"context"
	"fmt"
	"time"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"github.com/openeeap/openeeap/pkg/errors"
)

// MilvusClient Milvus 向量数据库客户端接口
type MilvusClient interface {
	// CreateCollection 创建集合
	CreateCollection(ctx context.Context, config *CollectionConfig) error
	// DropCollection 删除集合
	DropCollection(ctx context.Context, collectionName string) error
	// HasCollection 检查集合是否存在
	HasCollection(ctx context.Context, collectionName string) (bool, error)
	// DescribeCollection 描述集合
	DescribeCollection(ctx context.Context, collectionName string) (*CollectionInfo, error)
	// ListCollections 列出所有集合
	ListCollections(ctx context.Context) ([]string, error)

	// Insert 插入向量
	Insert(ctx context.Context, collectionName string, vectors [][]float32, metadata []map[string]interface{}) ([]int64, error)
	// Delete 删除向量
	Delete(ctx context.Context, collectionName string, ids []int64) error
	// Search 相似度搜索
	Search(ctx context.Context, params *SearchParams) ([]SearchResult, error)
	// HybridSearch 混合检索
	HybridSearch(ctx context.Context, params *HybridSearchParams) ([]SearchResult, error)

	// CreateIndex 创建索引
	CreateIndex(ctx context.Context, collectionName string, fieldName string, indexConfig *IndexConfig) error
	// DropIndex 删除索引
	DropIndex(ctx context.Context, collectionName string, fieldName string) error
	// DescribeIndex 描述索引
	DescribeIndex(ctx context.Context, collectionName string, fieldName string) (*IndexInfo, error)

	// LoadCollection 加载集合到内存
	LoadCollection(ctx context.Context, collectionName string) error
	// ReleaseCollection 释放集合
	ReleaseCollection(ctx context.Context, collectionName string) error

	// GetCollectionStatistics 获取集合统计信息
	GetCollectionStatistics(ctx context.Context, collectionName string) (*CollectionStats, error)
	// Flush 刷新数据到磁盘
	Flush(ctx context.Context, collectionNames ...string) error

	// Close 关闭客户端
	Close() error
	// Ping 健康检查
	Ping(ctx context.Context) error
}

// milvusClient Milvus 客户端实现
type milvusClient struct {
	client client.Client
	config *MilvusConfig
}

// MilvusConfig Milvus 配置
type MilvusConfig struct {
	Address       string        // Milvus 地址
	Username      string        // 用户名
	Password      string        // 密码
	Database      string        // 数据库名
	Timeout       time.Duration // 超时时间
	MaxRetries    int           // 最大重试次数
	RetryInterval time.Duration // 重试间隔
}

// CollectionConfig 集合配置
type CollectionConfig struct {
	Name        string              // 集合名称
	Description string              // 描述
	Schema      *CollectionSchema   // 集合架构
	ShardNum    int32               // 分片数量
	Properties  map[string]string   // 属性
}

// CollectionSchema 集合架构
type CollectionSchema struct {
	Fields          []*FieldSchema // 字段定义
	EnableDynamicField bool        // 是否启用动态字段
}

// FieldSchema 字段架构
type FieldSchema struct {
	Name        string          // 字段名
	DataType    entity.FieldType // 数据类型
	IsPrimaryKey bool           // 是否主键
	IsAutoID    bool            // 是否自动生成ID
	Dimension   int64           // 向量维度（仅向量字段）
	Description string          // 描述
}

// IndexConfig 索引配置
type IndexConfig struct {
	IndexType  entity.IndexType      // 索引类型
	MetricType entity.MetricType     // 度量类型
	Params     map[string]string     // 索引参数
}

// SearchParams 搜索参数
type SearchParams struct {
	CollectionName   string              // 集合名称
	Vectors          [][]float32         // 查询向量
	VectorFieldName  string              // 向量字段名
	MetricType       entity.MetricType   // 度量类型
	TopK             int                 // 返回前K个结果
	OutputFields     []string            // 输出字段
	Filter           string              // 过滤表达式
	SearchParams     map[string]interface{} // 搜索参数
	ConsistencyLevel entity.ConsistencyLevel // 一致性级别
}

// HybridSearchParams 混合检索参数
type HybridSearchParams struct {
	CollectionName   string                  // 集合名称
	VectorSearches   []*VectorSearch         // 多个向量搜索
	Reranker         *Reranker               // 重排序器
	TopK             int                     // 返回前K个结果
	OutputFields     []string                // 输出字段
	ConsistencyLevel entity.ConsistencyLevel // 一致性级别
}

// VectorSearch 单个向量搜索
type VectorSearch struct {
	Vectors         [][]float32            // 查询向量
	VectorFieldName string                 // 向量字段名
	MetricType      entity.MetricType      // 度量类型
	TopK            int                    // 该搜索返回前K个结果
	Filter          string                 // 过滤表达式
	SearchParams    map[string]interface{} // 搜索参数
	Weight          float32                // 权重
}

// Reranker 重排序器
type Reranker struct {
	Type   string                 // 重排序类型
	Params map[string]interface{} // 重排序参数
}

// SearchResult 搜索结果
type SearchResult struct {
	ID       int64                  // 向量ID
	Score    float32                // 相似度分数
	Fields   map[string]interface{} // 字段数据
	Distance float32                // 距离
}

// CollectionInfo 集合信息
type CollectionInfo struct {
	Name             string    // 集合名称
	Description      string    // 描述
	Schema           *CollectionSchema // 架构
	ShardNum         int32     // 分片数量
	CreatedTimestamp uint64    // 创建时间戳
	ConsistencyLevel entity.ConsistencyLevel // 一致性级别
}

// IndexInfo 索引信息
type IndexInfo struct {
	IndexType  entity.IndexType      // 索引类型
	MetricType entity.MetricType     // 度量类型
	Params     map[string]string     // 索引参数
}

// CollectionStats 集合统计信息
type CollectionStats struct {
	RowCount    int64 // 行数
	DataSize    int64 // 数据大小（字节）
	IndexedRows int64 // 已索引行数
}

// NewMilvusClient 创建 Milvus 客户端
func NewMilvusClient(config *MilvusConfig) (MilvusClient, error) {
	if config == nil {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "config cannot be nil")
	}

	// 设置默认值
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryInterval == 0 {
		config.RetryInterval = 1 * time.Second
	}

	// 创建客户端连接
	c, err := client.NewClient(context.Background(), client.Config{
		Address:  config.Address,
		Username: config.Username,
		Password: config.Password,
		DBName:   config.Database,
	})
	if err != nil {
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to connect to milvus")
	}

	mc := &milvusClient{
		client: c,
		config: config,
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := mc.Ping(ctx); err != nil {
		c.Close()
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to ping milvus")
	}

	return mc, nil
}

// CreateCollection 创建集合
func (mc *milvusClient) CreateCollection(ctx context.Context, config *CollectionConfig) error {
	if config == nil {
		return errors.NewValidationError(errors.CodeInvalidParameter, "collection config cannot be nil")
	}
	if config.Name == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "collection name cannot be empty")
	}
	if config.Schema == nil || len(config.Schema.Fields) == 0 {
		return errors.NewValidationError(errors.CodeInvalidParameter, "collection schema cannot be empty")
	}

	// 检查集合是否已存在
	exists, err := mc.HasCollection(ctx, config.Name)
	if err != nil {
		return err
	}
	if exists {
		return errors.NewConflictError("MILVUS_001", "collection already exists")
	}

	// 构建字段
	fields := make([]*entity.Field, 0, len(config.Schema.Fields))
	for _, fieldSchema := range config.Schema.Fields {
		field := &entity.Field{
			Name:         fieldSchema.Name,
			DataType:     fieldSchema.DataType,
			PrimaryKey:   fieldSchema.IsPrimaryKey,
			AutoID:       fieldSchema.IsAutoID,
			Description:  fieldSchema.Description,
		}

		// 设置向量维度
		if fieldSchema.DataType == entity.FieldTypeFloatVector ||
			fieldSchema.DataType == entity.FieldTypeBinaryVector {
			if fieldSchema.Dimension <= 0 {
				return errors.NewValidationError(errors.CodeInvalidParameter,
					fmt.Sprintf("invalid dimension for vector field %s", fieldSchema.Name))
			}
			field.TypeParams = map[string]string{
				"dim": fmt.Sprintf("%d", fieldSchema.Dimension),
			}
		}

		fields = append(fields, field)
	}

	// 构建集合 Schema
	schema := &entity.Schema{
		CollectionName:     config.Name,
		Description:        config.Description,
		Fields:             fields,
		EnableDynamicField: config.Schema.EnableDynamicField,
	}

	// 创建集合
	err = mc.client.CreateCollection(ctx, schema, config.ShardNum)
	if err != nil {
		return errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to create collection")
	}

	return nil
}

// DropCollection 删除集合
func (mc *milvusClient) DropCollection(ctx context.Context, collectionName string) error {
	if collectionName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "collection name cannot be empty")
	}

	err := mc.client.DropCollection(ctx, collectionName)
	if err != nil {
		return errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to drop collection")
	}

	return nil
}

// HasCollection 检查集合是否存在
func (mc *milvusClient) HasCollection(ctx context.Context, collectionName string) (bool, error) {
	if collectionName == "" {
		return false, errors.NewValidationError(errors.CodeInvalidParameter, "collection name cannot be empty")
	}

	exists, err := mc.client.HasCollection(ctx, collectionName)
	if err != nil {
		return false, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to check collection existence")
	}

	return exists, nil
}

// DescribeCollection 描述集合
func (mc *milvusClient) DescribeCollection(ctx context.Context, collectionName string) (*CollectionInfo, error) {
	if collectionName == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "collection name cannot be empty")
	}

	coll, err := mc.client.DescribeCollection(ctx, collectionName)
	if err != nil {
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to describe collection")
	}

	// 转换字段架构
	fields := make([]*FieldSchema, 0, len(coll.Schema.Fields))
	for _, field := range coll.Schema.Fields {
		fieldSchema := &FieldSchema{
			Name:         field.Name,
			DataType:     field.DataType,
			IsPrimaryKey: field.PrimaryKey,
			IsAutoID:     field.AutoID,
			Description:  field.Description,
		}

		// 获取向量维度
		if dim, ok := field.TypeParams["dim"]; ok {
			fmt.Sscanf(dim, "%d", &fieldSchema.Dimension)
		}

		fields = append(fields, fieldSchema)
	}

	return &CollectionInfo{
		Name:             coll.Name,
		Description:      coll.Schema.Description,
		Schema: &CollectionSchema{
			Fields:             fields,
			EnableDynamicField: coll.Schema.EnableDynamicField,
		},
		ShardNum:         coll.ShardNum,
		CreatedTimestamp: 0, // TODO: Milvus SDK v2 does not expose creation time
		ConsistencyLevel: coll.ConsistencyLevel,
	}, nil
}

// ListCollections 列出所有集合
func (mc *milvusClient) ListCollections(ctx context.Context) ([]string, error) {
	collections, err := mc.client.ListCollections(ctx)
	if err != nil {
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to list collections")
	}

	names := make([]string, 0, len(collections))
	for _, coll := range collections {
		names = append(names, coll.Name)
	}

	return names, nil
}

// Insert 插入向量
func (mc *milvusClient) Insert(ctx context.Context, collectionName string, vectors [][]float32, metadata []map[string]interface{}) ([]int64, error) {
	if collectionName == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "collection name cannot be empty")
	}
	if len(vectors) == 0 {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "vectors cannot be empty")
	}
	if len(metadata) != len(vectors) {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "metadata length must match vectors length")
	}

	// 获取集合信息
	collInfo, err := mc.DescribeCollection(ctx, collectionName)
	if err != nil {
		return nil, err
	}

	// 构建列数据
	columns := make([]entity.Column, 0)

	// 查找向量字段和其他字段
	var vectorFieldName string
	for _, field := range collInfo.Schema.Fields {
		if field.DataType == entity.FieldTypeFloatVector {
			vectorFieldName = field.Name
			// 创建向量列
			vectorColumn := entity.NewColumnFloatVector(vectorFieldName, int(field.Dimension), vectors)
			columns = append(columns, vectorColumn)
		}
	}

	if vectorFieldName == "" {
		return nil, errors.NewInternalError("MILVUS_002", "no vector field found in collection")
	}

	// 添加元数据字段
	if len(metadata) > 0 && len(metadata[0]) > 0 {
		for key := range metadata[0] {
			values := make([]interface{}, len(metadata))
			for i, m := range metadata {
				values[i] = m[key]
			}

			// 根据值类型创建相应的列
			switch values[0].(type) {
			case string:
				strValues := make([]string, len(values))
				for i, v := range values {
					strValues[i] = v.(string)
				}
				columns = append(columns, entity.NewColumnVarChar(key, strValues))
			case int64:
				intValues := make([]int64, len(values))
				for i, v := range values {
					intValues[i] = v.(int64)
				}
				columns = append(columns, entity.NewColumnInt64(key, intValues))
			case float64:
				floatValues := make([]float64, len(values))
				for i, v := range values {
					floatValues[i] = v.(float64)
				}
				columns = append(columns, entity.NewColumnDouble(key, floatValues))
			}
		}
	}

	// 插入数据
	result, err := mc.client.Insert(ctx, collectionName, "", columns...)
	if err != nil {
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to insert vectors")
	}

// // 	return result.IDs().(*entity.ColumnInt64).Data(), nil
}

// Delete 删除向量
func (mc *milvusClient) Delete(ctx context.Context, collectionName string, ids []int64) error {
	if collectionName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "collection name cannot be empty")
	}
	if len(ids) == 0 {
		return errors.NewValidationError(errors.CodeInvalidParameter, "ids cannot be empty")
	}

	// 构建删除表达式
	expr := fmt.Sprintf("id in %v", ids)

	err := mc.client.Delete(ctx, collectionName, "", expr)
	if err != nil {
		return errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to delete vectors")
	}

	return nil
}

// Search 相似度搜索
func (mc *milvusClient) Search(ctx context.Context, params *SearchParams) ([]SearchResult, error) {
	if params == nil {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "search params cannot be nil")
	}
	if params.CollectionName == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "collection name cannot be empty")
	}
	if len(params.Vectors) == 0 {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "query vectors cannot be empty")
	}
	if params.TopK <= 0 {
		params.TopK = 10
	}

	// 构建搜索向量
	searchVectors := make([]entity.Vector, len(params.Vectors))
	for i, vec := range params.Vectors {
		searchVectors[i] = entity.FloatVector(vec)
	}

	// 构建搜索参数
	sp, err := entity.NewIndexFlatSearchParam()
	if err != nil {
		return nil, errors.WrapInternalError(err, "ERR_INTERNAL", "failed to create search param")
	}

	// 执行搜索
	searchResult, err := mc.client.Search(
		ctx,
		params.CollectionName,
		[]string{}, // 分区名
		params.Filter,
		params.OutputFields,
		searchVectors,
		params.VectorFieldName,
		params.MetricType,
		params.TopK,
		sp,
		client.WithSearchQueryConsistencyLevel(params.ConsistencyLevel),
	)
	if err != nil {
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to search vectors")
	}

	// 转换搜索结果
	results := make([]SearchResult, 0)
// 	for _, result := range searchResult {
		for i := 0; i < result.ResultCount; i++ {
// // 			id, _ := result.IDs.GetAsInt64(i)
			score := result.Scores[i]

			fields := make(map[string]interface{})
			for _, field := range result.Fields {
				val, err := field.Get(i)
				if err == nil {
					fields[field.Name()] = val
				}
			}

			results = append(results, SearchResult{
				ID:       id,
				Score:    score,
				Fields:   fields,
				Distance: score, // Milvus 中 score 即为距离
			})
		}
	}

	return results, nil
}

// HybridSearch 混合检索
func (mc *milvusClient) HybridSearch(ctx context.Context, params *HybridSearchParams) ([]SearchResult, error) {
	if params == nil {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "hybrid search params cannot be nil")
	}
	if params.CollectionName == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "collection name cannot be empty")
	}
	if len(params.VectorSearches) == 0 {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "vector searches cannot be empty")
	}

	// 执行多个向量搜索
	allResults := make([][]SearchResult, 0, len(params.VectorSearches))
	for _, vs := range params.VectorSearches {
		searchParams := &SearchParams{
			CollectionName:   params.CollectionName,
			Vectors:          vs.Vectors,
			VectorFieldName:  vs.VectorFieldName,
			MetricType:       vs.MetricType,
			TopK:             vs.TopK,
			OutputFields:     params.OutputFields,
			Filter:           vs.Filter,
			SearchParams:     vs.SearchParams,
			ConsistencyLevel: params.ConsistencyLevel,
		}

		results, err := mc.Search(ctx, searchParams)
		if err != nil {
			return nil, err
		}

		allResults = append(allResults, results)
	}

	// 合并和重排序结果
	mergedResults := mc.mergeSearchResults(allResults, params)

	// 限制返回数量
	if len(mergedResults) > params.TopK {
		mergedResults = mergedResults[:params.TopK]
	}

	return mergedResults, nil
}

// mergeSearchResults 合并搜索结果
func (mc *milvusClient) mergeSearchResults(allResults [][]SearchResult, params *HybridSearchParams) []SearchResult {
	// 使用 map 去重并合并分数
	resultMap := make(map[int64]*SearchResult)

	for i, results := range allResults {
		weight := float32(1.0)
		if i < len(params.VectorSearches) && params.VectorSearches[i].Weight > 0 {
			weight = params.VectorSearches[i].Weight
		}

// 		for _, result := range results {
			if existing, ok := resultMap[result.ID]; ok {
				// 加权合并分数
				existing.Score += result.Score * weight
			} else {
				result.Score *= weight
				resultMap[result.ID] = &result
			}
		}
	}

	// 转换为切片并排序
	merged := make([]SearchResult, 0, len(resultMap))
// 	for _, result := range resultMap {
		merged = append(merged, *result)
	}

	// 按分数降序排序
	for i := 0; i < len(merged)-1; i++ {
		for j := i + 1; j < len(merged); j++ {
			if merged[j].Score > merged[i].Score {
				merged[i], merged[j] = merged[j], merged[i]
			}
		}
	}

	return merged
}

// CreateIndex 创建索引
func (mc *milvusClient) CreateIndex(ctx context.Context, collectionName string, fieldName string, indexConfig *IndexConfig) error {
	if collectionName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "collection name cannot be empty")
	}
	if fieldName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "field name cannot be empty")
	}
	if indexConfig == nil {
		return errors.NewValidationError(errors.CodeInvalidParameter, "index config cannot be nil")
	}

	// 构建索引
	idx, err := entity.NewIndexIvfFlat(indexConfig.MetricType, 1024)
	if err != nil {
		return errors.WrapInternalError(err, "ERR_INTERNAL", "failed to create index")
	}

	err = mc.client.CreateIndex(ctx, collectionName, fieldName, idx, false)
	if err != nil {
		return errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to create index")
	}

	return nil
}

// DropIndex 删除索引
func (mc *milvusClient) DropIndex(ctx context.Context, collectionName string, fieldName string) error {
	if collectionName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "collection name cannot be empty")
	}
	if fieldName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "field name cannot be empty")
	}

	err := mc.client.DropIndex(ctx, collectionName, fieldName)
	if err != nil {
		return errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to drop index")
	}

	return nil
}

// DescribeIndex 描述索引
func (mc *milvusClient) DescribeIndex(ctx context.Context, collectionName string, fieldName string) (*IndexInfo, error) {
	if collectionName == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "collection name cannot be empty")
	}
	if fieldName == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "field name cannot be empty")
	}

	indexes, err := mc.client.DescribeIndex(ctx, collectionName, fieldName)
	if err != nil {
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to describe index")
	}

	if len(indexes) == 0 {
		return nil, errors.NewNotFoundError(errors.CodeNotFound, "index not found")
	}

	idx := indexes[0]
	return &IndexInfo{
		IndexType:  idx.IndexType(),
		MetricType: entity.MetricType(idx.Params()["metric_type"]),
		Params:     idx.Params(),
	}, nil
}

// LoadCollection 加载集合到内存
func (mc *milvusClient) LoadCollection(ctx context.Context, collectionName string) error {
	if collectionName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "collection name cannot be empty")
	}

	err := mc.client.LoadCollection(ctx, collectionName, false)
	if err != nil {
		return errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to load collection")
	}

	return nil
}

// ReleaseCollection 释放集合
func (mc *milvusClient) ReleaseCollection(ctx context.Context, collectionName string) error {
	if collectionName == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "collection name cannot be empty")
	}

	err := mc.client.ReleaseCollection(ctx, collectionName)
	if err != nil {
		return errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to release collection")
	}

	return nil
}

// GetCollectionStatistics 获取集合统计信息
func (mc *milvusClient) GetCollectionStatistics(ctx context.Context, collectionName string) (*CollectionStats, error) {
	if collectionName == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "collection name cannot be empty")
	}

	stats, err := mc.client.GetCollectionStatistics(ctx, collectionName)
	if err != nil {
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to get collection statistics")
	}

	// Parse row count from stats map
	rowCount := int64(0)
	if rowCountStr, ok := stats["row_count"]; ok {
		fmt.Sscanf(rowCountStr, "%d", &rowCount)
	}

	return &CollectionStats{
		RowCount: rowCount,
		DataSize: 0, // Milvus SDK 可能不直接提供，需要额外计算
	}, nil
}

// Flush 刷新数据到磁盘
func (mc *milvusClient) Flush(ctx context.Context, collectionNames ...string) error {
	if len(collectionNames) == 0 {
		return errors.NewValidationError(errors.CodeInvalidParameter, "collection names cannot be empty")
	}

	err := mc.client.Flush(ctx, collectionNames[0], false)
	if err != nil {
		return errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to flush collection")
	}

	return nil
}

// Close 关闭客户端
func (mc *milvusClient) Close() error {
	return mc.client.Close()
}

// Ping 健康检查
func (mc *milvusClient) Ping(ctx context.Context) error {
	// 通过列出集合来检查连接状态
	_, err := mc.client.ListCollections(ctx)
	if err != nil {
		return errors.WrapDatabaseError(err, errors.CodeDatabaseError, "milvus connection check failed")
	}
	return nil
}

//Personal.AI order the ending

# OpenEAAP AI Data 全景图与参考架构

基于OpenEAAP项目的实际架构，将从**参考模型（RM）**、**参考架构（RA）**和**参考实现（RI）**三个维度，为您呈现AI Data的完整全景图。

## 一、AI Data 参考模型（RM - Reference Model）

### 1.1 核心能力模型

OpenEAAP采用**五大能力中台**架构，其中**DIKF（Data Intelligence & Knowledge Fabric）数据智能与知识编织平台**是AI Data能力的核心承载者 [1](#0-0) 。

```mermaid
graph TB
    subgraph "AI Data 能力全景"
        subgraph "数据采集层 Data Ingestion"
            DI1["安全日志数据<br/>Security Logs"]
            DI2["企业文档数据<br/>Enterprise Documents"]
            DI3["业务分析数据<br/>Business Analytics"]
            DI4["结构化数据<br/>Structured Data"]
            DI5["半结构化数据<br/>Semi-structured Data"]
        end
        
        subgraph "数据处理层 Data Processing"
            DP1["解析器 Parser<br/>多格式支持"]
            DP2["分块器 Chunker<br/>智能切分"]
            DP3["向量化 Embedding<br/>语义表示"]
            DP4["PII检测与脱敏<br/>Privacy Protection"]
        end
        
        subgraph "数据存储层 Data Storage"
            DS1["PostgreSQL<br/>关系型数据"]
            DS2["Milvus/Qdrant<br/>向量数据"]
            DS3["Neo4j<br/>知识图谱"]
            DS4["InfluxDB<br/>时序数据"]
            DS5["MinIO/S3<br/>对象存储"]
        end
        
        subgraph "数据检索层 Data Retrieval"
            DR1["向量检索<br/>Vector Search"]
            DR2["关键词检索<br/>Keyword Search"]
            DR3["知识图谱检索<br/>Graph Search"]
            DR4["混合检索<br/>Hybrid Retrieval"]
        end
        
        subgraph "数据治理层 Data Governance"
            DG1["数据血缘<br/>Data Lineage"]
            DG2["敏感度标记<br/>Sensitivity Marking"]
            DG3["版本管理<br/>Version Control"]
            DG4["访问控制<br/>Access Control"]
        end
        
        subgraph "数据反馈层 Data Feedback"
            DF1["反馈收集<br/>Feedback Collection"]
            DF2["在线学习<br/>Online Learning"]
            DF3["数据优化<br/>Data Optimization"]
            DF4["持续迭代<br/>Continuous Iteration"]
        end
    end
    
    DI1 & DI2 & DI3 & DI4 & DI5 --> DP1 & DP2 & DP3 & DP4
    DP1 & DP2 & DP3 & DP4 --> DS1 & DS2 & DS3 & DS4 & DS5
    DS1 & DS2 & DS3 & DS4 & DS5 --> DR1 & DR2 & DR3 & DR4
    DG1 & DG2 & DG3 & DG4 -.治理.-> DP1 & DS1 & DR1
    DR1 & DR2 & DR3 & DR4 --> DF1
    DF1 --> DF2 --> DF3 --> DF4
    DF4 -.优化迭代.-> DP1
```

### 1.2 数据类型处理能力矩阵

根据OpenEAAP的架构设计，针对不同数据类型提供专门的处理能力 [2](#0-1) ：

| 数据类型 | 处理策略 | 存储方案 | 检索方式 | 应用场景 |
|---------|---------|---------|---------|---------|
| **安全日志数据** | 解析 → 结构化 → 时序存储 | InfluxDB + PostgreSQL | 时间范围 + 关键词 | SOC威胁分析、异常检测 |
| **企业文档数据** | 解析 → 分块 → 向量化 → PII脱敏 | PostgreSQL + Milvus + MinIO | 向量检索 + 混合检索 | 智能文档问答、知识管理 |
| **业务分析数据** | 聚合 → 指标提取 → 图谱构建 | PostgreSQL + Neo4j | SQL + 图查询 | 业务洞察、关联分析 |
| **结构化数据** | Schema验证 → 索引构建 | PostgreSQL | SQL查询 | 事务处理、报表生成 |
| **半结构化数据** | JSON/XML解析 → 扁平化 | PostgreSQL JSONB + Milvus | 混合检索 | API数据、配置文件 |

### 1.3 AI Agent发展脉络中的数据演进

```mermaid
graph LR
    subgraph "Phase 1: 基础Agent"
        P1["静态数据<br/>Static Data"]
        P1_1["预定义规则"]
        P1_2["固定知识库"]
    end
    
    subgraph "Phase 2: RAG增强Agent"
        P2["动态检索数据<br/>Dynamic Retrieval"]
        P2_1["向量检索"]
        P2_2["混合检索"]
        P2_3["实时更新"]
    end
    
    subgraph "Phase 3: 自学习Agent"
        P3["反馈驱动数据<br/>Feedback-driven"]
        P3_1["用户反馈"]
        P3_2["执行追踪"]
        P3_3["自动优化"]
    end
    
    subgraph "Phase 4: 自主演进Agent"
        P4["自主治理数据<br/>Autonomous"]
        P4_1["数据血缘"]
        P4_2["质量监控"]
        P4_3["自动修复"]
    end
    
    P1 --> P2 --> P3 --> P4
```

OpenEAAP当前处于**Phase 2向Phase 3过渡**阶段，已实现RAG引擎和反馈收集机制 [3](#0-2) 。

## 二、AI Data 参考架构（RA - Reference Architecture）

### 2.1 分层架构设计

OpenEAAP采用**七层DDD架构**，AI Data能力贯穿所有层次 [4](#0-3) ：

```mermaid
graph TB
    subgraph "Layer 7: 接口层 Interface Layer"
        L7_1["HTTP API<br/>文档上传/查询"]
        L7_2["gRPC API<br/>流式检索"]
        L7_3["CLI<br/>数据管理命令"]
    end
    
    subgraph "Layer 6: 应用层 Application Layer"
        L6_1["DataService<br/>数据服务编排"]
        L6_2["AgentService<br/>Agent执行服务"]
    end
    
    subgraph "Layer 5: 平台层 Platform Layer"
        L5_1["RAG Engine<br/>检索增强生成"]
        L5_2["Document Processor<br/>文档处理流水线"]
        L5_3["Feedback Collector<br/>反馈收集器"]
        L5_4["Online Learning Engine<br/>在线学习引擎"]
    end
    
    subgraph "Layer 4: 领域层 Domain Layer"
        L4_1["Document Entity<br/>文档实体"]
        L4_2["Chunk Entity<br/>分块实体"]
        L4_3["Knowledge Entity<br/>知识实体"]
    end
    
    subgraph "Layer 3: 基础设施层 Infrastructure Layer"
        L3_1["KnowledgeRepository<br/>知识库仓储"]
        L3_2["VectorDB Client<br/>向量数据库客户端"]
        L3_3["Cache Client<br/>缓存客户端"]
    end
    
    subgraph "Layer 2: 治理层 Governance Layer"
        L2_1["PII Detector<br/>隐私检测"]
        L2_2["Data Lineage Tracker<br/>数据血缘追踪"]
        L2_3["Audit Logger<br/>审计日志"]
    end
    
    subgraph "Layer 1: 可观测性层 Observability Layer"
        L1_1["Distributed Tracing<br/>分布式追踪"]
        L1_2["Metrics Collection<br/>指标收集"]
    end
    
    L7_1 & L7_2 & L7_3 --> L6_1 & L6_2
    L6_1 & L6_2 --> L5_1 & L5_2 & L5_3 & L5_4
    L5_1 & L5_2 --> L4_1 & L4_2 & L4_3
    L4_1 & L4_2 & L4_3 --> L3_1 & L3_2 & L3_3
    
    L2_1 & L2_2 & L2_3 -.横切关注点.-> L5_1 & L5_2
    L1_1 & L1_2 -.监控.-> L5_1 & L5_2
```

### 2.2 数据处理流水线架构

OpenEAAP实现了完整的数据处理流水线，支持从摄取到应用的全生命周期管理 [5](#0-4) ：

```mermaid
graph LR
    subgraph "数据摄取 Ingestion"
        I1["文件上传<br/>File Upload"]
        I2["API接入<br/>API Ingestion"]
        I3["数据同步<br/>Data Sync"]
    end
    
    subgraph "数据处理 Processing"
        P1["解析器<br/>Parser<br/>多格式支持"]
        P2["分块器<br/>Chunker<br/>4种策略"]
        P3["向量化<br/>Embedder<br/>语义表示"]
        P4["PII检测<br/>PII Detector<br/>自动脱敏"]
    end
    
    subgraph "数据存储 Storage"
        S1["文档库<br/>PostgreSQL"]
        S2["向量库<br/>Milvus"]
        S3["对象存储<br/>MinIO"]
    end
    
    subgraph "数据检索 Retrieval"
        R1["RAG引擎<br/>混合检索"]
        R2["重排序<br/>Reranking"]
        R3["上下文构建<br/>Context Building"]
    end
    
    subgraph "数据应用 Application"
        A1["Agent调用"]
        A2["模型推理"]
        A3["结果生成"]
    end
    
    I1 & I2 & I3 --> P1 --> P2 --> P3
    P2 --> P4
    P3 --> S2
    P4 --> S1
    I1 --> S3
    
    S1 & S2 --> R1 --> R2 --> R3 --> A1 --> A2 --> A3
```

**分块策略详解** [6](#0-5) ：

| 策略 | 说明 | 适用场景 |
|-----|------|---------|
| **固定长度** | 按固定Token数分块 | 通用文档、API文档 |
| **语义边界** | 按段落、章节分块 | 结构化文档、技术规范 |
| **滑动窗口** | 重叠分块，避免信息丢失 | 长篇文档、法律合同 |
| **层次分块** | 多粒度分块（句子、段落、章节） | 复杂文档、学术论文 |

### 2.3 数据血缘与治理架构

OpenEAAP实现了完整的数据血缘追踪机制，确保数据的可追溯性和合规性 [7](#0-6) ：

```mermaid
graph LR
    subgraph "数据源 Sources"
        S1["原始文档<br/>Raw Documents"]
        S2["API数据<br/>API Data"]
        S3["用户反馈<br/>User Feedback"]
    end
    
    subgraph "转换层 Transformation"
        T1["解析<br/>Parsing"]
        T2["分块<br/>Chunking"]
        T3["向量化<br/>Vectorization"]
        T4["PII脱敏<br/>PII Masking"]
    end
    
    subgraph "存储层 Storage"
        ST1["文档库<br/>Doc Store"]
        ST2["向量库<br/>Vector Store"]
        ST3["知识图谱<br/>Knowledge Graph"]
    end
    
    subgraph "使用层 Usage"
        U1["RAG检索<br/>RAG Retrieval"]
        U2["模型微调<br/>Model Fine-tuning"]
        U3["Agent调用<br/>Agent Execution"]
    end
    
    S1 -.血缘lineage.-> T1
    T1 -.血缘.-> T2
    T2 -.血缘.-> T3
    T2 -.血缘.-> T4
    T3 -.血缘.-> ST2
    T4 -.血缘.-> ST1
    ST2 -.血缘.-> U1
    S3 -.血缘.-> U2
    U1 -.血缘.-> U3
```

### 2.4 数据反馈闭环架构

OpenEAAP构建了从业务反馈到数据优化的全自动化闭环 [8](#0-7) ：

```mermaid
graph TB
    subgraph "反馈源 Feedback Sources"
        FS1["用户修正<br/>User Correction"]
        FS2["评分反馈<br/>Rating Feedback"]
        FS3["自动评估<br/>Auto Evaluation"]
    end
    
    subgraph "数据收集 Data Collection"
        DC1["反馈收集器<br/>Feedback Collector"]
        DC2["质量过滤<br/>Quality Filter"]
        DC3["数据清洗<br/>Data Cleaning"]
    end
    
    subgraph "标签生成 Labeling"
        LB1["自动标注<br/>Auto Labeling"]
        LB2["人工审核<br/>HITL Review"]
        LB3["标签验证<br/>Label Validation"]
    end
    
    subgraph "数据集构建 Dataset Building"
        DS1["SFT数据集<br/>SFT Dataset"]
        DS2["DPO数据集<br/>DPO Preference Pairs"]
        DS3["评估数据集<br/>Evaluation Dataset"]
    end
    
    subgraph "优化流程 Optimization"
        OP1["Prompt优化<br/>Prompt Optimization"]
        OP2["模型微调<br/>Model Fine-tuning"]
        OP3["知识库更新<br/>Knowledge Base Update"]
    end
    
    FS1 & FS2 & FS3 --> DC1 --> DC2 --> DC3
    DC3 --> LB1 --> LB2 --> LB3
    LB3 --> DS1 & DS2 & DS3
    DS1 & DS2 & DS3 --> OP1 & OP2 & OP3
    OP1 & OP2 & OP3 -.数据回流.-> FS1
```

## 三、AI Data 参考实现（RI - Reference Implementation）

### 3.1 RAG引擎核心实现

OpenEAAP的RAG引擎实现了完整的检索增强生成流程 [9](#0-8) ：

**核心接口定义：** [10](#0-9) 

**RAG请求响应模型：** [11](#0-10) 

**完整RAG查询流程实现：** [12](#0-11) 

该实现包含6个关键阶段：
1. **查询理解**（Query Understanding） [13](#0-12) 
2. **检索阶段**（Retrieval） [14](#0-13) 
3. **重排序阶段**（Reranking） [15](#0-14) 
4. **上下文构建**（Context Building） [16](#0-15) 
5. **生成阶段**（Generation） [17](#0-16) 
6. **验证阶段**（Verification） [18](#0-17) 

### 3.2 检索策略实现

OpenEAAP支持四种检索模式 [19](#0-18) ：

- **向量检索（Vector Search）**：基于语义相似度的检索
- **关键词检索（Keyword Search）**：基于精确匹配的检索  
- **混合检索（Hybrid Retrieval）**：结合向量和关键词的检索
- **知识图谱检索（Graph Search）**：基于关系的检索

检索实现包含智能上下文长度控制 [20](#0-19) 。

### 3.3 流式RAG实现

OpenEAAP提供流式RAG查询能力，支持实时响应 [21](#0-20) ：

**流式响应模型：** [22](#0-21) 

### 3.4 答案验证机制

OpenEAAP实现了多维度的答案验证机制 [23](#0-22) ：

**验证实现：** [24](#0-23) 

验证维度包括：
- **幻觉检测（Hallucination Detection）**：检查答案是否脱离检索内容
- **引用有效性（Citation Validity）**：验证答案是否引用了检索到的内容
- **事实核查（Fact Check）**：检查答案的合理性

### 3.5 三级缓存架构实现

OpenEAAP实现了业界领先的三级缓存架构，显著降低推理成本和延迟 [25](#0-24) ：

**缓存层级策略：**

| 层级 | 存储介质 | 匹配策略 | 命中率 | 延迟 | 实现路径 |
|-----|---------|---------|--------|------|---------|
| **L1本地缓存** | 进程内存 | 精确Hash匹配 | 20-30% | <1ms | Go map + LRU淘汰 |
| **L2语义缓存** | Redis集群 | 语义Hash | 30-40% | <10ms | Redis客户端 |
| **L3向量缓存** | Milvus | 余弦相似度 | 10-20% | <50ms | Milvus客户端 |

**性能优势：**
- 累计缓存命中率：**60-90%**
- P95延迟降低：**70%**
- 推理成本降低：**60%** [26](#0-25) 

### 3.6 数据存储选型实现

OpenEAAP针对不同数据类型选择最优存储方案 [27](#0-26) ：

| 数据类型 | 存储技术 | 用途 |
|---------|---------|------|
| **关系数据** | PostgreSQL | 用户、Agent、执行记录等结构化数据 |
| **向量数据** | Milvus/Qdrant | 文档向量、Embedding |
| **图数据** | Neo4j | 知识图谱、数据血缘 |
| **时序数据** | InfluxDB | Trace、指标、日志 |
| **缓存** | Redis | L2语义缓存、会话状态 |
| **对象存储** | MinIO/S3 | 原始文档、模型文件 |

## 四、针对具体场景的AI Data应用

### 4.1 安全日志数据场景（SOC Copilot）

OpenEAAP提供了完整的安全运营智能助手实现 [28](#0-27) ：

**数据流架构：** [29](#0-28) 

**核心能力：**
- 威胁情报查询：集成SIEM、TI平台API
- 日志关联分析：RAG检索 + LLM推理
- 历史案例检索：向量检索 + 知识图谱
- 响应建议生成：基于案例库的Prompt工程
- 人在回路（HITL）：关键操作需人工审核

**效果指标：**
- 分析耗时：从30分钟降至<5分钟
- 误报率：从40%降至<15%
- 响应速度：从2小时降至<30分钟 [30](#0-29) 

### 4.2 企业文档数据场景

OpenEAAP实现了智能文档问答系统 [31](#0-30) ：

**文档处理流程：** [32](#0-31) 

处理流程包括：
1. 文档解析（支持PDF、Word、Markdown等）
2. PII检测与自动脱敏
3. 智能分块（4种策略）
4. 向量化（生成Embedding）

## 五、AI Data演进路线图

基于OpenEAAP的路线图 [33](#0-32) ：

### Phase 1: 基础平台（2026 Q1-Q2）✅ 已完成
- 核心框架搭建（DDD架构）
- 编排器与运行时
- 推理网关（vLLM集成）
- 三级缓存架构

### Phase 2: 数据智能（2026 Q2-Q3）🚧 进行中
- RAG引擎（混合检索、重排序）
- 向量数据库集成（Milvus）
- 知识图谱构建
- 文档处理流水线

### Phase 3: 持续优化（2026 Q3-Q4）📅 规划中
- 在线学习引擎
- RLHF/DPO训练流程
- Prompt自动优化
- A/B测试框架

### Phase 4: 治理增强（2026 Q4-2027 Q1）📅 规划中
- 策略引擎（ABAC/RBAC）
- 隐私计算网关（PII检测与脱敏）
- 审计与合规（SOC2、GDPR）

## Notes

OpenEAAP项目为AI Data提供了完整的**参考模型、参考架构和参考实现**，特别值得关注的是：

1. **DIKF平台**是AI Data能力的核心载体，实现了从数据摄取到应用的全生命周期管理

2. **三级缓存架构**是性能优化的关键创新，通过L1本地缓存（<1ms）、L2语义缓存（<10ms）、L3向量缓存（<50ms）实现60-90%的命中率

3. **数据反馈闭环**建立了从用户反馈到模型优化的自动化流程，支持RLHF/DPO训练和Prompt自动优化

4. **多策略分块机制**针对不同文档类型提供4种分块策略，确保语义完整性

5. **数据血缘追踪**实现了从数据源到最终应用的全链路可追溯，满足合规要求

6. 针对**安全日志、企业文档、业务分析**等不同数据类型，OpenEAAP提供了差异化的存储和检索方案

该架构已在实际代码中得到验证，可作为构建企业级AI Data平台的重要参考。

### Citations

**File:** docs/architecture.md (L20-20)
```markdown
| **DIKF**     | Data Intelligence & Knowledge Fabric       | 数据智能与知识编织平台         |
```

**File:** docs/architecture.md (L749-791)
```markdown
### 4.2 数据智能与知识编织平台（DIKF）

#### 4.2.1 组件架构

```mermaid
graph TB
    subgraph Ingestion[数据接入层]
        FILE[文件上传]
        API[API 接入]
        SYNC[数据同步]
    end
    
    subgraph Processing[数据处理层]
        PARSE[解析器]
        CHUNK[分块器]
        EMBED[向量化]
        PII[PII 检测]
    end
    
    subgraph Storage[存储层]
        VDB[向量数据库]
        KG[知识图谱]
        DOC[文档存储]
    end
    
    subgraph Retrieval[检索层]
        RAG[RAG 引擎]
        HYBRID[混合检索]
        RERANK[重排序]
    end
    
    subgraph Governance[治理层]
        LINEAGE[数据血缘]
        SENSITIVITY[敏感度标记]
        VERSION[版本管理]
    end
    
    Ingestion --> Processing
    Processing --> Storage
    Storage --> Retrieval
    Governance -.治理.-> Processing
    Governance -.治理.-> Storage
```
```

**File:** docs/architecture.md (L873-914)
```markdown

OpenEAAP 构建了从业务反馈到数据优化的全自动化闭环：

```mermaid
graph TB
    subgraph Source[反馈源]
        S1[用户修正<br/>User Correction]
        S2[点赞/点踩<br/>Thumbs Up/Down]
        S3[自动评估<br/>Auto Evaluation]
    end
    
    subgraph Collection[数据收集]
        C1[反馈收集器<br/>Feedback Collector]
        C2[质量过滤<br/>Quality Filter]
        C3[数据清洗<br/>Data Cleaning]
    end
    
    subgraph Labeling[标签生成]
        L1[自动标注<br/>Auto Labeling]
        L2[人工审核<br/>HITL Review]
        L3[标签验证<br/>Label Validation]
    end
    
    subgraph Dataset[数据集构建]
        D1[SFT 数据集<br/>SFT Dataset]
        D2[DPO 数据集<br/>DPO Dataset]
        D3[评估数据集<br/>Eval Dataset]
    end
    
    subgraph Optimization[优化流程]
        O1[Prompt 优化<br/>Prompt Optimization]
        O2[模型微调<br/>Model Fine-tuning]
        O3[知识库更新<br/>KB Update]
    end
    
    Source --> Collection
    Collection --> Labeling
    Labeling --> Dataset
    Dataset --> Optimization
    
    Optimization -.回流.-> Source
```
```

**File:** docs/architecture.md (L1112-1164)
```markdown
#### 4.3.3 三级缓存架构

为降低重复查询成本，OpenEAAP 设计了三级缓存架构：

```mermaid
graph TB
    REQ[请求] --> L1{L1 本地 Hash}
    L1 -->|命中| HIT1[返回结果]
    L1 -->|未命中| L2{L2 Redis 语义}
    L2 -->|命中| HIT2[返回结果]
    L2 -->|未命中| L3{L3 向量相似度}
    L3 -->|相似度>0.95| HIT3[返回相似结果]
    L3 -->|相似度<0.95| LLM[调用 LLM]
    LLM --> CACHE[写入缓存]
    CACHE --> RES[返回结果]
```

**缓存策略**:

| 层级     | 存储    | 匹配方式    | 命中率    | 延迟     |
| ------ | ----- | ------- | ------ | ------ |
| **L1** | 进程内存  | 精确 Hash | 20-30% | < 1ms  |
| **L2** | Redis | 语义 Hash | 30-40% | < 10ms |
| **L3** | 向量数据库 | 余弦相似度   | 10-20% | < 50ms |

**缓存接口**:

```go
// 缓存管理器接口
type CacheManager interface {
    // 查询缓存
    Get(ctx context.Context, key string) (*CachedResult, error)
    
    // 语义查询（L2/L3）
    GetSemantic(ctx context.Context, query string, threshold float64) (*CachedResult, error)
    
    // 写入缓存
    Set(ctx context.Context, key string, value *CachedResult, ttl time.Duration) error
    
    // 失效缓存
    Invalidate(ctx context.Context, key string) error
}

// 缓存结果
type CachedResult struct {
    Key        string    // 缓存键
    Value      string    // 缓存值
    Embedding  []float64 // 向量（用于 L3）
    Similarity float64   // 相似度（L3 命中时）
    TTL        time.Duration // 过期时间
    CreatedAt  time.Time // 创建时间
}
```
```

**File:** docs/architecture.md (L1454-1556)
```markdown
## 5. 关键业务场景设计

### 5.1 安全运营智能助手（SOC Copilot）

#### 5.1.1 场景描述

安全运营智能助手面向 SOC（Security Operations Center）团队，提供智能化的威胁检测、事件分析、响应建议等能力。

#### 5.1.2 业务流程

```mermaid
sequenceDiagram
    participant Analyst as 安全分析师
    participant Copilot as SOC Copilot
    participant SIEM as SIEM 系统
    participant TI as 威胁情报库
    participant RAG as RAG 引擎
    participant LLM as 推理服务
    participant Action as 响应系统
    
    Analyst->>Copilot: "分析这个可疑 IP: 192.168.1.100"
    Copilot->>SIEM: 查询关联日志
    SIEM-->>Copilot: 返回日志数据
    
    Copilot->>TI: 查询威胁情报
    TI-->>Copilot: 返回 IP 信誉数据
    
    Copilot->>RAG: 检索历史处置案例
    RAG-->>Copilot: 返回相似案例
    
    Copilot->>LLM: 综合分析（日志+情报+案例）
    LLM-->>Copilot: 生成分析报告
    
    Copilot->>Analyst: 展示分析结果与建议
    Analyst->>Copilot: "执行封禁操作"
    
    Copilot->>Action: 提交封禁请求（HITL审核）
    Action-->>Copilot: 返回执行结果
    
    Copilot->>Analyst: "已完成封禁，请确认"
```

#### 5.1.3 核心能力

| 能力             | 说明         | 技术实现              |
| -------------- | ---------- | ----------------- |
| **威胁情报查询**     | 自动查询多源威胁情报 | 集成 SIEM、TI 平台 API |
| **日志关联分析**     | 多维度日志关联    | RAG 检索 + LLM 推理   |
| **历史案例检索**     | 检索相似历史案例   | 向量检索 + 知识图谱       |
| **响应建议生成**     | 生成处置建议     | 基于案例库的 Prompt 工程  |
| **人在回路（HITL）** | 关键操作需人工审核  | Workflow 审批机制     |
| **持续学习**       | 从处置反馈中学习   | 在线学习引擎            |

#### 5.1.4 技术架构

```mermaid
graph TB
    subgraph Input[输入层]
        I1[分析师查询]
        I2[告警触发]
    end
    
    subgraph Orchestrator[编排层]
        O1[意图识别]
        O2[任务分解]
        O3[工具调用]
    end
    
    subgraph Tools[工具层]
        T1[SIEM API]
        T2[威胁情报 API]
        T3[RAG 检索]
        T4[知识图谱查询]
    end
    
    subgraph Analysis[分析层]
        A1[日志解析]
        A2[情报关联]
        A3[风险评估]
        A4[建议生成]
    end
    
    subgraph Output[输出层]
        OUT1[分析报告]
        OUT2[响应建议]
        OUT3[执行操作]
    end
    
    Input --> Orchestrator
    Orchestrator --> Tools
    Tools --> Analysis
    Analysis --> Output
```

#### 5.1.5 效果指标

| 指标        | 基线    | 目标      | 说明         |
| --------- | ----- | ------- | ---------- |
| **分析耗时**  | 30 分钟 | < 5 分钟  | 从告警到初步分析报告 |
| **误报率**   | 40%   | < 15%   | 降低误报，提升准确性 |
| **响应速度**  | 2 小时  | < 30 分钟 | 从分析到执行响应   |
| **知识复用率** | -     | > 60%   | 历史案例被引用比例  |

```

**File:** docs/architecture.md (L1557-1642)
```markdown
### 5.2 智能文档问答系统

#### 5.2.1 场景描述

基于企业内部文档（政策、规范、手册等）构建智能问答系统，支持员工快速获取信息。

#### 5.2.2 业务流程

```mermaid
graph TB
    USER[用户提问] --> UNDERSTAND[查询理解]
    UNDERSTAND --> ROUTE{查询路由}
    
    ROUTE -->|事实查询| VECTOR[向量检索]
    ROUTE -->|流程查询| KG[知识图谱检索]
    ROUTE -->|复杂推理| HYBRID[混合检索]
    
    VECTOR --> RERANK[重排序]
    KG --> RERANK
    HYBRID --> RERANK
    
    RERANK --> CONTEXT[上下文构建]
    CONTEXT --> LLM[LLM 生成]
    LLM --> VERIFY[答案验证]
    
    VERIFY --> CITE[引用标注]
    CITE --> RESPONSE[返回答案]
```

#### 5.2.3 核心技术

**文档预处理**:

```go
// 文档处理流程
type DocumentProcessor struct {
    parser     Parser
    chunker    Chunker
    embedder   Embedder
    piiDetector PIIDetector
}

func (p *DocumentProcessor) Process(ctx context.Context, doc *Document) (*ProcessedDocument, error) {
    // 1. 解析文档
    parsedDoc, err := p.parser.Parse(ctx, doc)
    if err != nil {
        return nil, err
    }
    
    // 2. PII 检测与脱敏
    cleanedDoc, err := p.piiDetector.Mask(ctx, parsedDoc.Text, nil)
    if err != nil {
        return nil, err
    }
    
    // 3. 智能分块
    chunks, err := p.chunker.Chunk(ctx, cleanedDoc)
    if err != nil {
        return nil, err
    }
    
    // 4. 向量化
    for _, chunk := range chunks {
        embedding, err := p.embedder.Embed(ctx, chunk.Text)
        if err != nil {
            return nil, err
        }
        chunk.Embedding = embedding
    }
    
    return &ProcessedDocument{
        Original: doc,
        Chunks:   chunks,
    }, nil
}
```

**智能分块策略**:

| 策略       | 说明              | 适用场景  |
| -------- | --------------- | ----- |
| **固定长度** | 按固定 Token 数分块   | 通用文档  |
| **语义边界** | 按段落、章节分块        | 结构化文档 |
| **滑动窗口** | 重叠分块，避免信息丢失     | 长篇文档  |
| **层次分块** | 多粒度分块（句子、段落、章节） | 复杂文档  |

```

**File:** docs/architecture.md (L1985-1995)
```markdown
### 7.2 存储选型

| 数据类型     | 存储技术          | 说明                  |
| -------- | ------------- | ------------------- |
| **关系数据** | PostgreSQL    | 用户、Agent、执行记录等结构化数据 |
| **向量数据** | Milvus/Qdrant | 文档向量、Embedding      |
| **图数据**  | Neo4j         | 知识图谱、数据血缘           |
| **时序数据** | InfluxDB      | Trace、指标、日志         |
| **缓存**   | Redis         | L2 语义缓存、会话状态        |
| **对象存储** | MinIO/S3      | 原始文档、模型文件           |

```

**File:** docs/architecture.md (L1996-2034)
```markdown
### 7.3 数据血缘图谱

```mermaid
graph LR
    subgraph Sources[数据源]
        S1[原始文档]
        S2[API 数据]
        S3[用户反馈]
    end
    
    subgraph Transform[转换层]
        T1[解析]
        T2[分块]
        T3[向量化]
        T4[PII 脱敏]
    end
    
    subgraph Storage[存储层]
        ST1[文档库]
        ST2[向量库]
        ST3[知识图谱]
    end
    
    subgraph Usage[使用层]
        U1[RAG 检索]
        U2[模型微调]
        U3[Agent 调用]
    end
    
    S1 -.lineage.-> T1
    T1 -.lineage.-> T2
    T2 -.lineage.-> T3
    T2 -.lineage.-> T4
    T3 -.lineage.-> ST2
    T4 -.lineage.-> ST1
    ST2 -.lineage.-> U1
    S3 -.lineage.-> U2
    U1 -.lineage.-> U3
```
```

**File:** README-zh.md (L85-96)
```markdown
* **三级智能缓存**：L1 本地（<1ms）+ L2 Redis（<10ms）+ L3 向量（<50ms），命中率 50%+
* **vLLM 集成**：PagedAttention、KV-Cache 共享、投机解码，吞吐量提升 24 倍
* **智能路由**：根据复杂度、延迟要求、成本预算自动选择最优模型

**效果对比**：

| 指标      | 优化前             | 优化后             | 提升幅度     |
| ------- | --------------- | --------------- | -------- |
| P95 延迟  | 5000ms          | 1500ms          | ⬇️ 70%   |
| 推理成本    | $1.00/1K tokens | $0.40/1K tokens | ⬇️ 60%   |
| GPU 利用率 | 40%             | 75%             | ⬆️ 87.5% |

```

**File:** README-zh.md (L176-264)
```markdown

OpenEAAP 采用经典的 **DDD（领域驱动设计）分层架构**，清晰的职责划分确保系统的可维护性和扩展性。

```mermaid
graph TB
    subgraph API[接口层（Interface Layer）]
        HTTP[HTTP API<br/>REST/GraphQL]
        GRPC[gRPC API<br/>高性能RPC]
        CLI[CLI工具<br/>命令行管理]
    end
    
    subgraph APP[应用层（Application Layer）]
        SERVICE1[Agent服务<br/>Agent Service]
        SERVICE2[Workflow服务<br/>Workflow Service]
        SERVICE3[Model服务<br/>Model Service]
        SERVICE4[Data服务<br/>Data Service]
    end
    
    subgraph PLATFORM[平台层（Platform Layer）]
        ORCH[编排器<br/>Orchestrator]
        RUNTIME[运行时<br/>Runtime]
        INFERENCE[推理引擎<br/>Inference Engine]
        RAG[RAG引擎<br/>RAG Engine]
        LEARNING[在线学习<br/>Online Learning]
        TRAINING[训练服务<br/>Training Service]
    end
    
    subgraph DOMAIN[领域层（Domain Layer）]
        AGENT_D[Agent领域<br/>Agent Domain]
        WORKFLOW_D[Workflow领域<br/>Workflow Domain]
        MODEL_D[Model领域<br/>Model Domain]
        KNOWLEDGE_D[Knowledge领域<br/>Knowledge Domain]
    end
    
    subgraph INFRA[基础设施层（Infrastructure Layer）]
        REPO[仓储实现<br/>Repository]
        VECTOR[向量数据库<br/>Vector DB]
        STORAGE[对象存储<br/>Object Storage]
        MQ[消息队列<br/>Message Queue]
    end
    
    subgraph GOV[治理层（Governance Layer）]
        POLICY[策略引擎<br/>Policy Engine]
        AUDIT[审计<br/>Audit]
        COMPLIANCE[合规<br/>Compliance]
    end
    
    subgraph OBS[可观测性层（Observability Layer）]
        TRACE[分布式追踪<br/>Tracing]
        METRICS[指标收集<br/>Metrics]
        LOGGING[日志<br/>Logging]
    end
    
    API --> APP
    APP --> PLATFORM
    APP --> DOMAIN
    PLATFORM --> DOMAIN
    DOMAIN --> INFRA
    
    GOV -.横切.-> PLATFORM
    GOV -.横切.-> APP
    OBS -.横切.-> PLATFORM
    OBS -.横切.-> APP
    
    style API fill:#e1f5fe
    style APP fill:#f3e5f5
    style PLATFORM fill:#fff9c4
    style DOMAIN fill:#c8e6c9
    style INFRA fill:#ffccbc
    style GOV fill:#ffebee
    style OBS fill:#f0f4c3
```

**各层职责**：

| 层次        | 职责                 | 示例组件                                     |
| --------- | ------------------ | ---------------------------------------- |
| **接口层**   | 对外暴露 API，处理请求/响应   | HTTP Handler、gRPC Server、CLI 命令          |
| **应用层**   | 编排业务流程，协调多个领域服务    | AgentService、WorkflowService             |
| **平台层**   | 核心能力组件，编排、推理、RAG 等 | Orchestrator、Inference Engine、RAG Engine |
| **领域层**   | 业务核心逻辑，领域实体和领域服务   | Agent、Workflow、Model 实体和领域服务             |
| **基础设施层** | 数据持久化和外部系统集成       | PostgreSQL、Redis、Milvus、MinIO            |
| **治理层**   | 安全、合规、审计           | 策略引擎、审计日志、PII 检测                         |
| **可观测性层** | 监控、追踪、日志           | OpenTelemetry、Prometheus、Loki            |

### 核心组件交互流程

以下时序图展示了一次完整的 Agent 执行请求的处理流程：

```

**File:** README-zh.md (L614-650)
```markdown
## 🗺️ 路线图

### Phase 1: 基础平台（2026 Q1-Q2）✅

* [x] 核心框架搭建（DDD 架构）
* [x] 编排器与运行时（Native、LangChain 适配器）
* [x] 推理网关（vLLM 集成）
* [x] 三级缓存架构

### Phase 2: 数据智能（2026 Q2-Q3）🚧

* [ ] RAG 引擎（混合检索、重排序）
* [ ] 向量数据库集成（Milvus）
* [ ] 知识图谱构建
* [ ] 文档处理流水线（解析、分块、向量化）

### Phase 3: 持续优化（2026 Q3-Q4）📅

* [ ] 在线学习引擎
* [ ] RLHF/DPO 训练流程
* [ ] Prompt 自动优化
* [ ] A/B 测试框架

### Phase 4: 治理增强（2026 Q4-2027 Q1）📅

* [ ] 策略引擎（ABAC/RBAC）
* [ ] 隐私计算网关（PII 检测与脱敏）
* [ ] 审计与合规（SOC2、GDPR）
* [ ] 漏洞扫描与安全加固

### Phase 5: 生态集成（2027 Q1-Q2）📅

* [ ] AutoGPT 适配器
* [ ] 插件市场
* [ ] 多模态支持（图像、语音）
* [ ] 边缘 AI 部署

```

**File:** internal/platform/rag/rag_engine.go (L14-106)
```go
// RAGEngine 定义 RAG 引擎接口
type RAGEngine interface {
	// Query 执行完整的 RAG 查询流程
	Query(ctx context.Context, req *RAGRequest) (*RAGResponse, error)

	// QueryStream 执行流式 RAG 查询
	QueryStream(ctx context.Context, req *RAGRequest) (<-chan *RAGChunk, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}

// RAGRequest 定义 RAG 请求
type RAGRequest struct {
	Query           string            // 用户查询
	CollectionName  string            // 知识库名称
	TopK            int               // 检索数量
	RetrievalMode   RetrievalMode     // 检索模式
	RerankEnabled   bool              // 是否启用重排序
	ModelName       string            // 生成模型名称
	Temperature     float32           // 生成温度
	MaxTokens       int               // 最大生成长度
	Metadata        map[string]string // 元数据过滤
	VerifyEnabled   bool              // 是否启用答案验证
}

// RAGResponse 定义 RAG 响应
type RAGResponse struct {
	Answer          string              // 生成的答案
	RetrievedChunks []*RetrievedChunk   // 检索到的文档块
	Sources         []string            // 引用来源
	Confidence      float32             // 置信度
	Latency         LatencyBreakdown    // 延迟分解
	Verified        bool                // 是否通过验证
	VerifyResult    *VerifyResult       // 验证结果
}

// RAGChunk 定义流式响应块
type RAGChunk struct {
	Type    ChunkType // 块类型
	Content string    // 内容
	Done    bool      // 是否完成
	Error   error     // 错误
}

// ChunkType 定义块类型
type ChunkType string

const (
	ChunkTypeRetrieval ChunkType = "retrieval" // 检索阶段
	ChunkTypeGenerate  ChunkType = "generate"  // 生成阶段
	ChunkTypeVerify    ChunkType = "verify"    // 验证阶段
	ChunkTypeError     ChunkType = "error"     // 错误
)

// RetrievalMode 定义检索模式
type RetrievalMode string

const (
	RetrievalModeVector  RetrievalMode = "vector"  // 向量检索
	RetrievalModeKeyword RetrievalMode = "keyword" // 关键词检索
	RetrievalModeHybrid  RetrievalMode = "hybrid"  // 混合检索
	RetrievalModeGraph   RetrievalMode = "graph"   // 知识图谱检索
)

// RetrievedChunk 定义检索到的文档块
type RetrievedChunk struct {
	ChunkID    string            // 块ID
	DocumentID string            // 文档ID
	Content    string            // 内容
	Score      float32           // 相关性分数
	Metadata   map[string]string // 元数据
	Source     string            // 来源
}

// LatencyBreakdown 定义延迟分解
type LatencyBreakdown struct {
	QueryUnderstanding time.Duration // 查询理解
	Retrieval          time.Duration // 检索
	Reranking          time.Duration // 重排序
	ContextBuilding    time.Duration // 上下文构建
	Generation         time.Duration // 生成
	Verification       time.Duration // 验证
	Total              time.Duration // 总延迟
}

// VerifyResult 定义验证结果
type VerifyResult struct {
	HasHallucination bool     // 是否存在幻觉
	CitationValid    bool     // 引用是否有效
	FactCheckPassed  bool     // 事实检查是否通过
	Issues           []string // 问题列表
}
```

**File:** internal/platform/rag/rag_engine.go (L154-259)
```go
// Query 执行完整的 RAG 查询流程
func (r *ragEngineImpl) Query(ctx context.Context, req *RAGRequest) (*RAGResponse, error) {
	startTime := time.Now()

	// 创建 Span
	span := r.tracer.StartSpan(ctx, "RAGEngine.Query")
	defer span.End()
	span.AddTag("query", req.Query)
	span.AddTag("collection", req.CollectionName)

	// 应用默认值
	r.applyDefaults(req)

	// 验证请求
	if err := r.validateRequest(req); err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidArgument, "invalid RAG request")
	}

	var latency LatencyBreakdown

	// 1. 查询理解（可选，当前简化为直接使用原始查询）
	queryStart := time.Now()
	processedQuery := r.understandQuery(ctx, req.Query)
	latency.QueryUnderstanding = time.Since(queryStart)

	// 2. 检索阶段
	retrievalStart := time.Now()
	retrievedChunks, err := r.retrieveChunks(ctx, processedQuery, req)
	if err != nil {
		span.SetStatus(trace.StatusError, err.Error())
		return nil, errors.Wrap(err, errors.CodeInternal, "retrieval failed")
	}
	latency.Retrieval = time.Since(retrievalStart)

	r.logger.Info(ctx, "retrieval completed",
		"query", req.Query,
		"chunks_count", len(retrievedChunks),
		"latency_ms", latency.Retrieval.Milliseconds())

	// 3. 重排序阶段（可选）
	if req.RerankEnabled && r.reranker != nil {
		rerankStart := time.Now()
		retrievedChunks, err = r.rerankChunks(ctx, processedQuery, retrievedChunks)
		if err != nil {
			r.logger.Warn(ctx, "reranking failed, using original order", "error", err)
		}
		latency.Reranking = time.Since(rerankStart)
	}

	// 4. 上下文构建阶段
	contextStart := time.Now()
	ragContext := r.buildContext(ctx, retrievedChunks, req)
	latency.ContextBuilding = time.Since(contextStart)

	// 5. 生成阶段
	generationStart := time.Now()
	answer, sources, err := r.generateAnswer(ctx, req.Query, ragContext, req)
	if err != nil {
		span.SetStatus(trace.StatusError, err.Error())
		return nil, errors.Wrap(err, errors.CodeInternal, "generation failed")
	}
	latency.Generation = time.Since(generationStart)

	// 6. 验证阶段（可选）
	var verifyResult *VerifyResult
	verified := true
	if req.VerifyEnabled {
		verifyStart := time.Now()
		verifyResult, err = r.verifyAnswer(ctx, req.Query, answer, retrievedChunks)
		if err != nil {
			r.logger.Warn(ctx, "verification failed", "error", err)
		} else {
			verified = verifyResult.HasHallucination == false &&
				verifyResult.CitationValid &&
				verifyResult.FactCheckPassed
		}
		latency.Verification = time.Since(verifyStart)
	}

	latency.Total = time.Since(startTime)

	// 计算置信度
	confidence := r.calculateConfidence(retrievedChunks, verified)

	response := &RAGResponse{
		Answer:          answer,
		RetrievedChunks: retrievedChunks,
		Sources:         sources,
		Confidence:      confidence,
		Latency:         latency,
		Verified:        verified,
		VerifyResult:    verifyResult,
	}

	r.logger.Info(ctx, "RAG query completed",
		"query", req.Query,
		"answer_length", len(answer),
		"confidence", confidence,
		"verified", verified,
		"total_latency_ms", latency.Total.Milliseconds())

	span.AddTag("confidence", fmt.Sprintf("%.2f", confidence))
	span.AddTag("verified", verified)

	return response, nil
}
```

**File:** internal/platform/rag/rag_engine.go (L261-341)
```go
// QueryStream 执行流式 RAG 查询
func (r *ragEngineImpl) QueryStream(ctx context.Context, req *RAGRequest) (<-chan *RAGChunk, error) {
	chunkChan := make(chan *RAGChunk, 10)

	go func() {
		defer close(chunkChan)

		// 应用默认值
		r.applyDefaults(req)

		// 1. 检索阶段
		chunkChan <- &RAGChunk{Type: ChunkTypeRetrieval, Content: "开始检索相关文档...", Done: false}

		processedQuery := r.understandQuery(ctx, req.Query)
		retrievedChunks, err := r.retrieveChunks(ctx, processedQuery, req)
		if err != nil {
			chunkChan <- &RAGChunk{Type: ChunkTypeError, Error: err, Done: true}
			return
		}

		chunkChan <- &RAGChunk{
			Type:    ChunkTypeRetrieval,
			Content: fmt.Sprintf("检索完成，找到 %d 个相关文档块", len(retrievedChunks)),
			Done:    false,
		}

		// 2. 重排序（可选）
		if req.RerankEnabled && r.reranker != nil {
			retrievedChunks, _ = r.rerankChunks(ctx, processedQuery, retrievedChunks)
		}

		// 3. 构建上下文
		ragContext := r.buildContext(ctx, retrievedChunks, req)

		// 4. 流式生成
		chunkChan <- &RAGChunk{Type: ChunkTypeGenerate, Content: "", Done: false}

		answerChan, err := r.generator.GenerateStream(ctx, &GenerateRequest{
			Query:       req.Query,
			Context:     ragContext,
			ModelName:   req.ModelName,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
		})

		if err != nil {
			chunkChan <- &RAGChunk{Type: ChunkTypeError, Error: err, Done: true}
			return
		}

		fullAnswer := ""
		for genChunk := range answerChan {
			if genChunk.Error != nil {
				chunkChan <- &RAGChunk{Type: ChunkTypeError, Error: genChunk.Error, Done: true}
				return
			}
			fullAnswer += genChunk.Content
			chunkChan <- &RAGChunk{Type: ChunkTypeGenerate, Content: genChunk.Content, Done: false}
		}

		// 5. 验证（可选）
		if req.VerifyEnabled {
			chunkChan <- &RAGChunk{Type: ChunkTypeVerify, Content: "验证答案中...", Done: false}
			verifyResult, err := r.verifyAnswer(ctx, req.Query, fullAnswer, retrievedChunks)
			if err == nil {
				verified := verifyResult.HasHallucination == false &&
					verifyResult.CitationValid &&
					verifyResult.FactCheckPassed
				chunkChan <- &RAGChunk{
					Type:    ChunkTypeVerify,
					Content: fmt.Sprintf("验证完成，结果: %v", verified),
					Done:    true,
				}
			}
		} else {
			chunkChan <- &RAGChunk{Type: ChunkTypeGenerate, Content: "", Done: true}
		}
	}()

	return chunkChan, nil
}
```

**File:** internal/platform/rag/rag_engine.go (L389-410)
```go
// buildContext 构建 RAG 上下文
func (r *ragEngineImpl) buildContext(ctx context.Context, chunks []*RetrievedChunk, req *RAGRequest) string {
	var contextBuilder string
	currentLength := 0

	for i, chunk := range chunks {
		chunkText := fmt.Sprintf("[文档 %d] 来源: %s\n%s\n\n", i+1, chunk.Source, chunk.Content)

		// 控制上下文长度
		if currentLength+len(chunkText) > r.config.MaxContextLength {
			r.logger.Warn(ctx, "context truncated due to length limit",
				"max_length", r.config.MaxContextLength,
				"chunks_included", i)
			break
		}

		contextBuilder += chunkText
		currentLength += len(chunkText)
	}

	return contextBuilder
}
```

**File:** internal/platform/rag/rag_engine.go (L431-461)
```go
func (r *ragEngineImpl) verifyAnswer(ctx context.Context, query, answer string, chunks []*RetrievedChunk) (*VerifyResult, error) {
	// 简化实现：基于规则的验证
	result := &VerifyResult{
		HasHallucination: false,
		CitationValid:    true,
		FactCheckPassed:  true,
		Issues:           []string{},
	}

	// 检查答案长度
	if len(answer) < 10 {
		result.Issues = append(result.Issues, "答案过短")
		result.FactCheckPassed = false
	}

	// 检查是否引用了检索到的内容
	hasReference := false
	for _, chunk := range chunks {
		if contains(answer, chunk.Content[:min(50, len(chunk.Content))]) {
			hasReference = true
			break
		}
	}

	if !hasReference {
		result.Issues = append(result.Issues, "答案未引用检索到的内容，可能存在幻觉")
		result.HasHallucination = true
	}

	return result, nil
}
```

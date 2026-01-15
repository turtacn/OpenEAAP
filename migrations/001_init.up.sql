-- migrations/001_init.up.sql
-- OpenEAAP 初始化数据库迁移（向上）
-- 创建所有核心表、索引、外键约束

-- ============================================
-- 1. 扩展与类型定义
-- ============================================

-- 启用 UUID 扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 启用 pg_trgm 扩展（用于文本相似度搜索）
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- 定义枚举类型
CREATE TYPE agent_status AS ENUM ('draft', 'active', 'paused', 'archived', 'error');
CREATE TYPE runtime_type AS ENUM ('native', 'langchain', 'autogpt', 'plugin');
CREATE TYPE workflow_status AS ENUM ('pending', 'running', 'paused', 'completed', 'failed', 'cancelled');
CREATE TYPE step_status AS ENUM ('pending', 'running', 'completed', 'failed', 'skipped');
CREATE TYPE model_type AS ENUM ('text', 'embedding', 'multimodal', 'vision', 'audio');
CREATE TYPE model_status AS ENUM ('active', 'inactive', 'deploying', 'error');
CREATE TYPE training_type AS ENUM ('sft', 'rlhf', 'dpo', 'lora', 'full_finetuning');
CREATE TYPE training_status AS ENUM ('pending', 'running', 'completed', 'failed', 'cancelled');
CREATE TYPE document_status AS ENUM ('uploading', 'processing', 'indexed', 'error');
CREATE TYPE sensitivity_level AS ENUM ('public', 'internal', 'confidential', 'restricted');
CREATE TYPE audit_action AS ENUM ('create', 'read', 'update', 'delete', 'execute', 'deploy', 'train');
CREATE TYPE decision_type AS ENUM ('permit', 'deny', 'conditional');

-- ============================================
-- 2. 核心业务表
-- ============================================

-- Agents 表
CREATE TABLE agents (
   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
   name VARCHAR(255) NOT NULL,
   description TEXT,
   runtime_type runtime_type NOT NULL DEFAULT 'native',
   status agent_status NOT NULL DEFAULT 'draft',
   config JSONB NOT NULL DEFAULT '{}',
   metadata JSONB DEFAULT '{}',
   version INTEGER NOT NULL DEFAULT 1,
   created_by UUID,
   created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
   updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
   deleted_at TIMESTAMP WITH TIME ZONE,

   CONSTRAINT agents_name_unique UNIQUE (name, deleted_at)
);

CREATE INDEX idx_agents_status ON agents(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_agents_runtime_type ON agents(runtime_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_agents_created_by ON agents(created_by) WHERE deleted_at IS NULL;
CREATE INDEX idx_agents_created_at ON agents(created_at DESC);
CREATE INDEX idx_agents_name_trgm ON agents USING gin(name gin_trgm_ops) WHERE deleted_at IS NULL;

-- Workflows 表
CREATE TABLE workflows (
   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
   name VARCHAR(255) NOT NULL,
   description TEXT,
   agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
   status workflow_status NOT NULL DEFAULT 'pending',
   trigger JSONB NOT NULL DEFAULT '{}',
   config JSONB DEFAULT '{}',
   metadata JSONB DEFAULT '{}',
   started_at TIMESTAMP WITH TIME ZONE,
   completed_at TIMESTAMP WITH TIME ZONE,
   created_by UUID,
   created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
   updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
   deleted_at TIMESTAMP WITH TIME ZONE,

   CONSTRAINT workflows_name_unique UNIQUE (name, deleted_at)
);

CREATE INDEX idx_workflows_agent_id ON workflows(agent_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_workflows_status ON workflows(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_workflows_created_at ON workflows(created_at DESC);

-- Workflow Steps 表
CREATE TABLE workflow_steps (
   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
   workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
   name VARCHAR(255) NOT NULL,
   step_type VARCHAR(100) NOT NULL,
   step_order INTEGER NOT NULL,
   status step_status NOT NULL DEFAULT 'pending',
   config JSONB NOT NULL DEFAULT '{}',
   input JSONB DEFAULT '{}',
   output JSONB DEFAULT '{}',
   error_message TEXT,
   started_at TIMESTAMP WITH TIME ZONE,
   completed_at TIMESTAMP WITH TIME ZONE,
   created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
   updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

   CONSTRAINT workflow_steps_order_unique UNIQUE (workflow_id, step_order)
);

CREATE INDEX idx_workflow_steps_workflow_id ON workflow_steps(workflow_id);
CREATE INDEX idx_workflow_steps_status ON workflow_steps(status);
CREATE INDEX idx_workflow_steps_order ON workflow_steps(workflow_id, step_order);

-- Models 表
CREATE TABLE models (
   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
   name VARCHAR(255) NOT NULL UNIQUE,
   display_name VARCHAR(255) NOT NULL,
   model_type model_type NOT NULL,
   status model_status NOT NULL DEFAULT 'active',
   provider VARCHAR(100) NOT NULL,
   version VARCHAR(50),
   endpoint TEXT,
   config JSONB NOT NULL DEFAULT '{}',
   capabilities JSONB DEFAULT '{}',
   pricing JSONB DEFAULT '{}',
   metadata JSONB DEFAULT '{}',
   created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
   updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
   deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_models_status ON models(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_models_type ON models(model_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_models_provider ON models(provider) WHERE deleted_at IS NULL;

-- Model Metrics 表（模型性能指标）
CREATE TABLE model_metrics (
   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
   model_id UUID NOT NULL REFERENCES models(id) ON DELETE CASCADE,
   metric_type VARCHAR(100) NOT NULL,
   value NUMERIC(10, 4) NOT NULL,
   metadata JSONB DEFAULT '{}',
   recorded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_model_metrics_model_id ON model_metrics(model_id);
CREATE INDEX idx_model_metrics_type ON model_metrics(metric_type);
CREATE INDEX idx_model_metrics_recorded_at ON model_metrics(recorded_at DESC);

-- ============================================
-- 3. 训练相关表
-- ============================================

-- Training Jobs 表
CREATE TABLE training_jobs (
   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
   name VARCHAR(255) NOT NULL,
   base_model_id UUID NOT NULL REFERENCES models(id),
   training_type training_type NOT NULL,
   status training_status NOT NULL DEFAULT 'pending',
   config JSONB NOT NULL DEFAULT '{}',
   dataset_path TEXT,
   output_path TEXT,
   metrics JSONB DEFAULT '{}',
   error_message TEXT,
   started_at TIMESTAMP WITH TIME ZONE,
   completed_at TIMESTAMP WITH TIME ZONE,
   created_by UUID,
   created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
   updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_training_jobs_status ON training_jobs(status);
CREATE INDEX idx_training_jobs_type ON training_jobs(training_type);
CREATE INDEX idx_training_jobs_base_model ON training_jobs(base_model_id);
CREATE INDEX idx_training_jobs_created_at ON training_jobs(created_at DESC);

-- Feedback 表（用户反馈）
CREATE TABLE feedback (
   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
   agent_id UUID REFERENCES agents(id) ON DELETE CASCADE,
   workflow_id UUID REFERENCES workflows(id) ON DELETE CASCADE,
   request_id UUID,
   feedback_type VARCHAR(50) NOT NULL,
   rating INTEGER CHECK (rating >= 1 AND rating <= 5),
   comment TEXT,
   metadata JSONB DEFAULT '{}',
   user_id UUID,
   created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_feedback_agent_id ON feedback(agent_id);
CREATE INDEX idx_feedback_workflow_id ON feedback(workflow_id);
CREATE INDEX idx_feedback_request_id ON feedback(request_id);
CREATE INDEX idx_feedback_type ON feedback(feedback_type);
CREATE INDEX idx_feedback_created_at ON feedback(created_at DESC);

-- ============================================
-- 4. 知识库相关表
-- ============================================

-- Documents 表
CREATE TABLE documents (
   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
   name VARCHAR(255) NOT NULL,
   source_type VARCHAR(100) NOT NULL,
   source_path TEXT NOT NULL,
   content_type VARCHAR(100),
   file_size BIGINT,
   status document_status NOT NULL DEFAULT 'uploading',
   sensitivity_level sensitivity_level NOT NULL DEFAULT 'internal',
   metadata JSONB DEFAULT '{}',
   checksum VARCHAR(64),
   uploaded_by UUID,
   created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
   updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
   deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_documents_status ON documents(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_documents_sensitivity ON documents(sensitivity_level) WHERE deleted_at IS NULL;
CREATE INDEX idx_documents_source_type ON documents(source_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_documents_created_at ON documents(created_at DESC);

-- Document Chunks 表（文档分块）
CREATE TABLE document_chunks (
   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
   document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
   chunk_index INTEGER NOT NULL,
   content TEXT NOT NULL,
   content_hash VARCHAR(64),
   token_count INTEGER,
   vector_id VARCHAR(255),
   metadata JSONB DEFAULT '{}',
   created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

   CONSTRAINT document_chunks_unique UNIQUE (document_id, chunk_index)
);

CREATE INDEX idx_document_chunks_document_id ON document_chunks(document_id);
CREATE INDEX idx_document_chunks_vector_id ON document_chunks(vector_id);
CREATE INDEX idx_document_chunks_content_hash ON document_chunks(content_hash);

-- Knowledge Graphs 表（知识图谱）
CREATE TABLE knowledge_graphs (
   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
   name VARCHAR(255) NOT NULL UNIQUE,
   description TEXT,
   config JSONB DEFAULT '{}',
   metadata JSONB DEFAULT '{}',
   created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
   updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
   deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_knowledge_graphs_name ON knowledge_graphs(name) WHERE deleted_at IS NULL;

-- Knowledge Graph Entities 表
CREATE TABLE kg_entities (
   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
   graph_id UUID NOT NULL REFERENCES knowledge_graphs(id) ON DELETE CASCADE,
   entity_type VARCHAR(100) NOT NULL,
   name VARCHAR(255) NOT NULL,
   properties JSONB DEFAULT '{}',
   vector_id VARCHAR(255),
   created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
   updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

   CONSTRAINT kg_entities_unique UNIQUE (graph_id, entity_type, name)
);

CREATE INDEX idx_kg_entities_graph_id ON kg_entities(graph_id);
CREATE INDEX idx_kg_entities_type ON kg_entities(entity_type);
CREATE INDEX idx_kg_entities_name ON kg_entities(name);

-- Knowledge Graph Relations 表
CREATE TABLE kg_relations (
   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
   graph_id UUID NOT NULL REFERENCES knowledge_graphs(id) ON DELETE CASCADE,
   source_entity_id UUID NOT NULL REFERENCES kg_entities(id) ON DELETE CASCADE,
   target_entity_id UUID NOT NULL REFERENCES kg_entities(id) ON DELETE CASCADE,
   relation_type VARCHAR(100) NOT NULL,
   properties JSONB DEFAULT '{}',
   weight NUMERIC(5, 4) DEFAULT 1.0,
   created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

   CONSTRAINT kg_relations_unique UNIQUE (graph_id, source_entity_id, target_entity_id, relation_type)
);

CREATE INDEX idx_kg_relations_graph_id ON kg_relations(graph_id);
CREATE INDEX idx_kg_relations_source ON kg_relations(source_entity_id);
CREATE INDEX idx_kg_relations_target ON kg_relations(target_entity_id);
CREATE INDEX idx_kg_relations_type ON kg_relations(relation_type);

-- ============================================
-- 5. 治理与审计表
-- ============================================

-- Policies 表（策略定义）
CREATE TABLE policies (
   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
   name VARCHAR(255) NOT NULL UNIQUE,
   description TEXT,
   policy_type VARCHAR(100) NOT NULL,
   rules JSONB NOT NULL,
   decision_default decision_type NOT NULL DEFAULT 'deny',
   priority INTEGER NOT NULL DEFAULT 0,
   enabled BOOLEAN NOT NULL DEFAULT TRUE,
   metadata JSONB DEFAULT '{}',
   created_by UUID,
   created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
   updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
   deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_policies_type ON policies(policy_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_policies_enabled ON policies(enabled) WHERE deleted_at IS NULL;
CREATE INDEX idx_policies_priority ON policies(priority DESC) WHERE deleted_at IS NULL;

-- Audit Logs 表（审计日志）
CREATE TABLE audit_logs (
   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
   trace_id VARCHAR(64),
   action audit_action NOT NULL,
   resource_type VARCHAR(100) NOT NULL,
   resource_id UUID,
   user_id UUID,
   user_ip VARCHAR(45),
   decision decision_type,
   policy_id UUID REFERENCES policies(id),
   request_data JSONB,
   response_data JSONB,
   error_message TEXT,
   metadata JSONB DEFAULT '{}',
   created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_trace_id ON audit_logs(trace_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_policy_id ON audit_logs(policy_id);

-- Compliance Reports 表（合规报告）
CREATE TABLE compliance_reports (
   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
   report_type VARCHAR(100) NOT NULL,
   framework VARCHAR(100) NOT NULL,
   period_start TIMESTAMP WITH TIME ZONE NOT NULL,
   period_end TIMESTAMP WITH TIME ZONE NOT NULL,
   findings JSONB NOT NULL,
   summary JSONB DEFAULT '{}',
   status VARCHAR(50) NOT NULL DEFAULT 'draft',
   generated_by UUID,
   created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_compliance_reports_type ON compliance_reports(report_type);
CREATE INDEX idx_compliance_reports_framework ON compliance_reports(framework);
CREATE INDEX idx_compliance_reports_period ON compliance_reports(period_start, period_end);
CREATE INDEX idx_compliance_reports_created_at ON compliance_reports(created_at DESC);

-- ============================================
-- 6. 缓存与性能表
-- ============================================

-- Inference Cache 表（推理缓存元数据）
CREATE TABLE inference_cache (
   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
   cache_key VARCHAR(64) NOT NULL UNIQUE,
   model_id UUID NOT NULL REFERENCES models(id),
   prompt_hash VARCHAR(64) NOT NULL,
   vector_id VARCHAR(255),
   response TEXT NOT NULL,
   metadata JSONB DEFAULT '{}',
   hit_count INTEGER NOT NULL DEFAULT 0,
   last_hit_at TIMESTAMP WITH TIME ZONE,
   expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
   created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inference_cache_key ON inference_cache(cache_key);
CREATE INDEX idx_inference_cache_model ON inference_cache(model_id);
CREATE INDEX idx_inference_cache_hash ON inference_cache(prompt_hash);
CREATE INDEX idx_inference_cache_vector ON inference_cache(vector_id);
CREATE INDEX idx_inference_cache_expires ON inference_cache(expires_at);

-- ============================================
-- 7. 系统配置表
-- ============================================

-- System Configs 表
CREATE TABLE system_configs (
   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
   config_key VARCHAR(255) NOT NULL UNIQUE,
   config_value JSONB NOT NULL,
   description TEXT,
   is_sensitive BOOLEAN NOT NULL DEFAULT FALSE,
   metadata JSONB DEFAULT '{}',
   created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
   updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_system_configs_key ON system_configs(config_key);

-- API Keys 表
CREATE TABLE api_keys (
   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
   key_hash VARCHAR(64) NOT NULL UNIQUE,
   name VARCHAR(255) NOT NULL,
   user_id UUID,
   scopes JSONB NOT NULL DEFAULT '[]',
   rate_limit INTEGER,
   expires_at TIMESTAMP WITH TIME ZONE,
   last_used_at TIMESTAMP WITH TIME ZONE,
   enabled BOOLEAN NOT NULL DEFAULT TRUE,
   metadata JSONB DEFAULT '{}',
   created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
   updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_enabled ON api_keys(enabled);

-- ============================================
-- 8. 触发器（自动更新 updated_at）
-- ============================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = NOW();
   RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_agents_updated_at BEFORE UPDATE ON agents
   FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workflows_updated_at BEFORE UPDATE ON workflows
   FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workflow_steps_updated_at BEFORE UPDATE ON workflow_steps
   FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_models_updated_at BEFORE UPDATE ON models
   FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_training_jobs_updated_at BEFORE UPDATE ON training_jobs
   FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_documents_updated_at BEFORE UPDATE ON documents
   FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_knowledge_graphs_updated_at BEFORE UPDATE ON knowledge_graphs
   FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_kg_entities_updated_at BEFORE UPDATE ON kg_entities
   FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_policies_updated_at BEFORE UPDATE ON policies
   FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_system_configs_updated_at BEFORE UPDATE ON system_configs
   FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_api_keys_updated_at BEFORE UPDATE ON api_keys
   FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- 9. 初始化数据
-- ============================================

-- 插入默认系统配置
INSERT INTO system_configs (config_key, config_value, description) VALUES
   ('inference.default_timeout', '30000', '默认推理超时时间（毫秒）'),
   ('inference.max_retries', '3', '推理最大重试次数'),
   ('cache.ttl.l1', '300', 'L1 缓存 TTL（秒）'),
   ('cache.ttl.l2', '3600', 'L2 缓存 TTL（秒）'),
   ('cache.ttl.l3', '86400', 'L3 缓存 TTL（秒）'),
   ('rag.chunk_size', '512', 'RAG 文档分块大小（tokens）'),
   ('rag.chunk_overlap', '50', 'RAG 文档分块重叠（tokens）'),
   ('training.max_concurrent_jobs', '5', '最大并发训练任务数');

-- Personal.AI order the ending

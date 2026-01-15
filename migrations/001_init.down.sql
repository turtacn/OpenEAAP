-- migrations/001_init.down.sql
-- OpenEAAP 数据库迁移回滚脚本
-- 用途：删除所有表，回滚到初始化之前的状态
-- 执行顺序：与 001_init.up.sql 相反，先删除有外键依赖的表

-- ============================================================================
-- 1. 删除审计和日志表
-- ============================================================================

-- 删除审计日志表
DROP TABLE IF EXISTS audit_logs CASCADE;

-- 删除操作日志表
DROP TABLE IF EXISTS operation_logs CASCADE;

-- ============================================================================
-- 2. 删除反馈和学习相关表
-- ============================================================================

-- 删除用户反馈表
DROP TABLE IF EXISTS user_feedbacks CASCADE;

-- 删除自动评估结果表
DROP TABLE IF EXISTS auto_evaluations CASCADE;

-- 删除训练任务表
DROP TABLE IF EXISTS training_jobs CASCADE;

-- 删除训练数据集表
DROP TABLE IF EXISTS training_datasets CASCADE;

-- ============================================================================
-- 3. 删除执行历史和缓存表
-- ============================================================================

-- 删除 Agent 执行历史表
DROP TABLE IF EXISTS agent_executions CASCADE;

-- 删除 Workflow 执行历史表
DROP TABLE IF EXISTS workflow_executions CASCADE;

-- 删除推理缓存表
DROP TABLE IF EXISTS inference_cache CASCADE;

-- 删除会话记忆表
DROP TABLE IF EXISTS conversation_memory CASCADE;

-- ============================================================================
-- 4. 删除知识库相关表
-- ============================================================================

-- 删除文档块表（包含向量数据）
DROP TABLE IF EXISTS document_chunks CASCADE;

-- 删除文档表
DROP TABLE IF EXISTS documents CASCADE;

-- 删除知识库表
DROP TABLE IF EXISTS knowledge_bases CASCADE;

-- ============================================================================
-- 5. 删除策略和权限表
-- ============================================================================

-- 删除策略规则表
DROP TABLE IF EXISTS policy_rules CASCADE;

-- 删除策略表
DROP TABLE IF EXISTS policies CASCADE;

-- 删除角色权限关联表
DROP TABLE IF EXISTS role_permissions CASCADE;

-- 删除权限表
DROP TABLE IF EXISTS permissions CASCADE;

-- 删除用户角色关联表
DROP TABLE IF EXISTS user_roles CASCADE;

-- 删除角色表
DROP TABLE IF EXISTS roles CASCADE;

-- ============================================================================
-- 6. 删除模型相关表
-- ============================================================================

-- 删除模型版本表
DROP TABLE IF EXISTS model_versions CASCADE;

-- 删除模型配置表
DROP TABLE IF EXISTS model_configs CASCADE;

-- 删除模型表
DROP TABLE IF EXISTS models CASCADE;

-- ============================================================================
-- 7. 删除 Workflow 相关表
-- ============================================================================

-- 删除 Workflow 步骤表
DROP TABLE IF EXISTS workflow_steps CASCADE;

-- 删除 Workflow 表
DROP TABLE IF EXISTS workflows CASCADE;

-- ============================================================================
-- 8. 删除 Agent 相关表
-- ============================================================================

-- 删除 Agent 工具关联表
DROP TABLE IF EXISTS agent_tools CASCADE;

-- 删除工具表
DROP TABLE IF EXISTS tools CASCADE;

-- 删除 Agent 表
DROP TABLE IF EXISTS agents CASCADE;

-- ============================================================================
-- 9. 删除用户和租户表
-- ============================================================================

-- 删除 API Key 表
DROP TABLE IF EXISTS api_keys CASCADE;

-- 删除用户表
DROP TABLE IF EXISTS users CASCADE;

-- 删除租户表
DROP TABLE IF EXISTS tenants CASCADE;

-- ============================================================================
-- 10. 删除枚举类型
-- ============================================================================

-- 删除训练类型枚举
DROP TYPE IF EXISTS training_type CASCADE;

-- 删除运行时类型枚举
DROP TYPE IF EXISTS runtime_type CASCADE;

-- 删除反馈类型枚举
DROP TYPE IF EXISTS feedback_type CASCADE;

-- 删除敏感度级别枚举
DROP TYPE IF EXISTS sensitivity_level CASCADE;

-- 删除执行状态枚举
DROP TYPE IF EXISTS execution_status CASCADE;

-- 删除任务状态枚举
DROP TYPE IF EXISTS job_status CASCADE;

-- 删除决策类型枚举
DROP TYPE IF EXISTS decision_type CASCADE;

-- 删除模型类型枚举
DROP TYPE IF EXISTS model_type CASCADE;

-- 删除 Agent 类型枚举
DROP TYPE IF EXISTS agent_type CASCADE;

-- 删除用户状态枚举
DROP TYPE IF EXISTS user_status CASCADE;

-- ============================================================================
-- 11. 删除扩展
-- ============================================================================

-- 删除 UUID 生成扩展
DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;

-- 删除向量搜索扩展（如果使用 pgvector）
DROP EXTENSION IF EXISTS vector CASCADE;

-- ============================================================================
-- 回滚完成确认
-- ============================================================================

DO $$
BEGIN
   RAISE NOTICE '========================================';
   RAISE NOTICE 'OpenEAAP 数据库迁移回滚完成';
   RAISE NOTICE '所有表、类型和扩展已删除';
   RAISE NOTICE '数据库已恢复到初始状态';
   RAISE NOTICE '========================================';
END $$;

-- Personal.AI order the ending


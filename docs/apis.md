# OpenEAAP API 文档

**版本**: v1.0
**最后更新**: 2026-01-13
**状态**: 设计阶段

---

## 目录

1. [概述](#1-概述)
2. [认证与鉴权](#2-认证与鉴权)
3. [通用约定](#3-通用约定)
4. [Agent API](#4-agent-api)
5. [Workflow API](#5-workflow-api)
6. [Model API](#6-model-api)
7. [Knowledge API](#7-knowledge-api)
8. [Training API](#8-training-api)
9. [Policy API](#9-policy-api)
10. [Audit API](#10-audit-api)
11. [错误码参考](#11-错误码参考)
12. [Webhooks](#12-webhooks)
13. [SDK 示例](#13-sdk-示例)

---

## 1. 概述

OpenEAAP 提供 **RESTful HTTP API** 和 **gRPC API** 两种接口形式，支持企业级 AI Agent 的全生命周期管理。

### 1.1 API 基础信息

| 项目           | 说明                            |
| ------------ | ----------------------------- |
| **Base URL** | `https://api.openeeap.com/v1` |
| **协议**       | HTTPS (TLS 1.2+)              |
| **认证方式**     | Bearer Token (JWT) / API Key  |
| **内容类型**     | `application/json`            |
| **字符编码**     | UTF-8                         |
| **请求限制**     | 1000 请求/分钟（可调整）               |

### 1.2 API 版本策略

* **当前版本**: v1
* **版本标识**: URL 路径中包含版本号 `/v1`
* **向后兼容**: 在大版本内保持向后兼容
* **废弃策略**: 废弃的 API 提前 6 个月通知，并提供迁移指南

### 1.3 接口风格对比

| 特性        | RESTful HTTP            | gRPC            |
| --------- | ----------------------- | --------------- |
| **适用场景**  | Web 应用、移动端              | 微服务间通信、高性能场景    |
| **性能**    | 中等                      | 高（Protobuf 序列化） |
| **流式支持**  | SSE（Server-Sent Events） | 原生支持双向流         |
| **浏览器支持** | 原生支持                    | 需要 gRPC-Web     |
| **调试便利性** | 高（cURL、Postman）         | 中等（需专用工具）       |

---

## 2. 认证与鉴权

### 2.1 认证方式

#### 2.1.1 Bearer Token（推荐）

```http
POST /v1/auth/login
Content-Type: application/json

{
  "username": "user@example.com",
  "password": "SecurePass123!"
}
```

**响应**:

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "dGVzdC1yZWZyZXNoLXRva2Vu..."
}
```

**使用方式**:

```http
GET /v1/agents
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

#### 2.1.2 API Key

```http
GET /v1/agents
X-API-Key: sk_live_1234567890abcdef
```

**获取 API Key**:

```http
POST /v1/api-keys
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "Production API Key",
  "scopes": ["agent:read", "agent:write"],
  "expires_at": "2027-01-01T00:00:00Z"
}
```

### 2.2 权限模型

OpenEAAP 使用 **RBAC (Role-Based Access Control)** 和 **ABAC (Attribute-Based Access Control)** 混合模型。

**预定义角色**:

| 角色            | 权限                              | 说明    |
| ------------- | ------------------------------- | ----- |
| **Admin**     | 全部权限                            | 系统管理员 |
| **Developer** | agent:*, workflow:*, model:read | 开发者   |
| **Operator**  | agent:read, workflow:execute    | 运维人员  |
| **Viewer**    | *:read                          | 只读用户  |

**权限格式**: `resource:action`，例如 `agent:create`、`workflow:execute`。

---

## 3. 通用约定

### 3.1 请求格式

**标准请求头**:

```http
POST /v1/agents
Authorization: Bearer {token}
Content-Type: application/json
Accept: application/json
X-Request-ID: 550e8400-e29b-41d4-a716-446655440000
```

**分页参数**:

```http
GET /v1/agents?page=1&page_size=20&sort_by=created_at&order=desc
```

| 参数          | 类型      | 默认值          | 说明             |
| ----------- | ------- | ------------ | -------------- |
| `page`      | integer | 1            | 页码（从 1 开始）     |
| `page_size` | integer | 20           | 每页数量（最大 100）   |
| `sort_by`   | string  | `created_at` | 排序字段           |
| `order`     | string  | `desc`       | 排序方向（asc/desc） |

### 3.2 响应格式

**成功响应**:

```json
{
  "success": true,
  "data": {
    "id": "agent_1234567890",
    "name": "Security Analyst Agent",
    "status": "active"
  },
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-13T10:30:00Z"
  }
}
```

**分页响应**:

```json
{
  "success": true,
  "data": [...],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total_pages": 5,
    "total_count": 95
  },
  "meta": {
    "request_id": "...",
    "timestamp": "..."
  }
}
```

**错误响应**:

```json
{
  "success": false,
  "error": {
    "code": "AGENT_NOT_FOUND",
    "message": "Agent with ID 'agent_123' not found",
    "details": {
      "agent_id": "agent_123"
    }
  },
  "meta": {
    "request_id": "...",
    "timestamp": "..."
  }
}
```

### 3.3 HTTP 状态码

| 状态码                           | 说明         | 使用场景             |
| ----------------------------- | ---------- | ---------------- |
| **200 OK**                    | 请求成功       | GET、PUT、PATCH 成功 |
| **201 Created**               | 资源创建成功     | POST 成功          |
| **204 No Content**            | 请求成功但无返回内容 | DELETE 成功        |
| **400 Bad Request**           | 请求参数错误     | 参数验证失败           |
| **401 Unauthorized**          | 未认证        | Token 无效或过期      |
| **403 Forbidden**             | 无权限        | 权限不足             |
| **404 Not Found**             | 资源不存在      | ID 不存在           |
| **409 Conflict**              | 资源冲突       | 名称重复、状态冲突        |
| **422 Unprocessable Entity**  | 语义错误       | 业务逻辑验证失败         |
| **429 Too Many Requests**     | 请求过多       | 超过限流阈值           |
| **500 Internal Server Error** | 服务器错误      | 内部异常             |
| **503 Service Unavailable**   | 服务不可用      | 维护中、过载           |

---

## 4. Agent API

### 4.1 创建 Agent

**端点**: `POST /v1/agents`

**请求示例**:

```json
{
  "name": "Security Analyst Agent",
  "description": "Analyzes security events and provides recommendations",
  "runtime_type": "native",
  "config": {
    "model": "gpt-4o",
    "temperature": 0.3,
    "max_tokens": 2000,
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "query_threat_intelligence",
          "description": "Query threat intelligence database",
          "parameters": {
            "type": "object",
            "properties": {
              "indicator": {"type": "string"},
              "indicator_type": {"type": "string", "enum": ["ip", "domain", "hash"]}
            },
            "required": ["indicator", "indicator_type"]
          }
        }
      }
    ],
    "memory": {
      "enabled": true,
      "max_messages": 50
    }
  },
  "metadata": {
    "department": "Security",
    "owner": "security-team@example.com"
  }
}
```

**响应**: `201 Created`

```json
{
  "success": true,
  "data": {
    "id": "agent_abc123def456",
    "name": "Security Analyst Agent",
    "runtime_type": "native",
    "status": "active",
    "created_at": "2026-01-13T10:30:00Z",
    "updated_at": "2026-01-13T10:30:00Z"
  }
}
```

### 4.2 列出 Agents

**端点**: `GET /v1/agents`

**查询参数**:

* `status`: 过滤状态（`active`、`inactive`、`error`）
* `runtime_type`: 过滤运行时类型
* `search`: 模糊搜索名称和描述

**请求示例**:

```http
GET /v1/agents?status=active&page=1&page_size=20
```

**响应**: `200 OK`

```json
{
  "success": true,
  "data": [
    {
      "id": "agent_abc123def456",
      "name": "Security Analyst Agent",
      "runtime_type": "native",
      "status": "active",
      "created_at": "2026-01-13T10:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total_pages": 1,
    "total_count": 5
  }
}
```

### 4.3 获取 Agent 详情

**端点**: `GET /v1/agents/{agent_id}`

**响应**: `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "agent_abc123def456",
    "name": "Security Analyst Agent",
    "description": "Analyzes security events...",
    "runtime_type": "native",
    "config": {...},
    "status": "active",
    "statistics": {
      "total_executions": 1523,
      "success_rate": 0.985,
      "avg_latency_ms": 1250
    },
    "created_at": "2026-01-13T10:30:00Z",
    "updated_at": "2026-01-13T10:30:00Z"
  }
}
```

### 4.4 更新 Agent

**端点**: `PATCH /v1/agents/{agent_id}`

**请求示例**:

```json
{
  "description": "Updated description",
  "config": {
    "temperature": 0.5
  }
}
```

**响应**: `200 OK`

### 4.5 删除 Agent

**端点**: `DELETE /v1/agents/{agent_id}`

**响应**: `204 No Content`

### 4.6 执行 Agent（同步）

**端点**: `POST /v1/agents/{agent_id}/execute`

**请求示例**:

```json
{
  "input": "Analyze the security event: IP 192.168.1.100 accessed admin panel",
  "context": {
    "user_id": "user_123",
    "session_id": "session_xyz"
  },
  "options": {
    "timeout_seconds": 60,
    "max_iterations": 5
  }
}
```

**响应**: `200 OK`

```json
{
  "success": true,
  "data": {
    "execution_id": "exec_789ghi012jkl",
    "agent_id": "agent_abc123def456",
    "output": "Analysis complete. IP 192.168.1.100 has been flagged...",
    "metadata": {
      "iterations": 3,
      "tools_called": ["query_threat_intelligence", "check_user_permissions"],
      "total_tokens": 1850,
      "latency_ms": 1230
    },
    "completed_at": "2026-01-13T10:31:15Z"
  }
}
```

### 4.7 执行 Agent（流式）

**端点**: `POST /v1/agents/{agent_id}/execute-stream`

**响应**: `200 OK` (Server-Sent Events)

```
data: {"type":"start","execution_id":"exec_789ghi012jkl"}

data: {"type":"token","content":"Analysis"}

data: {"type":"token","content":" complete."}

data: {"type":"tool_call","tool":"query_threat_intelligence","args":{...}}

data: {"type":"complete","output":"...","metadata":{...}}
```

---

## 5. Workflow API

### 5.1 创建 Workflow

**端点**: `POST /v1/workflows`

**请求示例**:

```json
{
  "name": "Incident Response Workflow",
  "description": "Automated incident response workflow",
  "trigger": {
    "type": "webhook",
    "config": {
      "url": "/webhooks/incident"
    }
  },
  "steps": [
    {
      "id": "step_1",
      "name": "Analyze Event",
      "type": "agent",
      "agent_id": "agent_abc123def456",
      "input": "{{ trigger.payload.event_description }}",
      "next": "step_2"
    },
    {
      "id": "step_2",
      "name": "Decision",
      "type": "condition",
      "condition": "{{ step_1.output.severity }} == 'high'",
      "on_true": "step_3",
      "on_false": "step_4"
    },
    {
      "id": "step_3",
      "name": "Escalate",
      "type": "action",
      "action": "send_notification",
      "config": {
        "channel": "pagerduty",
        "message": "High severity incident detected"
      }
    },
    {
      "id": "step_4",
      "name": "Log Event",
      "type": "action",
      "action": "log_to_siem"
    }
  ],
  "error_handling": {
    "retry": {
      "max_attempts": 3,
      "backoff": "exponential"
    }
  }
}
```

**响应**: `201 Created`

### 5.2 执行 Workflow

**端点**: `POST /v1/workflows/{workflow_id}/execute`

**请求示例**:

```json
{
  "input": {
    "event_description": "Suspicious login attempt detected",
    "source_ip": "192.168.1.100"
  }
}
```

**响应**: `200 OK`

```json
{
  "success": true,
  "data": {
    "execution_id": "wf_exec_abc123",
    "workflow_id": "workflow_xyz789",
    "status": "running",
    "started_at": "2026-01-13T10:30:00Z"
  }
}
```

### 5.3 获取 Workflow 执行状态

**端点**: `GET /v1/workflows/executions/{execution_id}`

**响应**: `200 OK`

```json
{
  "success": true,
  "data": {
    "execution_id": "wf_exec_abc123",
    "workflow_id": "workflow_xyz789",
    "status": "completed",
    "steps": [
      {
        "id": "step_1",
        "name": "Analyze Event",
        "status": "completed",
        "output": {...},
        "started_at": "2026-01-13T10:30:01Z",
        "completed_at": "2026-01-13T10:30:15Z"
      }
    ],
    "started_at": "2026-01-13T10:30:00Z",
    "completed_at": "2026-01-13T10:30:45Z"
  }
}
```

### 5.4 暂停/恢复 Workflow

**端点**: `POST /v1/workflows/executions/{execution_id}/pause`

**端点**: `POST /v1/workflows/executions/{execution_id}/resume`

**响应**: `200 OK`

---

## 6. Model API

### 6.1 列出模型

**端点**: `GET /v1/models`

**响应**: `200 OK`

```json
{
  "success": true,
  "data": [
    {
      "id": "model_gpt4o",
      "name": "GPT-4o",
      "type": "chat",
      "provider": "openai",
      "status": "available",
      "pricing": {
        "input_per_1k": 0.005,
        "output_per_1k": 0.015
      },
      "capabilities": ["function_calling", "vision"]
    }
  ]
}
```

### 6.2 注册自定义模型

**端点**: `POST /v1/models`

**请求示例**:

```json
{
  "name": "Custom LLM",
  "type": "chat",
  "endpoint": "https://custom-llm.example.com/v1/chat/completions",
  "authentication": {
    "type": "bearer",
    "token": "sk_custom_..."
  },
  "config": {
    "timeout_seconds": 30,
    "max_retries": 3
  }
}
```

**响应**: `201 Created`

### 6.3 测试模型

**端点**: `POST /v1/models/{model_id}/test`

**请求示例**:

```json
{
  "messages": [
    {"role": "user", "content": "Hello, how are you?"}
  ]
}
```

**响应**: `200 OK`

```json
{
  "success": true,
  "data": {
    "response": "I'm doing well, thank you!",
    "latency_ms": 850,
    "tokens": {
      "input": 7,
      "output": 8,
      "total": 15
    }
  }
}
```

---

## 7. Knowledge API

### 7.1 上传文档

**端点**: `POST /v1/knowledge/documents`

**请求**: Multipart Form Data

```http
POST /v1/knowledge/documents
Content-Type: multipart/form-data

------boundary
Content-Disposition: form-data; name="file"; filename="report.pdf"
Content-Type: application/pdf

[Binary PDF content]
------boundary
Content-Disposition: form-data; name="metadata"

{"collection":"security_reports","tags":["incident","2026-Q1"]}
------boundary--
```

**响应**: `201 Created`

```json
{
  "success": true,
  "data": {
    "document_id": "doc_abc123",
    "filename": "report.pdf",
    "size_bytes": 204800,
    "status": "processing",
    "estimated_completion": "2026-01-13T10:35:00Z"
  }
}
```

### 7.2 查询文档

**端点**: `POST /v1/knowledge/search`

**请求示例**:

```json
{
  "query": "What are the recommended mitigation steps for ransomware?",
  "collection": "security_reports",
  "top_k": 5,
  "filters": {
    "tags": ["incident_response"]
  },
  "rerank": true
}
```

**响应**: `200 OK`

```json
{
  "success": true,
  "data": {
    "results": [
      {
        "document_id": "doc_abc123",
        "chunk_id": "chunk_xyz789",
        "content": "Recommended mitigation steps for ransomware include...",
        "score": 0.92,
        "metadata": {
          "page": 5,
          "source": "NIST Cybersecurity Framework"
        }
      }
    ],
    "query_time_ms": 45
  }
}
```

---

## 8. Training API

### 8.1 创建训练任务

**端点**: `POST /v1/training/jobs`

**请求示例**:

```json
{
  "name": "Fine-tune Security Model",
  "base_model": "model_gpt4o",
  "training_type": "rlhf",
  "dataset": {
    "type": "feedback",
    "source": "feedback_dataset_123",
    "filters": {
      "rating": {"gte": 4}
    }
  },
  "hyperparameters": {
    "learning_rate": 1e-5,
    "batch_size": 32,
    "epochs": 3
  },
  "resources": {
    "gpu_type": "A100",
    "gpu_count": 4
  }
}
```

**响应**: `201 Created`

```json
{
  "success": true,
  "data": {
    "job_id": "train_job_abc123",
    "status": "queued",
    "estimated_start": "2026-01-13T11:00:00Z",
    "estimated_duration_minutes": 120
  }
}
```

### 8.2 获取训练状态

**端点**: `GET /v1/training/jobs/{job_id}`

**响应**: `200 OK`

```json
{
  "success": true,
  "data": {
    "job_id": "train_job_abc123",
    "status": "running",
    "progress": 0.45,
    "metrics": {
      "current_epoch": 2,
      "train_loss": 0.32,
      "eval_loss": 0.38
    },
    "started_at": "2026-01-13T11:00:00Z",
    "estimated_completion": "2026-01-13T13:00:00Z"
  }
}
```

---

## 9. Policy API

### 9.1 创建策略

**端点**: `POST /v1/policies`

**请求示例**:

```json
{
  "name": "Sensitive Data Access Policy",
  "type": "data_governance",
  "rules": [
    {
      "effect": "deny",
      "conditions": {
        "data.sensitivity_level": "high",
        "user.role": {"not_in": ["admin", "security_analyst"]}
      },
      "actions": ["read", "query"]
    }
  ],
  "enabled": true
}
```

**响应**: `201 Created`

### 9.2 评估策略

**端点**: `POST /v1/policies/evaluate`

**请求示例**:

```json
{
  "context": {
    "user": {
      "id": "user_123",
      "role": "developer"
    },
    "data": {
      "sensitivity_level": "high"
    },
    "action": "read"
  }
}
```

**响应**: `200 OK`

```json
{
  "success": true,
  "data": {
    "decision": "deny",
    "matched_policies": ["policy_abc123"],
    "reason": "User role 'developer' not authorized for high sensitivity data"
  }
}
```

---

## 10. Audit API

### 10.1 查询审计日志

**端点**: `GET /v1/audit/logs`

**查询参数**:

* `user_id`: 过滤用户
* `action`: 过滤操作类型
* `resource_type`: 过滤资源类型
* `start_time`: 开始时间（ISO 8601）
* `end_time`: 结束时间

**请求示例**:

```http
GET /v1/audit/logs?user_id=user_123&action=agent:execute&start_time=2026-01-13T00:00:00Z
```

**响应**: `200 OK`

```json
{
  "success": true,
  "data": [
    {
      "id": "audit_log_xyz789",
      "timestamp": "2026-01-13T10:30:00Z",
      "user_id": "user_123",
      "action": "agent:execute",
      "resource_type": "agent",
      "resource_id": "agent_abc123",
      "result": "success",
      "metadata": {
        "ip_address": "192.168.1.100",
        "user_agent": "OpenEAAP-SDK/1.0.0"
      }
    }
  ],
  "pagination": {...}
}
```

### 10.2 导出审计报告

**端点**: `POST /v1/audit/reports`

**请求示例**:

```json
{
  "format": "csv",
  "filters": {
    "start_time": "2026-01-01T00:00:00Z",
    "end_time": "2026-01-13T23:59:59Z"
  },
  "delivery": {
    "method": "email",
    "recipients": ["compliance@example.com"]
  }
}
```

**响应**: `202 Accepted`

```json
{
  "success": true,
  "data": {
    "report_id": "report_abc123",
    "status": "generating",
    "estimated_completion": "2026-01-13T11:00:00Z"
  }
}
```

---

## 11. 错误码参考

### 11.1 错误码列表

| 错误码                   | HTTP 状态码 | 说明             |
| --------------------- | -------- | -------------- |
| `INVALID_REQUEST`     | 400      | 请求格式错误         |
| `INVALID_PARAMETER`   | 400      | 参数验证失败         |
| `UNAUTHORIZED`        | 401      | 未认证或 Token 无效  |
| `FORBIDDEN`           | 403      | 权限不足           |
| `AGENT_NOT_FOUND`     | 404      | Agent 不存在      |
| `WORKFLOW_NOT_FOUND`  | 404      | Workflow 不存在   |
| `MODEL_NOT_FOUND`     | 404      | 模型不存在          |
| `AGENT_NAME_CONFLICT` | 409      | Agent 名称已存在    |
| `INVALID_AGENT_STATE` | 422      | Agent 状态不允许该操作 |
| `EXECUTION_TIMEOUT`   | 422      | 执行超时           |
| `RATE_LIMIT_EXCEEDED` | 429      | 超过请求限制         |
| `INTERNAL_ERROR`      | 500      | 服务器内部错误        |
| `MODEL_UNAVAILABLE`   | 503      | 模型服务不可用        |

### 11.2 错误响应示例

```json
{
  "success": false,
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "Validation failed for field 'temperature'",
    "details": {
      "field": "temperature",
      "constraint": "Must be between 0 and 2",
      "provided_value": 3.5
    }
  },
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-13T10:30:00Z",
    "documentation_url": "https://docs.openeeap.com/errors/INVALID_PARAMETER"
  }
}
```

---

## 12. Webhooks

### 12.1 Webhook 事件类型

OpenEAAP 支持以下 Webhook 事件：

| 事件类型                           | 说明          | 触发时机          |
| ------------------------------ | ----------- | ------------- |
| `agent.created`                | Agent 创建    | Agent 创建成功后   |
| `agent.updated`                | Agent 更新    | Agent 配置更新后   |
| `agent.deleted`                | Agent 删除    | Agent 删除后     |
| `agent.execution.started`      | 执行开始        | Agent 开始执行    |
| `agent.execution.completed`    | 执行完成        | Agent 执行成功完成  |
| `agent.execution.failed`       | 执行失败        | Agent 执行失败    |
| `workflow.execution.started`   | Workflow 开始 | Workflow 开始执行 |
| `workflow.execution.completed` | Workflow 完成 | Workflow 完成   |
| `training.job.started`         | 训练开始        | 训练任务开始        |
| `training.job.completed`       | 训练完成        | 训练任务完成        |

### 12.2 配置 Webhook

**端点**: `POST /v1/webhooks`

**请求示例**:

```json
{
  "url": "https://example.com/webhooks/openeeap",
  "events": ["agent.execution.completed", "agent.execution.failed"],
  "secret": "whsec_1234567890abcdef",
  "enabled": true
}
```

**响应**: `201 Created`

### 12.3 Webhook 请求格式

```http
POST /webhooks/openeeap
Content-Type: application/json
X-OpenEAAP-Signature: sha256=abc123def456...
X-OpenEAAP-Event: agent.execution.completed
X-OpenEAAP-Delivery: uuid-1234-5678-9012

{
  "event": "agent.execution.completed",
  "timestamp": "2026-01-13T10:31:15Z",
  "data": {
    "execution_id": "exec_789ghi012jkl",
    "agent_id": "agent_abc123def456",
    "status": "completed",
    "output": "Analysis complete. IP 192.168.1.100 has been flagged as suspicious.",
    "metadata": {
      "iterations": 3,
      "total_tokens": 1850,
      "latency_ms": 1230
    }
  }
}
```

### 12.4 验证 Webhook 签名

为确保 Webhook 请求的真实性，OpenEAAP 使用 HMAC-SHA256 签名。

**验证步骤**（Python 示例）:

```python
import hmac
import hashlib

def verify_webhook_signature(payload, signature, secret):
    """验证 Webhook 签名"""
    expected_signature = hmac.new(
        secret.encode('utf-8'),
        payload.encode('utf-8'),
        hashlib.sha256
    ).hexdigest()
    
    return hmac.compare_digest(
        f"sha256={expected_signature}",
        signature
    )

# 使用示例
payload = request.body.decode('utf-8')
signature = request.headers['X-OpenEAAP-Signature']
secret = "whsec_1234567890abcdef"

if verify_webhook_signature(payload, signature, secret):
    # 签名验证通过，处理事件
    process_webhook_event(payload)
else:
    # 签名验证失败，拒绝请求
    return 401
```

### 12.5 重试机制

如果 Webhook 推送失败（HTTP 状态码非 2xx 或超时），OpenEAAP 会自动重试：

| 重试次数 | 延迟时间  | 说明   |
| ---- | ----- | ---- |
| 1    | 1 分钟  | 立即重试 |
| 2    | 5 分钟  | 短延迟  |
| 3    | 30 分钟 | 中延迟  |
| 4    | 2 小时  | 长延迟  |
| 5    | 12 小时 | 最后尝试 |

超过 5 次失败后，Webhook 会被标记为失败，需要手动重新触发或检查配置。

---

## 13. SDK 示例

### 13.1 Python SDK

#### 安装

```bash
pip install openeeap-sdk
```

#### 初始化客户端

```python
from openeeap import Client

# 使用 API Key 认证
client = Client(api_key="sk_live_1234567890abcdef")

# 或使用 Bearer Token 认证
client = Client(access_token="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...")
```

#### 创建和执行 Agent

```python
# 创建 Agent
agent = client.agents.create(
    name="Security Analyst Agent",
    description="Analyzes security events",
    runtime_type="native",
    config={
        "model": "gpt-4o",
        "temperature": 0.3,
        "tools": [
            {
                "type": "function",
                "function": {
                    "name": "query_threat_intelligence",
                    "description": "Query threat intelligence database",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "indicator": {"type": "string"}
                        }
                    }
                }
            }
        ]
    }
)

print(f"Created Agent: {agent.id}")

# 执行 Agent（同步）
result = client.agents.execute(
    agent_id=agent.id,
    input="Analyze suspicious IP 192.168.1.100"
)

print(f"Output: {result.output}")
print(f"Latency: {result.metadata.latency_ms}ms")

# 执行 Agent（流式）
for chunk in client.agents.execute_stream(
    agent_id=agent.id,
    input="Analyze suspicious IP 192.168.1.100"
):
    if chunk.type == "token":
        print(chunk.content, end="", flush=True)
    elif chunk.type == "tool_call":
        print(f"\n[Tool Called: {chunk.tool}]")
```

#### Workflow 操作

```python
# 创建 Workflow
workflow = client.workflows.create(
    name="Incident Response Workflow",
    steps=[
        {
            "id": "step_1",
            "type": "agent",
            "agent_id": agent.id,
            "input": "{{ trigger.payload.event }}"
        }
    ]
)

# 执行 Workflow
execution = client.workflows.execute(
    workflow_id=workflow.id,
    input={"event": "Suspicious login detected"}
)

# 等待完成
execution.wait_until_complete(timeout=300)

print(f"Status: {execution.status}")
print(f"Output: {execution.output}")
```

#### 知识库操作

```python
# 上传文档
with open("security_report.pdf", "rb") as f:
    document = client.knowledge.upload_document(
        file=f,
        collection="security_reports",
        metadata={"tags": ["incident", "2026-Q1"]}
    )

# 等待处理完成
document.wait_until_ready()

# 查询知识库
results = client.knowledge.search(
    query="What are ransomware mitigation steps?",
    collection="security_reports",
    top_k=5
)

for result in results:
    print(f"Score: {result.score:.2f}")
    print(f"Content: {result.content[:200]}...")
```

### 13.2 JavaScript/TypeScript SDK

#### 安装

```bash
npm install @openeeap/sdk
```

#### 初始化客户端

```typescript
import { OpenEAAPClient } from '@openeeap/sdk';

const client = new OpenEAAPClient({
  apiKey: 'sk_live_1234567890abcdef'
});
```

#### 创建和执行 Agent

```typescript
// 创建 Agent
const agent = await client.agents.create({
  name: 'Security Analyst Agent',
  runtimeType: 'native',
  config: {
    model: 'gpt-4o',
    temperature: 0.3
  }
});

// 执行 Agent（同步）
const result = await client.agents.execute({
  agentId: agent.id,
  input: 'Analyze suspicious IP 192.168.1.100'
});

console.log('Output:', result.output);

// 执行 Agent（流式）
const stream = await client.agents.executeStream({
  agentId: agent.id,
  input: 'Analyze suspicious IP 192.168.1.100'
});

for await (const chunk of stream) {
  if (chunk.type === 'token') {
    process.stdout.write(chunk.content);
  } else if (chunk.type === 'complete') {
    console.log('\nCompleted:', chunk.metadata);
  }
}
```

#### 错误处理

```typescript
import { OpenEAAPError, RateLimitError, NotFoundError } from '@openeeap/sdk';

try {
  const agent = await client.agents.get('invalid_id');
} catch (error) {
  if (error instanceof NotFoundError) {
    console.error('Agent not found');
  } else if (error instanceof RateLimitError) {
    console.error('Rate limit exceeded, retry after:', error.retryAfter);
  } else if (error instanceof OpenEAAPError) {
    console.error('API Error:', error.code, error.message);
  } else {
    throw error;
  }
}
```

### 13.3 Go SDK

#### 安装

```bash
go get github.com/openeeap/openeeap-go
```

#### 初始化客户端

```go
package main

import (
    "context"
    "fmt"
    "github.com/openeeap/openeeap-go"
)

func main() {
    client := openeeap.NewClient("sk_live_1234567890abcdef")
    
    // 创建 Agent
    agent, err := client.Agents.Create(context.Background(), &openeeap.CreateAgentRequest{
        Name:        "Security Analyst Agent",
        RuntimeType: openeeap.RuntimeTypeNative,
        Config: openeeap.AgentConfig{
            Model:       "gpt-4o",
            Temperature: 0.3,
        },
    })
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Created Agent: %s\n", agent.ID)
    
    // 执行 Agent
    result, err := client.Agents.Execute(context.Background(), agent.ID, &openeeap.ExecuteRequest{
        Input: "Analyze suspicious IP 192.168.1.100",
    })
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Output: %s\n", result.Output)
}
```

#### 流式执行

```go
stream, err := client.Agents.ExecuteStream(context.Background(), agent.ID, &openeeap.ExecuteRequest{
    Input: "Analyze suspicious IP 192.168.1.100",
})
if err != nil {
    panic(err)
}
defer stream.Close()

for {
    chunk, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        panic(err)
    }
    
    switch chunk.Type {
    case openeeap.ChunkTypeToken:
        fmt.Print(chunk.Content)
    case openeeap.ChunkTypeToolCall:
        fmt.Printf("\n[Tool: %s]\n", chunk.Tool)
    case openeeap.ChunkTypeComplete:
        fmt.Printf("\nCompleted in %dms\n", chunk.Metadata.LatencyMs)
    }
}
```

---

## 14. 最佳实践

### 14.1 性能优化

#### 使用缓存

```python
# 启用语义缓存以减少重复推理
result = client.agents.execute(
    agent_id=agent.id,
    input="Analyze IP 192.168.1.100",
    options={
        "enable_cache": True,
        "cache_ttl": 3600  # 缓存 1 小时
    }
)
```

#### 批量操作

```python
# 批量执行以提高吞吐量
inputs = [
    "Analyze IP 192.168.1.100",
    "Analyze IP 192.168.1.101",
    "Analyze IP 192.168.1.102"
]

results = client.agents.execute_batch(
    agent_id=agent.id,
    inputs=inputs,
    batch_size=10  # 并发数
)
```

### 14.2 错误处理

#### 重试机制

```python
from openeeap.exceptions import RateLimitError, ServiceUnavailableError
import time

def execute_with_retry(client, agent_id, input_text, max_retries=3):
    for attempt in range(max_retries):
        try:
            return client.agents.execute(agent_id=agent_id, input=input_text)
        except RateLimitError as e:
            if attempt < max_retries - 1:
                wait_time = e.retry_after or (2 ** attempt)
                time.sleep(wait_time)
            else:
                raise
        except ServiceUnavailableError:
            if attempt < max_retries - 1:
                time.sleep(2 ** attempt)
            else:
                raise
```

### 14.3 安全建议

1. **保护 API Key**：永远不要在代码中硬编码 API Key，使用环境变量或密钥管理服务
2. **使用 HTTPS**：始终通过 HTTPS 调用 API
3. **验证 Webhook 签名**：始终验证 Webhook 请求的签名
4. **限制权限**：为不同用途创建不同的 API Key，并赋予最小权限
5. **监控使用量**：定期检查 API 使用量和异常活动

---

## 15. 版本历史

### v1.0.0（2026-01-13）

**初始版本发布**

* ✅ Agent 管理 API（创建、查询、更新、删除、执行）
* ✅ Workflow 管理 API
* ✅ Model 管理 API
* ✅ Knowledge 管理 API
* ✅ Training API
* ✅ Policy API
* ✅ Audit API
* ✅ Webhook 支持
* ✅ Python、JavaScript、Go SDK

**已知限制**：

* 流式执行暂不支持工具调用中断
* Workflow 并发步骤数限制为 10

---

## 16. 支持与反馈

### 16.1 文档资源

* **官方文档**: [https://docs.openeeap.com](https://docs.openeeap.com)
* **API 参考**: [https://docs.openeeap.com/api](https://docs.openeeap.com/api)
* **示例代码**: [https://github.com/openeeap/examples](https://github.com/openeeap/examples)
* **变更日志**: [https://docs.openeeap.com/changelog](https://docs.openeeap.com/changelog)


---

## 17. 附录

### 17.1 Rate Limiting 详情

OpenEAAP 使用 **令牌桶算法** 实现限流，默认配额如下：

| 层级             | 请求限制    | Tokens 限制 | 并发限制  |
| -------------- | ------- | --------- | ----- |
| **Free**       | 100/分钟  | 100K/月    | 5 并发  |
| **Pro**        | 1000/分钟 | 1M/月      | 20 并发 |
| **Enterprise** | 自定义     | 自定义       | 自定义   |

**响应头**:

```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 995
X-RateLimit-Reset: 1705147200
```

### 17.2 数据保留策略

| 数据类型      | 保留期限 | 说明          |
| --------- | ---- | ----------- |
| **执行日志**  | 30 天 | 可配置延长至 90 天 |
| **审计日志**  | 1 年  | 合规要求        |
| **训练数据**  | 永久   | 除非手动删除      |
| **知识库文档** | 永久   | 除非手动删除      |

### 17.3 区域可用性

OpenEAAP 在以下区域提供服务：

| 区域       | Endpoint                                  | 延迟     |
| -------- | ----------------------------------------- | ------ |
| **美国东部** | `https://us-east-1.api.openeeap.com`      | < 50ms |
| **欧洲西部** | `https://eu-west-1.api.openeeap.com`      | < 60ms |
| **亚太地区** | `https://ap-southeast-1.api.openeeap.com` | < 80ms |

---

**文档版本**: v1.0
**最后更新**: 2026-01-13
**维护者**: OpenEAAP API 团队

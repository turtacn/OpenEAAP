module github.com/openeeap/openeeap

go 1.21

require (
   // Web Framework
   github.com/gin-gonic/gin v1.9.1
   github.com/gin-contrib/cors v1.5.0
   github.com/gin-contrib/zap v0.2.0

   // gRPC
   google.golang.org/grpc v1.60.1
   google.golang.org/protobuf v1.32.0

   // Database
   gorm.io/gorm v1.25.5
   gorm.io/driver/postgres v1.5.4
   github.com/jackc/pgx/v5 v5.5.1

   // Cache
   github.com/redis/go-redis/v9 v9.3.1
   github.com/allegro/bigcache/v3 v3.1.0

   // Vector Database
   github.com/milvus-io/milvus-sdk-go/v2 v2.3.4

   // Object Storage
   github.com/minio/minio-go/v7 v7.0.66

   // Message Queue
   github.com/segmentio/kafka-go v0.4.47
   github.com/IBM/sarama v1.42.1

   // Configuration
   github.com/spf13/viper v1.18.2
   github.com/spf13/cobra v1.8.0

   // Observability
   go.opentelemetry.io/otel v1.21.0
   go.opentelemetry.io/otel/trace v1.21.0
   go.opentelemetry.io/otel/metric v1.21.0
   go.opentelemetry.io/otel/exporters/jaeger v1.17.0
   go.opentelemetry.io/otel/exporters/prometheus v0.44.0
   github.com/prometheus/client_golang v1.18.0
   go.uber.org/zap v1.26.0

   // AI/ML Libraries
   github.com/tmc/langchaingo v0.1.5
   github.com/sashabaranov/go-openai v1.17.9

   // Utilities
   github.com/google/uuid v1.5.0
   github.com/golang-jwt/jwt/v5 v5.2.0
   golang.org/x/crypto v0.17.0
   golang.org/x/time v0.5.0
   github.com/go-playground/validator/v10 v10.16.0

   // Testing
   github.com/stretchr/testify v1.8.4
   github.com/testcontainers/testcontainers-go v0.27.0
   github.com/golang/mock v1.6.0

   // Database Migration
   github.com/golang-migrate/migrate/v4 v4.17.0

   // HTTP Client
   github.com/go-resty/resty/v2 v2.11.0

   // Data Processing
   github.com/tidwall/gjson v1.17.0
   github.com/goccy/go-json v0.10.2

   // Privacy & Security
   github.com/dlclark/regexp2 v1.10.0

   // Rate Limiting
   github.com/ulule/limiter/v3 v3.11.2
   golang.org/x/sync v0.6.0

   // Metrics & Stats
   github.com/montanaflynn/stats v0.7.1

   // Graph Database (for Knowledge Graph)
   github.com/neo4j/neo4j-go-driver/v5 v5.15.0

   // Vector Operations
   github.com/viterin/vek v0.4.2
   gonum.org/v1/gonum v0.14.0

   // NLP Utilities
   github.com/ikawaha/kagome/v2 v2.9.3

   // Distributed Computing
   github.com/hashicorp/consul/api v1.27.0
   github.com/hashicorp/vault/api v1.11.0

   // Workflow Engine
   github.com/google/cel-go v0.18.2

   // Plugin System
   github.com/hashicorp/go-plugin v1.6.0

   // Encryption
   github.com/gtank/cryptopasta v0.0.0-20170601214702-1f550f6f2f69

   // Context & Error Handling
   github.com/pkg/errors v0.9.1
   golang.org/x/exp v0.0.0-20231226003508-02704c960a9b
)

require (
   // Indirect dependencies
   github.com/bytedance/sonic v1.10.2 // indirect
   github.com/cespare/xxhash/v2 v2.2.0 // indirect
   github.com/chenzhuoyu/base64x v0.0.0-20230717121745-296ad89f973d // indirect
   github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
   github.com/gabriel-vasile/mimetype v1.4.3 // indirect
   github.com/gin-contrib/sse v0.1.0 // indirect
   github.com/go-playground/locales v0.14.1 // indirect
   github.com/go-playground/universal-translator v0.18.1 // indirect
   github.com/golang/protobuf v1.5.3 // indirect
   github.com/grpc-ecosystem/grpc-gateway/v2 v2.18.1 // indirect
   github.com/json-iterator/go v1.1.12 // indirect
   github.com/klauspost/cpuid/v2 v2.2.6 // indirect
   github.com/leodido/go-urn v1.2.4 // indirect
   github.com/mattn/go-isatty v0.0.20 // indirect
   github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
   github.com/modern-go/reflect2 v1.0.2 // indirect
   github.com/pelletier/go-toml/v2 v2.1.1 // indirect
   github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
   github.com/ugorji/go/codec v1.2.12 // indirect
   go.uber.org/multierr v1.11.0 // indirect
   golang.org/x/arch v0.6.0 // indirect
   golang.org/x/net v0.19.0 // indirect
   golang.org/x/sys v0.15.0 // indirect
   golang.org/x/text v0.14.0 // indirect
   google.golang.org/genproto/googleapis/api v0.0.0-20231212172506-995d672761c0 // indirect
   google.golang.org/genproto/googleapis/rpc v0.0.0-20231212172506-995d672761c0 // indirect
   gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Replace directives for local development or specific versions
replace (
   // Use local version of plugin runtime during development
   // github.com/openeeap/plugin-runtime => ./pkg/plugin-runtime

   // Pin specific vulnerable dependency versions
   golang.org/x/crypto => golang.org/x/crypto v0.17.0
   golang.org/x/net => golang.org/x/net v0.19.0
)

// Exclude vulnerable or problematic versions
exclude (
   github.com/golang/protobuf v1.3.0
   golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2
)

// Personal.AI order the ending


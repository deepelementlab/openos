# Agent OS gRPC API

This directory contains the Protocol Buffers definitions for the Agent OS gRPC API.

## Structure

```
api/proto/
└── v1/
    ├── common.proto      # Common types and messages
    ├── agent.proto       # Agent lifecycle management
    ├── tenant.proto      # Multi-tenancy management
    ├── runtime.proto     # Container runtime operations
    └── monitoring.proto  # Monitoring and observability
```

## Services

### AgentService

Manages AI agent lifecycle:
- `CreateAgent` - Create a new agent
- `GetAgent` - Get agent details
- `ListAgents` - List agents with filtering
- `UpdateAgent` - Update agent configuration
- `DeleteAgent` - Delete an agent
- `StartAgent` / `StopAgent` / `RestartAgent` - Lifecycle operations
- `GetAgentLogs` / `StreamAgentLogs` - Log streaming
- `GetAgentMetrics` / `StreamAgentMetrics` - Metrics streaming
- `ExecuteCommand` - Execute commands in running agents

### TenantService

Manages multi-tenancy:
- `CreateTenant` / `DeleteTenant` - Tenant management
- `GetTenant` / `ListTenants` - Tenant queries
- `GetTenantQuota` / `UpdateTenantQuota` - Quota management
- `GetTenantUsage` - Resource usage tracking
- `AddTenantMember` / `RemoveTenantMember` - Member management
- `SuspendTenant` / `ActivateTenant` - Tenant lifecycle

### RuntimeService

Manages container runtimes:
- `ListRuntimes` / `GetRuntime` - Runtime information
- `ListImages` / `GetImage` / `PullImage` / `RemoveImage` - Image management
- `ListNodes` / `GetNode` - Node management
- `DrainNode` / `CordonNode` / `UncordonNode` - Node operations

### MonitoringService

Provides observability:
- `HealthCheck` - System health status
- `GetSystemMetrics` / `StreamSystemMetrics` - System metrics
- `GetServiceStatus` - Service status
- `GetAlerts` / `AcknowledgeAlert` - Alert management
- `StreamEvents` - Real-time event streaming
- `GetAuditLogs` - Audit trail
- `GetPerformanceMetrics` - Performance metrics

## Code Generation

To generate Go code from proto files, run:

```bash
# Install protoc plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate code
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       api/proto/v1/*.proto
```

Or use buf for modern protobuf management:

```bash
# Install buf
go install github.com/bufbuild/buf/cmd/buf@latest

# Generate
buf generate
```

## Multi-Tenancy

All services support multi-tenancy through:

1. **Tenant Context**: Extracted from `X-Tenant-ID` header or JWT claims
2. **Resource Isolation**: Tenant ID filtering on all queries
3. **Quota Enforcement**: Automatic quota checks on resource creation
4. **Audit Logging**: All tenant operations are logged

## Authentication

gRPC services support:

1. **JWT Tokens**: Via `Authorization: Bearer <token>` metadata
2. **API Keys**: Via `X-API-Key` header (optional)
3. **mTLS**: TLS client certificate authentication (optional)

## Interceptors

The gRPC server includes the following interceptors:

1. **Recovery**: Recovers from panics
2. **Logging**: Logs all requests
3. **Authentication**: Validates JWT tokens
4. **Tenant**: Extracts and validates tenant context
5. **Quota**: Enforces resource quotas
6. **Metrics**: Prometheus metrics collection

## gRPC-Gateway (REST Bridge)

The API can be accessed via REST through gRPC-Gateway:

```go
gateway, err := gateway.NewGateway(gateway.Config{
    GRPCAddress: "localhost:9090",
    HTTPAddress: ":8080",
}, logger)
```

This provides REST endpoints that proxy to gRPC services.

## Example Usage

### Go Client

```go
conn, err := grpc.Dial("localhost:9090", grpc.WithInsecure())
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewAgentServiceClient(conn)

ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
    "authorization", "Bearer "+token,
    "x-tenant-id", tenantID,
))

agent, err := client.CreateAgent(ctx, &pb.CreateAgentRequest{
    Name:  "my-agent",
    Image: "agentos/base:latest",
})
```

### curl via gRPC-Gateway

```bash
# Create agent
curl -X POST http://localhost:8080/v1/agents \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"name": "my-agent", "image": "agentos/base:latest"}'

# List agents
curl http://localhost:8080/v1/agents \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

## Streaming

The API supports server-side streaming for:
- Agent logs (`StreamAgentLogs`)
- Agent metrics (`StreamAgentMetrics`)
- System events (`StreamEvents`)
- System metrics (`StreamSystemMetrics`)

Clients should handle streaming responses appropriately for their language.

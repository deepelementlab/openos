# OpenAPI / Swagger

Generate REST documentation from protobuf using `protoc` + `grpc-gateway` plugins:

```bash
protoc -I. --openapi_out=docs/openapi api/proto/*.proto
```

The HTTP surface is bridged via `grpc-gateway`; keep this directory for generated `swagger.json` in CI.

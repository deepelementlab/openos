# AOS SDKs

| Language   | Path        | Notes |
|------------|-------------|-------|
| Go         | `sdk/go`    | Use module `github.com/agentos/aos/sdk/go` — thin wrapper over gRPC stubs |
| Python     | `sdk/python`| Generate with `grpcio-tools` from `api/proto` |
| TypeScript | `sdk/ts`    | Generate with `ts-proto` or `@grpc/grpc-js` |

CI should run `buf generate` or equivalent to keep stubs in sync with protos.

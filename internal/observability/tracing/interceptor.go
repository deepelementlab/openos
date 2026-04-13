package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor records one server span per RPC with standard attributes.
func UnaryServerInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	tracer := otel.Tracer(serviceName)
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx, span := tracer.Start(ctx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("rpc.service", serviceName),
				attribute.String("rpc.method", info.FullMethod),
			),
		)
		defer span.End()

		// Propagate trace context into metadata for logging downstream
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if v := md.Get("x-trace-id"); len(v) > 0 {
				span.SetAttributes(attribute.String("aos.trace_id", v[0]))
			}
		}

		resp, err := handler(ctx, req)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			if s, ok := status.FromError(err); ok {
				span.SetAttributes(attribute.Int("rpc.grpc.status_code", int(s.Code())))
			}
			return resp, err
		}
		span.SetStatus(codes.Ok, "")
		return resp, nil
	}
}

// StreamServerInterceptor records a server span for streaming RPCs.
func StreamServerInterceptor(serviceName string) grpc.StreamServerInterceptor {
	tracer := otel.Tracer(serviceName)
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		ctx, span := tracer.Start(ctx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("rpc.service", serviceName),
				attribute.String("rpc.method", info.FullMethod),
			),
		)
		defer span.End()

		wrapped := &serverStream{ServerStream: ss, ctx: ctx}
		err := handler(srv, wrapped)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.SetStatus(codes.Ok, "")
		return nil
	}
}

type serverStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *serverStream) Context() context.Context { return s.ctx }

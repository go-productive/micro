package micro

import (
	"context"
	"google.golang.org/grpc/metadata"
	"time"
)

const (
	Timeout            = time.Second * 5
	metadataKeyTraceID = "trace_id"
)

func AppendToContext(ctx context.Context, key, value string) context.Context {
	if ctx == nil {
		ctx = context.TODO()
	}
	return metadata.AppendToOutgoingContext(ctx, key, value)
}

func FromContext(key string, ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	md, _ := metadata.FromIncomingContext(ctx)
	if value := md[key]; len(value) > 0 {
		return value[0], true
	}
	return "", false
}

func AppendTraceIDToCtx(traceID string, ctx context.Context) context.Context {
	return AppendToContext(ctx, metadataKeyTraceID, traceID)
}

func TraceIDFromContext(ctx context.Context) (string, bool) {
	return FromContext(metadataKeyTraceID, ctx)
}

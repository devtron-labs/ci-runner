package context

import (
	"context"
)

type CiContext struct {
	context.Context // Embedding original Go context
	ciRequest       CommonWorkflowRequest
}

func BuildCiContext(ctx context.Context, ciRequest *CommonWorkflowRequest) CiContext {
	return CiContext{
		Context:   ctx,
		ciRequest: *ciRequest,
	}
}

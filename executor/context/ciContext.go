package context

import (
	"context"
)

type CiContext struct {
	context.Context     // Embedding original Go context
	EnableSecretMasking bool
}

func BuildCiContext(ctx context.Context, enableSecretMasking bool) CiContext {
	return CiContext{
		Context:             ctx,
		EnableSecretMasking: enableSecretMasking,
	}
}

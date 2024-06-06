package context

import (
	"context"
)

type CiContext struct {
	context.Context     // Embedding original Go context
	enableSecretMasking bool
}

func BuildCiContext(ctx context.Context, enableSecretMasking bool) CiContext {
	return CiContext{
		Context:             ctx,
		enableSecretMasking: enableSecretMasking,
	}
}

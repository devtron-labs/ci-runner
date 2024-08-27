package dockerOperations

import (
	"context"
	"github.com/devtron-labs/common-lib/utils/bean"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

// GetImageDigestByImage fetches imageDigest from image using docker api client
func GetImageDigestByImage(ctx context.Context, image string, dockerAuth *bean.DockerAuthConfig) (string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logrus.Error()
		return "", err
	}
	var encodedRegistryAuth string
	if dockerAuth != nil {
		encodedRegistryAuth, err = dockerAuth.GetEncodedRegistryAuth()
		if err != nil {
			return "", err
		}
	}
	inspectOutput, err := cli.DistributionInspect(ctx, image, encodedRegistryAuth)
	if err != nil {
		return "", err
	}

	return inspectOutput.Descriptor.Digest.String(), nil
}

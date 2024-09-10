package helper

import "time"

type Kind string
type CredentialSourceType string
type ArtifactType string

const (
	PluginArtifactsKind           Kind                 = "PluginArtifacts"
	GlobalContainerRegistrySource CredentialSourceType = "global_container_registry"
	ArtifactTypeContainer         ArtifactType         = "CONTAINER"
)

type PluginArtifacts struct {
	Kind      Kind       `json:"Kind"`
	Artifacts []Artifact `json:"Artifacts"`
}

func NewPluginArtifact() *PluginArtifacts {
	return &PluginArtifacts{
		Kind:      PluginArtifactsKind,
		Artifacts: make([]Artifact, 0),
	}
}

func (p *PluginArtifacts) MergePluginArtifact(pluginArtifact *PluginArtifacts) {
	if pluginArtifact == nil {
		return
	}
	p.Artifacts = append(p.Artifacts, pluginArtifact.Artifacts...)
}

type Artifact struct {
	Type                      ArtifactType         `json:"Type"`
	Data                      []string             `json:"Data"`
	CredentialsSourceType     CredentialSourceType `json:"CredentialsSourceType"`
	CredentialSourceValue     string               `json:"CredentialSourceValue"`
	CreatedByPluginIdentifier string               `json:"createdByPluginIdentifier"`
	CreatedOn                 time.Time            `json:"createdOn"`
}

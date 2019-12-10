package main

type TaskYaml struct {
	Version      string         `yaml:"version"`
	PipelineConf PipelineConfig `yaml:"pipelineConf"`
}

type PipelineConfig struct {
	AppliesTo []AppliesTo `yaml:"appliesTo"`
}

type AppliesTo struct {
	Type string `yaml:"type"`
}

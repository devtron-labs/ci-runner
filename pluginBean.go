package main

type WorkFlowRequest struct {
	PreCiSteps  []*StepObject      `json:"preCiSteps"`
	PostCiSteps []*StepObject      `json:"postCiSteps"`
	RefPlugins  []*RefPluginObject `json:"refPlugins"`
}

type RefPluginObject struct {
	Id    int           `json:"id"`
	Steps []*StepObject `json:"steps"`
}

type StepObject struct {
	Index                    int                `json:"index"`
	StepType                 string             `json:"stepType"` // REF_PLUGIN or INLINE
	RefPluginId              int                `json:"refPluginId"`
	Script                   string             `json:"script"`
	InputVars                []*VariableObject  `json:"inputVars"`
	ExposedPorts             map[int]int        `json:"exposedPorts"` //map of host:container
	OutputVars               []*VariableObject  `json:"outputVars"`
	TriggerSkipConditions    []*ConditionObject `json:"triggerSkipConditions"`
	SuccessFailureConditions []*ConditionObject `json:"successFailureConditions"`
	DockerImage              string             `json:"dockerImage"`
	Command                  string             `json:"command"`
	Args                     []string           `json:"args"`
	CustomScriptMount        *MountPath         `json:"customScriptMountDestinationPath"` // destination path - storeScriptAt
	SourceCodeMount          *MountPath         `json:"sourceCodeMountDestinationPath"`   // destination path - mountCodeToContainerPath
	ExtraVolumeMounts        []*MountPath       `json:"extraVolumeMounts"`                // filePathMapping
	ArtifactPaths            []*string          `json:"artifactPaths"`
}

type VariableObject struct {
	Name   string `json:"name"`
	Format Format `json:"format"`

	//only for input type
	Value                      string      `json:"value"`
	GlobalVarName              string      `json:"globalVarName"`
	ReferenceVariableName      string      `json:"referenceVariableName"`
	ReferenceVariableStepIndex int         `json:"referenceVariableStepIndex"`
	DeducedValue               interface{} `json:"-"` //typeCased and deduced
}

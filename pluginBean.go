package main

type RefPluginObject struct {
	Id    int           `json:"id"`
	Steps []*StepObject `json:"steps"`
}

const STEP_TYPE_INLINE = "INLINE"
const STEP_TYPE_REF_PLUGIN = "REF_PLUGIN"

/*script string,
envInputVars map[string]string,
outputVars []string
trigger/skip ConditionObject
success/fail condition
ArtifactPaths
*/

type StepObject struct {
	Index                    int                `json:"index"`
	StepType                 string             `json:"stepType"`     // REF_PLUGIN or INLINE
	ExecutorType             ExecutorType       `json:"executorType"` //continer_image/ shell
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
	ArtifactPaths            []string           `json:"artifactPaths"`
}

//------------
type Format int

const (
	STRING Format = iota
	NUMBER
	BOOL
	DATE
)

func (d Format) ValuesOf(format string) Format {
	if format == "NUMBER" {
		return NUMBER
	} else if format == "BOOL" {
		return BOOL
	} else if format == "STRING" {
		return STRING
	} else if format == "DATE" {
		return DATE
	}
	return STRING
}

func (d Format) String() string {
	return [...]string{"NUMBER", "BOOL", "STRING", "DATE"}[d]
}

type ExecutorType int

const (
	CONTAINER_IMAGE ExecutorType = iota
	SHELL
)

func (d ExecutorType) ValueOf(executorType string) ExecutorType {
	if executorType == "CONTAINER_IMAGE" {
		return CONTAINER_IMAGE
	} else if executorType == "SHELL" {
		return SHELL
	}
	return SHELL
}
func (d ExecutorType) String() string {
	return [...]string{"CONTAINER_IMAGE", "SHELL"}[d]
}

type ReferenceVariableStage int

const (
	PREE_CI ReferenceVariableStage = iota
	POST_CI
)

func (d ReferenceVariableStage) ValueOf(referenceVariableStage string) ReferenceVariableStage {
	if referenceVariableStage == "PREE_CI" {
		return PREE_CI
	} else if referenceVariableStage == "POST_CI" {
		return POST_CI
	}
	return PREE_CI
}

type VariableObject struct {
	Name   string `json:"name"`
	Format Format `json:"format"`
	//only for input type
	Value                      string                 `json:"value"`
	GlobalVarName              string                 `json:"globalVarName"`
	ReferenceVariableName      string                 `json:"referenceVariableName"`
	ReferenceVariableStage     ReferenceVariableStage `json:"referenceVariableStage"`
	ReferenceVariableStepIndex int                    `json:"referenceVariableStepIndex"`
	DeducedValue               interface{}            `json:"-"` //typeCased and deduced
}

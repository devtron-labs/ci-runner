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

type VariableType int

const (
	VALUE VariableType = iota
	REF_PRE_CI
	REF_POST_CI
	REF_GLOBAL
	REF_PLUGIN
)

func (d VariableType) ValueOf(variableType string) VariableType {
	if variableType == "VALUE" {
		return VALUE
	} else if variableType == "REF_PRE_CI" {
		return REF_PRE_CI
	} else if variableType == "REF_POST_CI" {
		return REF_POST_CI
	} else if variableType == "REF_GLOBAL" {
		return REF_GLOBAL
	} else if variableType == "REF_PLUGIN" {
		return REF_PLUGIN
	}
	return VALUE
}
func (d VariableType) String() string {
	return [...]string{"VALUE", "REF_PRE_CI", "REF_POST_CI", "REF_GLOBAL", "REF_PLUGIN"}[d]
}

type VariableObject struct {
	Name   string `json:"name"`
	Format Format `json:"format"`
	//only for input type
	Value string `json:"value"`
	//	GlobalVarName              string       `json:"globalVarName"`
	ReferenceVariableName      string       `json:"referenceVariableName"`
	VariableType               VariableType `json:"variableType"`
	ReferenceVariableStepIndex int          `json:"referenceVariableStepIndex"`
	TypedValue                 interface{}  `json:"-"` //typeCased and deduced
}

func (v *VariableObject) TypeCheck() error {
	typedValue, err := typeConverter(v.Value, v.Format)
	if err != nil {
		return err
	}
	v.TypedValue = typedValue
	return nil
}

type ConditionType int

const (
	TRIGGER = iota
	SKIP
	SUCCESS
	FAILURE
)

func (d ConditionType) ValueOf(executorType string) ConditionType {
	if executorType == "TRIGGER" {
		return TRIGGER
	} else if executorType == "SKIP" {
		return SKIP
	} else if executorType == "SUCCESS" {
		return SUCCESS
	} else if executorType == "FAILURE" {
		return FAILURE
	}
	return SUCCESS
}
func (d ConditionType) String() string {
	return [...]string{"TRIGGER", "SKIP", "SUCCESS", "FAILURE"}[d]
}

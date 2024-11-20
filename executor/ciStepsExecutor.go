package executor

import (
	"github.com/devtron-labs/ci-runner/helper"
)

type CiStep interface {
	Run(input interface{}) (output interface{}, err error)
	HandleFailure()
	HandleSuccess()
	RunCleanUp()
	ParseInputParams(stepsInputMap, stepsOutputMap map[string]interface{}) (interface{}, error)
	GetStepName() string
}

type CiExecutor struct {
	steps          []CiStep
	stepsInputMap  map[string]interface{}
	stepsOutputMap map[string]interface{}
}

func (e *CiExecutor) AddSteps(step CiStep) {
	e.steps = append(e.steps, step)
}

func (e *CiExecutor) SortSteps(requiredOrder []string) {
	var sortedSteps []CiStep
	// sort e.steps
	e.steps = sortedSteps
}

func (e *CiExecutor) RunSteps(request helper.CommonWorkflowRequest) error {
	for _, step := range e.steps {

		input, err := step.ParseInputParams(e.stepsInputMap, e.stepsOutputMap)
		if err != nil {
		}

		e.stepsOutputMap[step.GetStepName()] = input

		output, err := step.Run(input)
		if err != nil {
			step.HandleFailure()
			return err
		}

		e.stepsOutputMap[step.GetStepName()] = output

		step.HandleSuccess()
		step.RunCleanUp()
	}
	return nil
}

type CacheDownloadStep struct {
	name   string // cacheDownloader
	action string // upload/download
}

func NewCacheDownloadHandler(name string, request helper.CommonWorkflowRequest) *CacheDownloadStep {
	return &CacheDownloadStep{
		name: name,
	}
}

type CacheDownloadInput struct {
}

type CacheDownloadOutput struct {
}

func (cacheDownloader *CacheDownloadStep) Run(input interface{}) (output interface{}, err error) {

	input = input.(*CacheDownloadInput)
	//d := CacheDownloadInput{}
	return CacheDownloadOutput{}, nil
}

func (cacheDownloader *CacheDownloadStep) HandleFailure() {

}

func (cacheDownloader *CacheDownloadStep) HandleSuccess() {

}

func (cacheDownloader *CacheDownloadStep) RunCleanUp() {

}

func (cacheDownloader *CacheDownloadStep) ParseInputParams(stepsInputMap, stepsOutputMap map[string]interface{}) (interface{}, error) {
	return nil, nil
}

func (cacheDownloader *CacheDownloadStep) GetStepName() string {
	return "cacheDownloadStep"
}

func run(request helper.CommonWorkflowRequest) {
	executor := &CiExecutor{}

	cacheDownloader := NewCacheDownloadHandler("cacheDownloadStep", request)

	executor.AddSteps(cacheDownloader)

	executor.SortSteps([]string{"cacheDownloadStep"})

	err := executor.RunSteps(request)
	if err != nil {
	}
}

type DockerBuildStep struct {
	dockerRegistryDetails interface{}
	name                  string
}

type DockerBuildStepInput struct {
	input1 interface{}
}

type DockerBuildStepOutput struct {
	dest   string
	digest string
}

func (dockerBuildStep *DockerBuildStep) Run(input interface{}) (output interface{}, err error) {

	_ = input.(DockerBuildStepInput)
	stepOutput := DockerBuildStepOutput{
		dest:   "",
		digest: "",
	}
	return stepOutput, nil
}

func (dockerBuildStep *DockerBuildStep) ParseInputParams(stepsInputMap, stepsOutputMap map[string]interface{}) (interface{}, error) {
	cacheDownloadInput := stepsOutputMap["cacheDownloadStep"].(CacheDownloadOutput)
	dockerInput := DockerBuildStepInput{cacheDownloadInput}
	return dockerInput, nil
}

func (dockerBuildStep *DockerBuildStep) HandleFailure() {

}

func (dockerBuildStep *DockerBuildStep) HandleSuccess() {

}

func (dockerBuildStep *DockerBuildStep) RunCleanUp() {

}

func (dockerBuildStep *DockerBuildStep) GetStepName() string {
	return dockerBuildStep.name
}

// Needed steps

type DockerTagStep struct {
}

type DockerPushStep struct {
}

type PostCiStep struct {
}

type DockerDaemonStep struct {
	name   string // dockerDaemonStep
	action string // start/stop
}

type DockerLoginStep struct {
	name   string //
	action string // login
}

type PreCISteps struct {
	name string // PreCiStep
}

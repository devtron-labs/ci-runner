package helper

import "github.com/devtron-labs/ci-runner/util"

func IsCIOrJobTypeEvent(eventType string) bool {
	return eventType == util.CIEVENT || eventType == util.JOBEVENT
}

func IsEventTypeEligibleToUploadLogs(eventType string) bool {
	return eventType == util.CIEVENT || eventType == util.JOBEVENT || eventType == util.CDSTAGE
}

func IsEventTypeEligibleToScanImage(eventType string) bool {
	return eventType == util.CIEVENT
}

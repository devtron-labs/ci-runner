/*
 *  Copyright 2020 Devtron Labs
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package main

import (
	"encoding/json"
	"fmt"
	_ "github.com/aws/aws-sdk-go/aws"
	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/util"
	blob_storage "github.com/devtron-labs/common-lib/blob-storage"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
)

var handleOnce sync.Once

func handleCleanup(ciCdRequest helper.CiCdTriggerEvent, exitCode *int, source string) {
	handleOnce.Do(func() {
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go cleanUpBuildxK8sDriver(ciCdRequest, wg)
		log.Println(util.DEVTRON, " CI-Runner cleanup executed with exit Code", *exitCode, source)
		uploadLogs(ciCdRequest, exitCode)
		wg.Wait()
		log.Println(util.DEVTRON, " Exiting with exit code ", *exitCode)
		os.Exit(*exitCode)
	})
}

func main() {
	args := `{"type":"CI","ciRequest":{"DockerBuildTargetPlatform":"linux/arm64", "workflowNamePrefix":"16-ci-25-w5x1-70","pipelineName":"ci-25-w5x1","pipelineId":70,"dockerImageTag":"da3ba326-70-17","dockerRegistryId":"devtron-quay","dockerRegistryType":"dcp","dockerRegistryURL":"asia-south1-docker.pkg.dev/deepak-test-project-354711/deepak-test","dockerConnection":"secure","dockerCert":"","dockerBuildArgs":"{}","dockerRepository":"test","dockerfileLocation":"Dockerfile","dockerUsername":"_json_key","dockerPassword":"{
  "type": "service_account",
  "project_id": "deepak-test-project-354711",
  "private_key_id": "7b2420bf5c50beeffdc92d5781dad9b1e28b3d0c",
  "private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDXSYOokLfsQmP4\n+P5D8ntY3SrrCAHU5toWzvHRgRMTGvDD2ygyCdebsKcsKKTFoYjD4JQdmK8ot6l+\n/+aMipu4WI2gWLs0h5+JxaC1a4GR/Z9+vqhG2bCtKIXtxcP3JhZZYkKzyjrM5QJ2\nYCP/Sc/+W+wEzHUUA+C5+oRyCDp7jQ7PJf5tP+qjOjyh313YMcl9yMzpTDMr3vvO\nZtiBhDbrWKJtW5UBc+UDv0gbmfeP4MIuwqVp5OqLmi8BFehD4ijCyjPv9s7agVQV\nyYrzczEAogkSTT6NKf3ruDv0GoTHthtKWQB61rRCCYoREFZTjB9EKNYv1cPzlR4B\n6Ym845LbAgMBAAECggEAFgfZV74bfCtdFKTSTDy7n5/eXPfQTCulhLD/sPs+6vUW\nT0yVg+1a6X091jiAiBLcLoNEVPUDc6y7xRnFy3sBrW8uawq5eYopas1VAUuzki98\ni1lSWhG70TR05ByZTajRn6r8/W4D72S+CEQVWvIAPVzFvcX4cyhkblOzCwJJjcv0\n5DDsBRoSyGD132bkGTx2sQojlGqLT2+SSElrgDpIlX4aT5VCivnAdwehjdJmlB03\nJF9ZE5rgQFw8NkFLOW8zAWD3kaKcfsYyliyphenIIwmW4VmH49whFfVAmVsCT3r5\n+8Pv2lIEkOTvCgYZ80SbSViKj7Jby5H633pzFxSx8QKBgQD5508V8jBDmd3N0qHh\nXSoHVU+zOPatHTfQpWsj1GbqHUKrAwaJZ/cwiJnZICX4QcCAEnvRXTEKgQYxaRgG\nnavwIG+6XWkGD28C0nXNMkQO1gjyhFIe5ma7WZIK5OuYtd7GbUHiCB+XCBA+psSZ\nERUcVdZaM8328RuU/vg+mQ4VcQKBgQDcigUgU0hN1KLoX47jSrPIb+/2kYsvSxCI\nqumoltSNMMWt6IG9ZTfwyCxqcQsfrKXGGwlV1gELkT/H1DUsnDazX38Qjr+1T5P5\nQzP+ZVmUJ5fGOWBTzGZGeu2BEEReUZPJpDK2dT1Uu1alhYuVNAO/zrQOJ6A4u8n3\nMkNv11OXCwKBgETqVfPuYwLxdqpg8MVuZL26+Ayro9MfoJnIVGCAHZVoVk9EuVPB\nOPjIYuzuoanxr/1hm4WkFncYF7YejkKczqKcv1L8mY7TSMDVeykIOJ6CxdrjRKZC\n0YfO7qhUcugdF39O+AE2TkffMGOmp8ayYEj9HuynJqB34yxWl+zjVm/xAoGAaBHB\nZYWnYwLqdRlSxjMkL3uTExmPQpv7i2KLrICwgIf5YJ2NS6COC1OKkhgSFbpU5+0u\nNJEuIRVDsbqT9R8qOO7heSDDmn2Y6FEsIeoVoXIljubYa/LSeIPdu7+/Y3q/cLHJ\nNIySin903drtCVVoR4T1NpDAbMVBAyN26zoDOg0CgYEA0zHgCLwq0oOwwjq2UA6E\neVJpVByXop3HmuNKSNKdCd6K/Aq4tysCjPlkojB51doWlTu4sEnHilDpeRXJ2fKV\nqAgbDAaftFlGb0rqBOYgFduej1zyCVGusyLLJckLF3P9GLn5TTrmwTZDIoqKGBJB\nRTRFqcvkP/w4mJPet96n/IY=\n-----END PRIVATE KEY-----\n",
  "client_email": "deepak-sa@deepak-test-project-354711.iam.gserviceaccount.com",
  "client_id": "105475086115839446199",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/deepak-sa%40deepak-test-project-354711.iam.gserviceaccount.com"
}","awsRegion":"","accessKey":"","secretKey":"","ciCacheLocation":"","ciCacheRegion":"","ciCacheFileName":"ci-25-w5x1-70.tar.gz","ciProjectDetails":[{"gitRepository":"https://github.com/devtron-labs/getting-started-nodejs","materialName":"1-getting-started-nodejs","checkoutPath":"./","fetchSubmodules":false,"commitHash":"da3ba3254712965b5944a6271e71bff91fe51f20","gitTag":"","commitTime":"2022-04-12T20:26:08+05:30","type":"SOURCE_TYPE_BRANCH_FIXED","message":"Update README.md","author":"Prakarsh \u003c71125043+prakarsh-dt@users.noreply.github.com\u003e","gitOptions":{"userName":"","password":"","sshPrivateKey":"","accessToken":"","authMode":"ANONYMOUS"},"sourceType":"SOURCE_TYPE_BRANCH_FIXED","sourceValue":"master","WebhookData":{"Id":0,"EventActionType":"","Data":null}}],"containerResources":{"minCpu":"","maxCpu":"","minStorage":"","maxStorage":"","minEphStorage":"","maxEphStorage":"","minMem":"","maxMem":""},"activeDeadlineSeconds":3600,"ciImage":"quay.io/devtron/ci-runner:1290cf23-182-8015","namespace":"devtron-ci","workflowId":16,"triggeredBy":8,"cacheLimit":5000000000,"beforeDockerBuildScripts":null,"afterDockerBuildScripts":null,"ciArtifactLocation":"","invalidateCache":true,"scanEnabled":false,"cloudProvider":"AZURE","azureBlobConfig":{"enabled":true,"accountName":"devtrondemoblob","blobContainerCiLog":"","blobContainerCiCache":"cache","accountKey":"y1/K13YMp/v7uuvZNkKJ4dS3CyGc37bPIN9Hv8MVhog6OkG0joV05proQReMQIJQ8qXp0JVpj+mz+AStHNKR3Q=="},"minioEndpoint":"","defaultAddressPoolBaseCidr":"","defaultAddressPoolSize":0,"preCiSteps":[{"name":"Task 1","index":1,"stepType":"INLINE","executorType":"SHELL","refPluginId":0,"script":"echo $","inputVars":null,"exposedPorts":{"0":0},"outputVars":null,"triggerSkipConditions":null,"successFailureConditions":null,"dockerImage":"","command":"","args":null,"customScriptMountDestinationPath":{"sourcePath":"","destinationPath":""},"sourceCodeMountDestinationPath":{"sourcePath":"","destinationPath":""},"extraVolumeMounts":null,"artifactPaths":null}],"postCiSteps":null,"refPlugins":null},"cdRequest":null}`
	//args := `{"type":"DryRun","dryRunRequest":{"buildPackParams":{"builderId":"gcr.io/buildpacks/builder:v1"},"DockerBuildTargetPlatform":"", "workflowNamePrefix":"16-ci-25-w5x1-70","pipelineName":"ci-25-w5x1","pipelineId":70,"dockerImageTag":"da3ba326-70-17","dockerRegistryId":"devtron-quay","dockerRegistryType":"other","dockerRegistryURL":"https://quay.io/devtron","dockerConnection":"secure","dockerCert":"","dockerBuildArgs":"{}","dockerRepository":"test","dockerfileLocation":"Dockerfile","dockerUsername":"devtron+devtest","dockerPassword":"5WEDXDJMP6RV1CG1KKFJQL3MQOLC64JKM6K684WPEBKVWKOZ4LSMBHEHJU1HBGXK","awsRegion":"","accessKey":"","secretKey":"","ciCacheLocation":"","ciCacheRegion":"","ciCacheFileName":"ci-25-w5x1-70.tar.gz","ciProjectDetails":[{"gitRepository":"https://github.com/devtron-labs/sample-go-app","materialName":"1-getting-started-nodejs","checkoutPath":"./","fetchSubmodules":false,"commitHash":"8654623ec2bd9efd663935cb8332c8c765541837","gitTag":"","commitTime":"2022-04-12T20:26:08+05:30","type":"SOURCE_TYPE_BRANCH_FIXED","message":"Update README.md","author":"Prakarsh \u003c71125043+prakarsh-dt@users.noreply.github.com\u003e","gitOptions":{"userName":"","password":"","sshPrivateKey":"","accessToken":"","authMode":"ANONYMOUS"},"sourceType":"SOURCE_TYPE_BRANCH_FIXED","sourceValue":"master","WebhookData":{"Id":0,"EventActionType":"","Data":null}}],"containerResources":{"minCpu":"","maxCpu":"","minStorage":"","maxStorage":"","minEphStorage":"","maxEphStorage":"","minMem":"","maxMem":""},"activeDeadlineSeconds":3600,"ciImage":"quay.io/devtron/ci-runner:1290cf23-182-8015","namespace":"devtron-ci","workflowId":16,"triggeredBy":8,"cacheLimit":5000000000,"beforeDockerBuildScripts":null,"afterDockerBuildScripts":null,"ciArtifactLocation":"","invalidateCache":true,"scanEnabled":false,"cloudProvider":"AZURE","azureBlobConfig":{"enabled":true,"accountName":"devtrondemoblob","blobContainerCiLog":"","blobContainerCiCache":"cache","accountKey":"y1/K13YMp/v7uuvZNkKJ4dS3CyGc37bPIN9Hv8MVhog6OkG0joV05proQReMQIJQ8qXp0JVpj+mz+AStHNKR3Q=="},"minioEndpoint":"","defaultAddressPoolBaseCidr":"","defaultAddressPoolSize":0,"preCiSteps":[{"name":"Task 1","index":1,"stepType":"INLINE","executorType":"SHELL","refPluginId":0,"script":"echo $","inputVars":null,"exposedPorts":{"0":0},"outputVars":null,"triggerSkipConditions":null,"successFailureConditions":null,"dockerImage":"","command":"","args":null,"customScriptMountDestinationPath":{"sourcePath":"","destinationPath":""},"sourceCodeMountDestinationPath":{"sourcePath":"","destinationPath":""},"extraVolumeMounts":null,"artifactPaths":null}],"postCiSteps":null,"refPlugins":null},"cdRequest":null}`
	//' {"workflowNamePrefix":"55-suraj-23-ci-suraj-test-pipeline-8","pipelineName":"suraj-23-ci-suraj-test-pipeline","pipelineId":8,"dockerImageTag":"a6b809c4be87c217feba4af15cf5ebc3cafe21e0","dockerRegistryURL":"686244538589.dkr.ecr.us-east-2.amazonaws.com","dockerRepository":"test/suraj-23","dockerfileLocation":"./notifier/Dockerfile","awsRegion":"us-east-2","ciCacheLocation":"ci-caching","ciCacheFileName":"suraj-23-ci-suraj-test-pipeline.tar.gz","ciProjectDetails":[{"gitRepository":"https://gitlab.com/devtron/notifier.git","materialName":"1-notifier","checkoutPath":"./notifier","commitHash":"d4df38bcd065004014d255c2203d592a91585955","commitTime":"0001-01-01T00:00:00Z","branch":"ci_with_argo","type":"SOURCE_TYPE_BRANCH_FIXED","message":"test-commit","gitOptions":{"userName":"Suraj24","password":"Devtron@1234","sshKey":"","accessToken":"","authMode":"USERNAME_PASSWORD"}},{"gitRepository":"https://gitlab.com/devtron/orchestrator.git","materialName":"2-orchestrator","checkoutPath":"./orch","commitHash":"","commitTime":"0001-01-01T00:00:00Z","branch":"ci_with_argo","type":"SOURCE_TYPE_BRANCH_FIXED","message":"","gitOptions":{"userName":"Suraj24","password":"Devtron@1234","sshKey":"","accessToken":"","authMode":""}}],"ciImage":"686244538589.dkr.ecr.us-east-2.amazonaws.com/cirunner:latest","namespace":"default"}'
	//args = `{"type":"CI","commonWorkflowRequest":{"workflowNamePrefix":"25-ci-1-315f-3","pipelineName":"ci-1-315f","pipelineId":3,"dockerImageTag":"4534a6a4-3-25","dockerRegistryId":"ayush-gcr-test","dockerRegistryType":"artifact-registry","dockerRegistryURL":"asia-south1-docker.pkg.dev/deepak-test-project-354711/deepak-test","dockerConnection":"","dockerCert":"","dockerRepository":"deepak-test-project-354711/deepak-test/test","checkoutPath":"","dockerUsername":"_json_key","dockerPassword":"'{\n  \"type\": \"service_account\",\n  \"project_id\": \"deepak-test-project-354711\",\n  \"private_key_id\": \"7b2420bf5c50beeffdc92d5781dad9b1e28b3d0c\",\n  \"private_key\": \"-----BEGIN PRIVATE KEY-----\\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDXSYOokLfsQmP4\\n+P5D8ntY3SrrCAHU5toWzvHRgRMTGvDD2ygyCdebsKcsKKTFoYjD4JQdmK8ot6l+\\n/+aMipu4WI2gWLs0h5+JxaC1a4GR/Z9+vqhG2bCtKIXtxcP3JhZZYkKzyjrM5QJ2\\nYCP/Sc/+W+wEzHUUA+C5+oRyCDp7jQ7PJf5tP+qjOjyh313YMcl9yMzpTDMr3vvO\\nZtiBhDbrWKJtW5UBc+UDv0gbmfeP4MIuwqVp5OqLmi8BFehD4ijCyjPv9s7agVQV\\nyYrzczEAogkSTT6NKf3ruDv0GoTHthtKWQB61rRCCYoREFZTjB9EKNYv1cPzlR4B\\n6Ym845LbAgMBAAECggEAFgfZV74bfCtdFKTSTDy7n5/eXPfQTCulhLD/sPs+6vUW\\nT0yVg+1a6X091jiAiBLcLoNEVPUDc6y7xRnFy3sBrW8uawq5eYopas1VAUuzki98\\ni1lSWhG70TR05ByZTajRn6r8/W4D72S+CEQVWvIAPVzFvcX4cyhkblOzCwJJjcv0\\n5DDsBRoSyGD132bkGTx2sQojlGqLT2+SSElrgDpIlX4aT5VCivnAdwehjdJmlB03\\nJF9ZE5rgQFw8NkFLOW8zAWD3kaKcfsYyliyphenIIwmW4VmH49whFfVAmVsCT3r5\\n+8Pv2lIEkOTvCgYZ80SbSViKj7Jby5H633pzFxSx8QKBgQD5508V8jBDmd3N0qHh\\nXSoHVU+zOPatHTfQpWsj1GbqHUKrAwaJZ/cwiJnZICX4QcCAEnvRXTEKgQYxaRgG\\nnavwIG+6XWkGD28C0nXNMkQO1gjyhFIe5ma7WZIK5OuYtd7GbUHiCB+XCBA+psSZ\\nERUcVdZaM8328RuU/vg+mQ4VcQKBgQDcigUgU0hN1KLoX47jSrPIb+/2kYsvSxCI\\nqumoltSNMMWt6IG9ZTfwyCxqcQsfrKXGGwlV1gELkT/H1DUsnDazX38Qjr+1T5P5\\nQzP+ZVmUJ5fGOWBTzGZGeu2BEEReUZPJpDK2dT1Uu1alhYuVNAO/zrQOJ6A4u8n3\\nMkNv11OXCwKBgETqVfPuYwLxdqpg8MVuZL26+Ayro9MfoJnIVGCAHZVoVk9EuVPB\\nOPjIYuzuoanxr/1hm4WkFncYF7YejkKczqKcv1L8mY7TSMDVeykIOJ6CxdrjRKZC\\n0YfO7qhUcugdF39O+AE2TkffMGOmp8ayYEj9HuynJqB34yxWl+zjVm/xAoGAaBHB\\nZYWnYwLqdRlSxjMkL3uTExmPQpv7i2KLrICwgIf5YJ2NS6COC1OKkhgSFbpU5+0u\\nNJEuIRVDsbqT9R8qOO7heSDDmn2Y6FEsIeoVoXIljubYa/LSeIPdu7+/Y3q/cLHJ\\nNIySin903drtCVVoR4T1NpDAbMVBAyN26zoDOg0CgYEA0zHgCLwq0oOwwjq2UA6E\\neVJpVByXop3HmuNKSNKdCd6K/Aq4tysCjPlkojB51doWlTu4sEnHilDpeRXJ2fKV\\nqAgbDAaftFlGb0rqBOYgFduej1zyCVGusyLLJckLF3P9GLn5TTrmwTZDIoqKGBJB\\nRTRFqcvkP/w4mJPet96n/IY=\\n-----END PRIVATE KEY-----\\n\",\n  \"client_email\": \"deepak-sa@deepak-test-project-354711.iam.gserviceaccount.com\",\n  \"client_id\": \"105475086115839446199\",\n  \"auth_uri\": \"https://accounts.google.com/o/oauth2/auth\",\n  \"token_uri\": \"https://oauth2.googleapis.com/token\",\n  \"auth_provider_x509_cert_url\": \"https://www.googleapis.com/oauth2/v1/certs\",\n  \"client_x509_cert_url\": \"https://www.googleapis.com/robot/v1/metadata/x509/deepak-sa%40deepak-test-project-354711.iam.gserviceaccount.com\"\n}'","awsRegion":"","accessKey":"","secretKey":"","ciCacheLocation":"devtron-ci-cache","ciCacheRegion":"us-west-2","ciCacheFileName":"ci-1-315f-3.tar.gz","ciProjectDetails":[{"gitRepository":"https://github.com/amit24nov2000/Sample-App.git","materialName":"1-Sample-App","checkoutPath":"./","fetchSubmodules":false,"commitHash":"4534a6a4496d0aefbbde472a220f957bce0038e3","gitTag":"","commitTime":"2023-11-17T10:01:02Z","type":"SOURCE_TYPE_BRANCH_FIXED","message":"Update Dockerfile\n\nNginx changes","author":"Amit kumar \u003c69798873+amit24nov2000@users.noreply.github.com\u003e","gitOptions":{"userName":"","password":"","sshPrivateKey":"","accessToken":"","authMode":"ANONYMOUS"},"sourceType":"SOURCE_TYPE_BRANCH_FIXED","sourceValue":"main","WebhookData":{"id":0,"eventActionType":"","data":null}}],"containerResources":{"minCpu":"","maxCpu":"","minStorage":"","maxStorage":"","minEphStorage":"","maxEphStorage":"","minMem":"","maxMem":""},"activeDeadlineSeconds":3600,"ciImage":"quay.io/devtron/test:a4f6835d-82-2203","namespace":"devtron-ci","workflowId":25,"triggeredBy":2,"cacheLimit":5000000000,"beforeDockerBuildScripts":null,"afterDockerBuildScripts":null,"ciArtifactLocation":"s3://devtron-ci-log/devtron/ci-artifacts/25/25.zip","ciArtifactBucket":"devtron-ci-log","ciArtifactFileName":"devtron/ci-artifacts/25/25.zip","ciArtifactRegion":"","scanEnabled":false,"cloudProvider":"S3","blobStorageConfigured":false,"blobStorageS3Config":{"accessKey":"iCoD3HUkUgQNS07C0igYBSExITOw4gT9","passkey":"O2m6ERN15DU41WKLGzT743YpqpGPl6TS","endpointUrl":"http://devtron-minio.devtroncd:9000","isInSecure":true,"ciLogBucketName":"devtron-ci-log","ciLogRegion":"us-west-2","ciLogBucketVersioning":false,"ciCacheBucketName":"devtron-ci-cache","ciCacheRegion":"us-west-2","ciCacheBucketVersioning":false,"ciArtifactBucketName":"devtron-ci-log","ciArtifactRegion":"us-west-2","ciArtifactBucketVersioning":false},"azureBlobConfig":null,"gcpBlobConfig":null,"blobStorageLogsKey":"devtron/25-ci-1-315f-3","inAppLoggingEnabled":false,"defaultAddressPoolBaseCidr":"","defaultAddressPoolSize":0,"preCiSteps":[{"name":"Task 1","index":1,"stepType":"INLINE","executorType":"SHELL","script":"#!/bin/sh \nset -eo pipefail \n#set -v  ## uncomment this to debug the script \n","inputVars":null,"exposedPorts":null,"outputVars":null,"triggerSkipConditions":null,"successFailureConditions":null,"dockerImage":"","command":"","args":null,"customScriptMount":null,"sourceCodeMount":null,"extraVolumeMounts":null,"artifactPaths":null,"triggerIfParentStageFail":false}],"postCiSteps":null,"refPlugins":null,"appName":"ecr-test","triggerByAuthor":"admin","ciBuildConfig":{"id":3,"gitMaterialId":1,"buildContextGitMaterialId":1,"useRootBuildContext":true,"ciBuildType":"self-dockerfile-build","dockerBuildConfig":{"dockerfileRelativePath":"Dockerfile","dockerfileContent":"","buildContext":".","useBuildx":false,"buildxProvenanceMode":""},"buildPackConfig":null,"pipelineType":"CI_BUILD"},"ciBuildDockerMtuValue":-1,"ignoreDockerCachePush":false,"ignoreDockerCachePull":false,"cacheInvalidate":false,"IsPvcMounted":false,"extraEnvironmentVariables":{},"enableBuildContext":true,"appId":1,"environmentId":0,"orchestratorHost":"http://devtroncd-orchestrator-service-prod.devtroncd/webhook/msg/nats","orchestratorToken":"OGNPUEpvMWg5QzBXM3ZlbTJsVkNjNTRtVG9RPQo","isExtRun":false,"imageRetryCount":0,"imageRetryInterval":5,"workflowRunnerId":0,"cdPipelineId":0,"stageYaml":"","artifactLocation":"","ciArtifactDTO":{"id":0,"pipelineId":0,"image":"","imageDigest":"","materialInfo":"","dataSource":"","workflowId":null},"cdImage":"","stageType":"","cdCacheLocation":"","cdCacheRegion":"","workflowPrefixForLog":"","deploymentTriggerTime":"0001-01-01T00:00:00Z","workflowExecutor":"AWF","prePostDeploySteps":null,"ciArtifactLastFetch":"0001-01-01T00:00:00Z","ciPipelineType":"","useExternalClusterBlob":false,"registryDestinationImageMap":null,"registryCredentialMap":null,"pluginArtifactStage":"","pushImageBeforePostCI":false,"Type":"CI","Pipeline":null,"Env":null,"AppLabels":{},"Scope":{"appId":1,"envId":0,"clusterId":0}}}`
	args = `{"type":"CI ","commonWorkflowRequest":{"workflowNamePrefix":"25-ci-1-315f-3","pipelineName":"ci-1-315f","pipelineId":3,"dockerImageTag":"4534a6a4-3-25","dockerRegistryId":"ayush-gcr-test","dockerRegistryType":"artifact-registry","dockerRegistryURL":"asia-south1-docker.pkg.dev/deepak-test-project-354711/deepak-test","dockerConnection":"","dockerCert":"","dockerRepository":"deepak-test-project-354711/deepak-test/test","checkoutPath":"","dockerUsername":"_json_key","dockerPassword":"HSBFJHFBBSAF(SDG","awsRegion":"","accessKey":"","secretKey":"","ciCacheLocation":"devtron-ci-cache","ciCacheRegion":"us-west-2","ciCacheFileName":"ci-1-315f-3.tar.gz","ciProjectDetails":[{"gitRepository":"https://github.com/amit24nov2000/Sample-App.git","materialName":"1-Sample-App","checkoutPath":"./","fetchSubmodules":false,"commitHash":"4534a6a4496d0aefbbde472a220f957bce0038e3","gitTag":"","commitTime":"2023-11-17T10:01:02Z","type":"SOURCE_TYPE_BRANCH_FIXED","message":"Update Dockerfile\n\nNginx changes","author":"Amit kumar \u003c69798873+amit24nov2000@users.noreply.github.com\u003e","gitOptions":{"userName":"","password":"","sshPrivateKey":"","accessToken":"","authMode":"ANONYMOUS"},"sourceType":"SOURCE_TYPE_BRANCH_FIXED","sourceValue":"main","WebhookData":{"id":0,"eventActionType":"","data":null}}],"containerResources":{"minCpu":"","maxCpu":"","minStorage":"","maxStorage":"","minEphStorage":"","maxEphStorage":"","minMem":"","maxMem":""},"activeDeadlineSeconds":3600,"ciImage":"quay.io/devtron/test:a4f6835d-82-2203","namespace":"devtron-ci","workflowId":25,"triggeredBy":2,"cacheLimit":5000000000,"beforeDockerBuildScripts":null,"afterDockerBuildScripts":null,"ciArtifactLocation":"s3://devtron-ci-log/devtron/ci-artifacts/25/25.zip","ciArtifactBucket":"devtron-ci-log","ciArtifactFileName":"devtron/ci-artifacts/25/25.zip","ciArtifactRegion":"","scanEnabled":false,"cloudProvider":"S3","blobStorageConfigured":false,"blobStorageS3Config":{"accessKey":"iCoD3HUkUgQNS07C0igYBSExITOw4gT9","passkey":"O2m6ERN15DU41WKLGzT743YpqpGPl6TS","endpointUrl":"http://devtron-minio.devtroncd:9000","isInSecure":true,"ciLogBucketName":"devtron-ci-log","ciLogRegion":"us-west-2","ciLogBucketVersioning":false,"ciCacheBucketName":"devtron-ci-cache","ciCacheRegion":"us-west-2","ciCacheBucketVersioning":false,"ciArtifactBucketName":"devtron-ci-log","ciArtifactRegion":"us-west-2","ciArtifactBucketVersioning":false},"azureBlobConfig":null,"gcpBlobConfig":null,"blobStorageLogsKey":"devtron/25-ci-1-315f-3","inAppLoggingEnabled":false,"defaultAddressPoolBaseCidr":"","defaultAddressPoolSize":0,"preCiSteps":[{"name":"Task 1","index":1,"stepType":"INLINE","executorType":"SHELL","script":"#!/bin/sh \nset -eo pipefail \n#set -v  ## uncomment this to debug the script \n","inputVars":null,"exposedPorts":null,"outputVars":null,"triggerSkipConditions":null,"successFailureConditions":null,"dockerImage":"","command":"","args":null,"customScriptMount":null,"sourceCodeMount":null,"extraVolumeMounts":null,"artifactPaths":null,"triggerIfParentStageFail":false}],"postCiSteps":null,"refPlugins":null,"appName":"ecr-test","triggerByAuthor":"admin","ciBuildConfig":{"id":3,"gitMaterialId":1,"buildContextGitMaterialId":1,"useRootBuildContext":true,"ciBuildType":"self-dockerfile-build","dockerBuildConfig":{"dockerfileRelativePath":"Dockerfile","dockerfileContent":"","buildContext":".","useBuildx":false,"buildxProvenanceMode":""},"buildPackConfig":null,"pipelineType":"CI_BUILD"},"ciBuildDockerMtuValue":-1,"ignoreDockerCachePush":false,"ignoreDockerCachePull":false,"cacheInvalidate":false,"IsPvcMounted":false,"extraEnvironmentVariables":{},"enableBuildContext":true,"appId":1,"environmentId":0,"orchestratorHost":"http://devtroncd-orchestrator-service-prod.devtroncd/webhook/msg/nats","orchestratorToken":"OGNPUEpvMWg5QzBXM3ZlbTJsVkNjNTRtVG9RPQo","isExtRun":false,"imageRetryCount":0,"imageRetryInterval":5,"workflowRunnerId":0,"cdPipelineId":0,"stageYaml":"","artifactLocation":"","ciArtifactDTO":{"id":0,"pipelineId":0,"image":"","imageDigest":"","materialInfo":"","dataSource":"","workflowId":null},"cdImage":"","stageType":"","cdCacheLocation":"","cdCacheRegion":"","workflowPrefixForLog":"","deploymentTriggerTime":"0001-01-01T00:00:00Z","workflowExecutor":"AWF","prePostDeploySteps":null,"ciArtifactLastFetch":"0001-01-01T00:00:00Z","ciPipelineType":"","useExternalClusterBlob":false,"registryDestinationImageMap":null,"registryCredentialMap":null,"pluginArtifactStage":"","pushImageBeforePostCI":false,"Type":"CI","Pipeline":null,"Env":null,"AppLabels":{},"Scope":{"appId":1,"envId":0,"clusterId":0}}}`
	LoggingMode := "RunMode"
	if len(os.Args) > 1 {
		LoggingMode = os.Args[1]
	}

	if os.Getenv(util.InAppLogging) == "true" && LoggingMode == "PARENT_MODE" {
		log.Println(util.DEVTRON, " Starting in app logger... ")
		util.SpawnProcessWithLogging()
	}

	//args := os.Getenv(util.CiCdEventEnvKey)
	processEvent(args)
}

func processEvent(args string) {

	exitCode := 0
	ciCdRequest := &helper.CiCdTriggerEvent{}
	err := json.Unmarshal([]byte(args), ciCdRequest)
	if err != nil {
		log.Println(err)
		exitCode = util.DefaultErrorCode
		return
	}

	// Create a channel to receive the SIGTERM signal
	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, syscall.SIGTERM)

	go func() {
		var abortErrorCode = util.AbortErrorCode
		log.Println(util.DEVTRON, "SIGTERM listener started!")
		receivedSignal := <-sigTerm
		log.Println(util.DEVTRON, "signal received: ", receivedSignal)
		handleCleanup(*ciCdRequest, &abortErrorCode, util.Source_Signal)
	}()

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" || logLevel == "DEBUG" {
		log.Println(util.DEVTRON, " ci-cd request details -----> ", args)
	}

	defer handleCleanup(*ciCdRequest, &exitCode, util.Source_Defer)
	if ciCdRequest.Type == util.CIEVENT {
		HandleCIEvent(ciCdRequest, &exitCode)
	} else {
		HandleCDEvent(ciCdRequest, &exitCode)
	}
	return
}

func cleanUpBuildxK8sDriver(ciCdRequest helper.CiCdTriggerEvent, wg *sync.WaitGroup) {
	defer wg.Done()
	if valid, eligibleBuildxK8sDriverNodes := helper.ValidBuildxK8sDriverOptions(ciCdRequest.CommonWorkflowRequest); valid {
		log.Println(util.DEVTRON, "starting buildx k8s driver clean up ,before terminating ci-runner")
		err := helper.CleanBuildxK8sDriver(eligibleBuildxK8sDriverNodes)
		if err != nil {
			log.Println(util.DEVTRON, "error in cleaning up buildx K8s driver, err : ", err)
		}
	}
}

func uploadLogs(event helper.CiCdTriggerEvent, exitCode *int) {
	var storageModuleConfigured bool
	var blobStorageLogKey string
	var cloudProvider blob_storage.BlobStorageType
	var blobStorageS3Config *blob_storage.BlobStorageS3Config
	var azureBlobConfig *blob_storage.AzureBlobConfig
	var gcpBlobConfig *blob_storage.GcpBlobConfig
	var inAppLoggingEnabled bool

	if event.Type == util.CIEVENT && event.CommonWorkflowRequest.BlobStorageConfigured {
		storageModuleConfigured = true
		blobStorageLogKey = event.CommonWorkflowRequest.BlobStorageLogsKey
		cloudProvider = event.CommonWorkflowRequest.CloudProvider
		blobStorageS3Config = event.CommonWorkflowRequest.BlobStorageS3Config
		azureBlobConfig = event.CommonWorkflowRequest.AzureBlobConfig
		gcpBlobConfig = event.CommonWorkflowRequest.GcpBlobConfig
		inAppLoggingEnabled = event.CommonWorkflowRequest.InAppLoggingEnabled

	} else if event.Type == util.CDSTAGE && event.CommonWorkflowRequest.BlobStorageConfigured {
		storageModuleConfigured = true
		blobStorageLogKey = event.CommonWorkflowRequest.BlobStorageLogsKey
		cloudProvider = event.CommonWorkflowRequest.CloudProvider
		blobStorageS3Config = event.CommonWorkflowRequest.BlobStorageS3Config
		azureBlobConfig = event.CommonWorkflowRequest.AzureBlobConfig
		gcpBlobConfig = event.CommonWorkflowRequest.GcpBlobConfig
		inAppLoggingEnabled = event.CommonWorkflowRequest.InAppLoggingEnabled
	}

	cloudHelperConfig := &util.CloudHelperBaseConfig{
		StorageModuleConfigured: storageModuleConfigured,
		BlobStorageLogKey:       blobStorageLogKey,
		CloudProvider:           cloudProvider,
		UseExternalClusterBlob:  event.CommonWorkflowRequest.UseExternalClusterBlob,
		BlobStorageS3Config:     blobStorageS3Config,
		AzureBlobConfig:         azureBlobConfig,
		GcpBlobConfig:           gcpBlobConfig,
		BlobStorageObjectType:   util.BlobStorageObjectTypeLog,
	}
	if r := recover(); r != nil {
		fmt.Println(r, string(debug.Stack()))
		*exitCode = 1
	}
	log.Println(util.DEVTRON, " blob storage configured ", storageModuleConfigured)
	log.Println(util.DEVTRON, " in app logging enabled ", inAppLoggingEnabled)
	if inAppLoggingEnabled {
		helper.UploadLogs(cloudHelperConfig)
	} else {
		log.Println(util.DEVTRON, "not uploading logs from app")
	}
}

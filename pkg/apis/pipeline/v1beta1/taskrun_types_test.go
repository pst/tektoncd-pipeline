/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test/diff"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestTaskRun_GetPipelineRunPVCName(t *testing.T) {
	tests := []struct {
		name            string
		tr              *v1beta1.TaskRun
		expectedPVCName string
	}{{
		name: "invalid owner reference",
		tr: &v1beta1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{{
					Kind: "SomeOtherOwner",
					Name: "testpr",
				}},
			},
		},
		expectedPVCName: "",
	}, {
		name: "valid pipelinerun owner",
		tr: &v1beta1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{{
					Kind: "PipelineRun",
					Name: "testpr",
				}},
			},
		},
		expectedPVCName: "testpr-pvc",
	}, {
		name:            "nil taskrun",
		expectedPVCName: "",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tr.GetPipelineRunPVCName() != tt.expectedPVCName {
				t.Fatalf("taskrun pipeline run pvc name mismatch: got %s ; expected %s", tt.tr.GetPipelineRunPVCName(), tt.expectedPVCName)
			}
		})
	}
}

func TestTaskRun_HasPipelineRun(t *testing.T) {
	tests := []struct {
		name string
		tr   *v1beta1.TaskRun
		want bool
	}{{
		name: "invalid owner reference",
		tr: &v1beta1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{{
					Kind: "SomeOtherOwner",
					Name: "testpr",
				}},
			},
		},
		want: false,
	}, {
		name: "valid pipelinerun owner",
		tr: &v1beta1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{{
					Kind: "PipelineRun",
					Name: "testpr",
				}},
			},
		},
		want: true,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tr.HasPipelineRunOwnerReference() != tt.want {
				t.Fatalf("taskrun pipeline run pvc name mismatch: got %s ; expected %t", tt.tr.GetPipelineRunPVCName(), tt.want)
			}
		})
	}
}

func TestTaskRunIsDone(t *testing.T) {
	tr := &v1beta1.TaskRun{
		Status: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionFalse,
				}},
			},
		},
	}
	if !tr.IsDone() {
		t.Fatal("Expected taskrun status to be done")
	}
}

func TestTaskRunIsCancelled(t *testing.T) {
	tr := &v1beta1.TaskRun{
		Spec: v1beta1.TaskRunSpec{
			Status: v1beta1.TaskRunSpecStatusCancelled,
		},
	}
	if !tr.IsCancelled() {
		t.Fatal("Expected taskrun status to be cancelled")
	}
}

func TestTaskRunIsCancelledWithMessage(t *testing.T) {
	expectedStatusMessage := "test message"
	tr := &v1beta1.TaskRun{
		Spec: v1beta1.TaskRunSpec{
			Status:        v1beta1.TaskRunSpecStatusCancelled,
			StatusMessage: v1beta1.TaskRunSpecStatusMessage(expectedStatusMessage),
		},
	}
	if !tr.IsCancelled() {
		t.Fatal("Expected pipelinerun status to be cancelled")
	}

	if string(tr.Spec.StatusMessage) != expectedStatusMessage {
		t.Fatalf("Expected StatusMessage is %s but got %s", v1beta1.TaskRunCancelledByPipelineMsg, tr.Spec.StatusMessage)
	}
}

func TestTaskRunIsTaskRunResultVerified(t *testing.T) {
	tr := &v1beta1.TaskRun{
		Status: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionType(v1beta1.TaskRunConditionResultsVerified.String()),
					Status:  corev1.ConditionTrue,
					Reason:  v1beta1.TaskRunReasonResultsVerified.String(),
					Message: "Successfully verified all spire signed taskrun results",
				}},
			},
		},
	}
	if !tr.IsTaskRunResultVerified() {
		t.Fatal("Expected pipelinerun status to be results verified")
	}
	if tr.Status.GetCondition(apis.ConditionType(v1beta1.TaskRunConditionResultsVerified.String())).Reason != v1beta1.TaskRunReasonResultsVerified.String() {
		t.Fatal("Expected pipelinerun status reason to be TaskRunResultsVerified")
	}
}

func TestTaskRunEmptyIsTaskRunResultVerified(t *testing.T) {
	tr := &v1beta1.TaskRun{
		Status: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{},
		},
	}
	if tr.IsTaskRunResultVerified() {
		t.Fatal("Expected false as no condition exists for SignedResultsVerified")
	}
}

func TestTaskRunIsTaskRunResultDone(t *testing.T) {
	tr := &v1beta1.TaskRun{
		Status: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionType(v1beta1.TaskRunConditionResultsVerified.String()),
					Status:  corev1.ConditionUnknown,
					Reason:  v1beta1.AwaitingTaskRunResults.String(),
					Message: "Waiting upon TaskRun results and signatures to verify",
				}},
			},
		},
	}
	if tr.IsTaskRunResultDone() {
		t.Fatal("Expected pipelinerun status to be unknown and waiting")
	}
	if tr.Status.GetCondition(apis.ConditionType(v1beta1.TaskRunConditionResultsVerified.String())).Reason != v1beta1.AwaitingTaskRunResults.String() {
		t.Fatal("Expected pipelinerun status reason to be AwaitingTaskRunResults")
	}
}

func TestTaskRunEmptyIsTaskRunResultDone(t *testing.T) {
	tr := &v1beta1.TaskRun{
		Status: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{},
		},
	}
	if tr.IsTaskRunResultDone() {
		t.Fatal("Expected false as no condition exists for SignedResultsVerified")
	}
}

func TestTaskRunHasVolumeClaimTemplate(t *testing.T) {
	tr := &v1beta1.TaskRun{
		Spec: v1beta1.TaskRunSpec{
			Workspaces: []v1beta1.WorkspaceBinding{{
				Name: "my-workspace",
				VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pvc",
					},
					Spec: corev1.PersistentVolumeClaimSpec{},
				},
			}},
		},
	}
	if !tr.HasVolumeClaimTemplate() {
		t.Fatal("Expected taskrun to have a volumeClaimTemplate workspace")
	}
}

func TestTaskRunKey(t *testing.T) {
	tr := &v1beta1.TaskRun{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "trunname"}}
	n := tr.GetNamespacedName()
	expected := "foo/trunname"
	if n.String() != expected {
		t.Fatalf("Expected name to be %s but got %s", expected, n.String())
	}
}

func TestTaskRunHasStarted(t *testing.T) {
	params := []struct {
		name          string
		trStatus      v1beta1.TaskRunStatus
		expectedValue bool
	}{{
		name:          "trWithNoStartTime",
		trStatus:      v1beta1.TaskRunStatus{},
		expectedValue: false,
	}, {
		name: "trWithStartTime",
		trStatus: v1beta1.TaskRunStatus{
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				StartTime: &metav1.Time{Time: now},
			},
		},
		expectedValue: true,
	}, {
		name: "trWithZeroStartTime",
		trStatus: v1beta1.TaskRunStatus{
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				StartTime: &metav1.Time{},
			},
		},
		expectedValue: false,
	}}
	for _, tc := range params {
		t.Run(tc.name, func(t *testing.T) {
			tr := &v1beta1.TaskRun{}
			tr.Status = tc.trStatus
			if tr.HasStarted() != tc.expectedValue {
				t.Fatalf("Expected taskrun HasStarted() to return %t but got %t", tc.expectedValue, tr.HasStarted())
			}
		})
	}
}

func TestHasTimedOut(t *testing.T) {
	// IsZero reports whether t represents the zero time instant, January 1, year 1, 00:00:00 UTC
	zeroTime := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	testCases := []struct {
		name           string
		taskRun        *v1beta1.TaskRun
		expectedStatus bool
	}{{
		name: "TaskRun not started",
		taskRun: &v1beta1.TaskRun{
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: []apis.Condition{{
						Type:   apis.ConditionSucceeded,
						Status: corev1.ConditionFalse,
					}},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: zeroTime},
				},
			},
		},
		expectedStatus: false,
	}, {
		name: "TaskRun no timeout",
		taskRun: &v1beta1.TaskRun{
			Spec: v1beta1.TaskRunSpec{
				Timeout: &metav1.Duration{
					Duration: 0 * time.Minute,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: []apis.Condition{{
						Type:   apis.ConditionSucceeded,
						Status: corev1.ConditionFalse,
					}},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: now.Add(-15 * time.Hour)},
				},
			},
		},
		expectedStatus: false,
	}, {
		name: "TaskRun timed out",
		taskRun: &v1beta1.TaskRun{
			Spec: v1beta1.TaskRunSpec{
				Timeout: &metav1.Duration{
					Duration: 10 * time.Second,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: []apis.Condition{{
						Type:   apis.ConditionSucceeded,
						Status: corev1.ConditionFalse,
					}},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: now.Add(-15 * time.Second)},
				},
			},
		},
		expectedStatus: true,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.taskRun.HasTimedOut(context.Background(), testClock)
			if d := cmp.Diff(result, tc.expectedStatus); d != "" {
				t.Fatalf(diff.PrintWantGot(d))
			}
		})
	}
}

func TestInitializeTaskRunConditions(t *testing.T) {
	tr := &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-name",
			Namespace: "test-ns",
		},
	}
	tr.Status.InitializeConditions()

	if tr.Status.StartTime.IsZero() {
		t.Fatalf("TaskRun StartTime not initialized correctly")
	}

	condition := tr.Status.GetCondition(apis.ConditionSucceeded)
	if condition.Reason != v1beta1.TaskRunReasonStarted.String() {
		t.Fatalf("TaskRun initialize reason should be %s, got %s instead", v1beta1.TaskRunReasonStarted.String(), condition.Reason)
	}

	// Change the reason before we initialize again
	tr.Status.SetCondition(&apis.Condition{
		Type:    apis.ConditionSucceeded,
		Status:  corev1.ConditionUnknown,
		Reason:  "not just started",
		Message: "hello",
	})

	tr.Status.InitializeConditions()

	newCondition := tr.Status.GetCondition(apis.ConditionSucceeded)
	if newCondition.Reason != "not just started" {
		t.Fatalf("PipelineRun initialize reset the condition reason to %s", newCondition.Reason)
	}
}

/*
Copyright 2023 The Kubernetes Authors.

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

package app

import (
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"k8s.io/apimachinery/pkg/util/sets"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	cpnames "k8s.io/cloud-provider/names"
	"k8s.io/component-base/featuregate"
	featuregatetesting "k8s.io/component-base/featuregate/testing"
	"k8s.io/kubernetes/cmd/kube-controller-manager/names"
	"k8s.io/kubernetes/pkg/features"
	"k8s.io/utils/strings/slices"
)

func TestControllerNamesConsistency(t *testing.T) {
	controllerNameRegexp := regexp.MustCompile("^[a-z]([-a-z]*[a-z])?$")

	for _, name := range KnownControllers() {
		if !controllerNameRegexp.MatchString(name) {
			t.Errorf("name consistency check failed: controller %q must consist of lower case alphabetic characters or '-', and must start and end with an alphabetic character", name)
		}
		if !strings.HasSuffix(name, "-controller") {
			t.Errorf("name consistency check failed: controller %q must have \"-controller\" suffix", name)
		}
	}
}

func TestControllerNamesDeclaration(t *testing.T) {
	declaredControllers := sets.NewString(
		names.ServiceAccountTokenController,
		names.EndpointsController,
		names.EndpointSliceController,
		names.EndpointSliceMirroringController,
		names.ReplicationControllerController,
		names.PodGarbageCollectorController,
		names.ResourceQuotaController,
		names.NamespaceController,
		names.ServiceAccountController,
		names.GarbageCollectorController,
		names.DaemonSetController,
		names.JobController,
		names.DeploymentController,
		names.ReplicaSetController,
		names.HorizontalPodAutoscalerController,
		names.DisruptionController,
		names.StatefulSetController,
		names.CronJobController,
		names.CertificateSigningRequestSigningController,
		names.CertificateSigningRequestApprovingController,
		names.CertificateSigningRequestCleanerController,
		names.TTLController,
		names.BootstrapSignerController,
		names.TokenCleanerController,
		names.NodeIpamController,
		names.NodeLifecycleController,
		names.TaintEvictionController,
		cpnames.ServiceLBController,
		cpnames.NodeRouteController,
		cpnames.CloudNodeLifecycleController,
		names.PersistentVolumeBinderController,
		names.PersistentVolumeAttachDetachController,
		names.PersistentVolumeExpanderController,
		names.ClusterRoleAggregationController,
		names.PersistentVolumeClaimProtectionController,
		names.PersistentVolumeProtectionController,
		names.TTLAfterFinishedController,
		names.RootCACertificatePublisherController,
		names.EphemeralVolumeController,
		names.StorageVersionGarbageCollectorController,
		names.ResourceClaimController,
		names.LegacyServiceAccountTokenCleanerController,
		names.ValidatingAdmissionPolicyStatusController,
	)

	for _, name := range KnownControllers() {
		if !declaredControllers.Has(name) {
			t.Errorf("name declaration check failed: controller name %q should be declared in  \"controller_names.go\" and added to this test", name)
		}
	}
}

func TestNewControllerDescriptorsShouldNotPanic(t *testing.T) {
	NewControllerDescriptors()
}

func TestNewControllerDescriptorsAlwaysReturnsDescriptorsForAllControllers(t *testing.T) {
	controllersWithoutFeatureGates := KnownControllers()

	defer featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, "AllAlpha", true)()
	defer featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, "AllBeta", true)()

	controllersWithFeatureGates := KnownControllers()

	if diff := cmp.Diff(controllersWithoutFeatureGates, controllersWithFeatureGates); diff != "" {
		t.Errorf("unexpected controllers after enabling feature gates, NewControllerDescriptors should always return all controller descriptors. Controllers should define required feature gates in ControllerDescriptor.requiredFeatureGates. Diff of returned controllers:\n%s", diff)
	}
}

func TestFeatureGatedControllersShouldNotDefineAliases(t *testing.T) {
	featureGateRegex := regexp.MustCompile("^([a-zA-Z0-9]+)")

	alphaFeatures := sets.NewString()
	for _, featureText := range utilfeature.DefaultFeatureGate.KnownFeatures() {
		// we have to parse this from KnownFeatures, because usage of mutable FeatureGate is not allowed in unit tests
		feature := featureGateRegex.FindString(featureText)
		if strings.Contains(featureText, string(featuregate.Alpha)) && feature != "AllAlpha" {
			alphaFeatures.Insert(feature)
		}

	}

	for name, controller := range NewControllerDescriptors() {
		if len(controller.GetAliases()) == 0 {
			continue
		}

		requiredFeatureGates := controller.GetRequiredFeatureGates()
		if len(requiredFeatureGates) == 0 {
			continue
		}

		// DO NOT ADD any new controllers here. These two controllers are an exception, because they were added before this test was introduced
		if name == names.LegacyServiceAccountTokenCleanerController || name == names.ResourceClaimController {
			continue
		}

		areAllRequiredFeaturesAlpha := true
		for _, feature := range requiredFeatureGates {
			if !alphaFeatures.Has(string(feature)) {
				areAllRequiredFeaturesAlpha = false
				break
			}
		}

		if areAllRequiredFeaturesAlpha {
			t.Errorf("alias check failed: controller name %q should not be aliased as it is still guarded by alpha feature gates (%v) and thus should have only a canonical name", name, requiredFeatureGates)
		}
	}
}

// TestTaintEvictionControllerDeclaration ensures that it is possible to run taint-manager as a separated controller
// only when the SeparateTaintEvictionController feature is enabled
func TestTaintEvictionControllerDeclaration(t *testing.T) {
	defer featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.SeparateTaintEvictionController, true)()
	if !slices.Contains(KnownControllers(), names.TaintEvictionController) {
		t.Errorf("TaintEvictionController should be a registered controller when the SeparateTaintEvictionController feature is enabled")
	}

	defer featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.SeparateTaintEvictionController, false)()
	if slices.Contains(KnownControllers(), names.TaintEvictionController) {
		t.Errorf("TaintEvictionController should not be a registered controller when the SeparateTaintEvictionController feature is disabled")
	}
}

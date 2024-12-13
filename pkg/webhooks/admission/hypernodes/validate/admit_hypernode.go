/*
Copyright 2024 The Volcano Authors.

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

package validate

import (
	"context"
	"fmt"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	whv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	hypernodev1alpha1 "volcano.sh/apis/pkg/apis/topology/v1alpha1"
	"volcano.sh/volcano/pkg/webhooks/router"
	"volcano.sh/volcano/pkg/webhooks/schema"
	"volcano.sh/volcano/pkg/webhooks/util"
)

const (
	HyperNodeLabel = "volcano.sh/hypernodes"
)

var config = &router.AdmissionServiceConfig{}

var service = &router.AdmissionService{
	Path: "/hypernodes/validate",
	Func: AdmitHyperNode,

	Config: config,

	ValidatingConfig: &whv1.ValidatingWebhookConfiguration{
		Webhooks: []whv1.ValidatingWebhook{{
			Name: "validatehypernode.volcano.sh",
			Rules: []whv1.RuleWithOperations{
				{
					Operations: []whv1.OperationType{whv1.Create, whv1.Update},
					Rule: whv1.Rule{
						APIGroups:   []string{hypernodev1alpha1.SchemeGroupVersion.Group},
						APIVersions: []string{hypernodev1alpha1.SchemeGroupVersion.Version},
						Resources:   []string{"hypernodes"},
					},
				},
			},
		}},
	},
}

func init() {
	router.RegisterAdmission(service)
}

// AdmitHyperNode is to admit hypernode and return response.
// Reference: https://github.com/volcano-sh/volcano/issues/3883
func AdmitHyperNode(ar admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	klog.V(3).Infof("admitting hypernode -- %s", ar.Request.Operation)

	hypernode, err := schema.DecodeHyperNode(ar.Request.Object, ar.Request.Resource)
	if err != nil {
		return util.ToAdmissionResponse(err)
	}

	switch ar.Request.Operation {
	case admissionv1.Create:
		err = validateHyperNodeCreate(hypernode)
		if err != nil {
			return util.ToAdmissionResponse(err)
		}

	case admissionv1.Update:
		oldHyperNode, err := schema.DecodeHyperNode(ar.Request.OldObject, ar.Request.Resource)
		if err != nil {
			return util.ToAdmissionResponse(err)
		}
		err = validateHyperNodeUpdate(oldHyperNode, hypernode)
		if err != nil {
			return util.ToAdmissionResponse(err)
		}
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}

// validateHyperNodeCreate is to validate hypernode create
func validateHyperNodeCreate(hypernode *hypernodev1alpha1.HyperNode) error {
	if err := validateHyperNodeMembers(hypernode); err != nil {
		return err
	}

	if hypernode.Labels != nil && hypernode.Labels[HyperNodeLabel] != "" {
		hypernodeList := strings.Split(hypernode.Labels[HyperNodeLabel], ",")

		for _, hypernodeName := range hypernodeList {
			if !strings.HasPrefix(hypernodeName, "hypernode-") {
				return fmt.Errorf("the label %s must be like `hypernode-0,hypernode-1,...,hypernode-n`", HyperNodeLabel)
			}

			if _, err := config.VolcanoClient.TopologyV1alpha1().HyperNodes().Get(context.TODO(), hypernodeName, metav1.GetOptions{}); err != nil {
				return fmt.Errorf("failed to get hypernode %s: %v", hypernodeName, err)
			}
		}
	}

	return nil
}

// validateHyperNodeUpdate is to validate hypernode update
func validateHyperNodeUpdate(oldHyperNode, hypernode *hypernodev1alpha1.HyperNode) error {
	if err := validateHyperNodeMembers(hypernode); err != nil {
		return err
	}

	var oldHyperNodeList []string
	var newHyperNodeList []string

	if oldHyperNode.Labels != nil && oldHyperNode.Labels[HyperNodeLabel] != "" {
		oldHyperNodeList = strings.Split(oldHyperNode.Labels[HyperNodeLabel], ",")
	}

	if hypernode.Labels != nil && hypernode.Labels[HyperNodeLabel] != "" {
		newHyperNodeList = strings.Split(hypernode.Labels[HyperNodeLabel], ",")
	}

	// set hypernode list to empty is ok
	if len(newHyperNodeList) == 0 {
		return nil
	}

	// change hypernode list length is not allowed
	if len(newHyperNodeList) != len(oldHyperNodeList) {
		return fmt.Errorf("change hypernode list length is not allowed")
	}

	// get changed hypernode and validate
	for i := range newHyperNodeList {
		if newHyperNodeList[i] != oldHyperNodeList[i] {
			if !strings.HasPrefix(newHyperNodeList[i], "hypernode-") {
				return fmt.Errorf("the label %s must be like `hypernode-0,hypernode-1,...,hypernode-n`", HyperNodeLabel)
			}

			hypernode, err := config.VolcanoClient.TopologyV1alpha1().HyperNodes().Get(context.TODO(), newHyperNodeList[i], metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get hypernode %s: %v", newHyperNodeList[i], err)
			}

			if hypernode.Spec.Tier != "1" {
				return fmt.Errorf("changed hypernode %s tier is not tier 1 is not allowed", newHyperNodeList[i])
			}
		}
	}

	return nil
}

// validateHyperNodeMembers is to validate hypernode members
func validateHyperNodeMembers(hypernode *hypernodev1alpha1.HyperNode) error {
	if len(hypernode.Spec.Members) == 0 {
		return nil
	}

	for _, member := range hypernode.Spec.Members {
		if member.Selector == (hypernodev1alpha1.MemberSelector{}) {
			continue
		}

		if member.Selector.ExactMatch != nil && member.Selector.RegexMatch != nil {
			return fmt.Errorf("exactMatch and regexMatch cannot be specified together")
		}
	}
	return nil
}

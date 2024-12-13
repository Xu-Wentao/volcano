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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	hypernodev1alpha1 "volcano.sh/apis/pkg/apis/topology/v1alpha1"
	fakeclient "volcano.sh/apis/pkg/client/clientset/versioned/fake"
)

func TestValidateHyperNodeMembers(t *testing.T) {
	testCases := []struct {
		Name      string
		HyperNode *hypernodev1alpha1.HyperNode
		ExpectErr string
	}{
		{
			Name: "validate valid hypernode",
			HyperNode: &hypernodev1alpha1.HyperNode{
				Spec: hypernodev1alpha1.HyperNodeSpec{
					Members: []hypernodev1alpha1.MemberSpec{
						{
							Selector: hypernodev1alpha1.MemberSelector{
								ExactMatch: &hypernodev1alpha1.ExactMatch{
									Name: "node-1",
								},
							},
						},
					},
				},
			},
			ExpectErr: "",
		},
		{
			Name: "validate valid hypernode with empty selector",
			HyperNode: &hypernodev1alpha1.HyperNode{
				Spec: hypernodev1alpha1.HyperNodeSpec{
					Members: []hypernodev1alpha1.MemberSpec{
						{
							Selector: hypernodev1alpha1.MemberSelector{},
						},
					},
				},
			},
			ExpectErr: "",
		},
		{
			Name: "validate invalid hypernode with both exactMatch and regexMatch",
			HyperNode: &hypernodev1alpha1.HyperNode{
				Spec: hypernodev1alpha1.HyperNodeSpec{
					Members: []hypernodev1alpha1.MemberSpec{
						{
							Selector: hypernodev1alpha1.MemberSelector{
								ExactMatch: &hypernodev1alpha1.ExactMatch{
									Name: "node-1",
								},
								RegexMatch: &hypernodev1alpha1.RegexMatch{
									Pattern: "node-1",
								},
							},
						},
					},
				},
			},
			ExpectErr: "exactMatch and regexMatch cannot be specified together",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			err := validateHyperNodeMembers(testCase.HyperNode)
			if err != nil && err.Error() != testCase.ExpectErr {
				t.Errorf("validateHyperNodeLabels failed: %v", err)
			}
		})
	}

}

// TestValidateHyperNodeLabels tests the validateHyperNodeLabels function
func TestValidateCreatedHyperNodeLabels(t *testing.T) {
	testCases := []struct {
		Name      string
		HyperNode *hypernodev1alpha1.HyperNode
		ExpectErr string
	}{
		{
			Name: "validate valid hypernode labels",
			HyperNode: &hypernodev1alpha1.HyperNode{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"volcano.sh/hypernodes": "hypernode-0,hypernode-1",
					},
				},
			},
			ExpectErr: "",
		},
		{
			Name: "validate invalid hypernode labels with name of non-hypernode",
			HyperNode: &hypernodev1alpha1.HyperNode{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"volcano.sh/hypernodes": "non-hypernode-0,non-hypernode-1",
					},
				},
			},
			ExpectErr: "the label volcano.sh/hypernodes must be like `hypernode-0,hypernode-1,...,hypernode-n`",
		},
		{
			Name: "validate invalid hypernode labels with not exist hypernode",
			HyperNode: &hypernodev1alpha1.HyperNode{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"volcano.sh/hypernodes": "hypernode-3,hypernode-4",
					},
				},
			},
			ExpectErr: "failed to get hypernode hypernode-3: hypernodes.topology.volcano.sh \"hypernode-3\" not found",
		},
	}

	// create hyper node for test
	hypernodeList := &hypernodev1alpha1.HyperNodeList{
		Items: []hypernodev1alpha1.HyperNode{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "hypernode-0",
				},
				Spec: hypernodev1alpha1.HyperNodeSpec{
					Tier: "1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "hypernode-1",
				},
				Spec: hypernodev1alpha1.HyperNodeSpec{
					Tier: "2",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "hypernode-2",
				},
				Spec: hypernodev1alpha1.HyperNodeSpec{
					Tier: "3",
				},
			},
		},
	}

	client := fakeclient.NewSimpleClientset(hypernodeList)
	config.VolcanoClient = client

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			err := validateHyperNodeCreate(testCase.HyperNode)
			if err != nil && err.Error() != testCase.ExpectErr {
				t.Errorf("validateHyperNodeLabels failed: %v", err)
			}
		})
	}
}

func TestValidateUpdateHyperNodeLabels(t *testing.T) {
	testCases := []struct {
		Name      string
		HyperNode *hypernodev1alpha1.HyperNode
		ExpectErr string
	}{
		{
			Name: "validate valid hypernode labels",
			HyperNode: &hypernodev1alpha1.HyperNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "hypernode-0",
					Labels: map[string]string{
						"volcano.sh/hypernodes": "hypernode-0-new,hypernode-1",
					},
				},
			},
			ExpectErr: "",
		},
		{
			Name: "validate hypernode labels with set to empty",
			HyperNode: &hypernodev1alpha1.HyperNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "hypernode-0",
					Labels: map[string]string{
						"volcano.sh/hypernodes": "",
					},
				},
			},
			ExpectErr: "",
		},
		{
			Name: "validate invalid hypernode selector with delete one hypernode",
			HyperNode: &hypernodev1alpha1.HyperNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "hypernode-0",
					Labels: map[string]string{
						"volcano.sh/hypernodes": "hypernode-1",
					},
				},
			},
			ExpectErr: "change hypernode list length is not allowed",
		},
		{
			Name: "validate invalid hypernode selector with add one hypernode",
			HyperNode: &hypernodev1alpha1.HyperNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "hypernode-0",
					Labels: map[string]string{
						"volcano.sh/hypernodes": "hypernode-0,hypernode-1,hypernode-2",
					},
				},
			},
			ExpectErr: "change hypernode list length is not allowed",
		},
		{
			Name: "validate invalid hypernode selector change hypernode not tier 1",
			HyperNode: &hypernodev1alpha1.HyperNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "hypernode-0",
					Labels: map[string]string{
						"volcano.sh/hypernodes": "hypernode-0,hypernode-3",
					},
				},
			},
			ExpectErr: "changed hypernode hypernode-3 tier is not tier 1 is not allowed",
		},
		{
			Name: "validate invalid hypernode selector change hypernode with not exist hypernode",
			HyperNode: &hypernodev1alpha1.HyperNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "hypernode-0",
					Labels: map[string]string{
						"volcano.sh/hypernodes": "hypernode-0,hypernode-2",
					},
				},
			},
			ExpectErr: "failed to get hypernode hypernode-2: hypernodes.topology.volcano.sh \"hypernode-2\" not found",
		},
	}

	// create hyper node for test
	oldHyperNode := &hypernodev1alpha1.HyperNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hypernode-0",
			Labels: map[string]string{
				"volcano.sh/hypernodes": "hypernode-0,hypernode-1",
			},
		},
		Spec: hypernodev1alpha1.HyperNodeSpec{
			Tier: "1",
		},
	}

	hyperNode1 := &hypernodev1alpha1.HyperNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hypernode-1",
		},
		Spec: hypernodev1alpha1.HyperNodeSpec{
			Tier: "2",
		},
	}

	hyperNodeNew := &hypernodev1alpha1.HyperNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hypernode-0-new",
		},
		Spec: hypernodev1alpha1.HyperNodeSpec{
			Tier: "1",
		},
	}

	hyperNode3 := &hypernodev1alpha1.HyperNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hypernode-3",
		},
		Spec: hypernodev1alpha1.HyperNodeSpec{
			Tier: "3",
		},
	}

	client := fakeclient.NewSimpleClientset(oldHyperNode, hyperNode1, hyperNodeNew, hyperNode3)
	config.VolcanoClient = client

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			err := validateHyperNodeUpdate(oldHyperNode, testCase.HyperNode)
			if err != nil && err.Error() != testCase.ExpectErr {
				t.Errorf("validateHyperNodeLabels failed: %v", err)
			}
		})
	}
}

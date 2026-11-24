/*
Portions Copyright (c) Microsoft Corporation.

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

package launchtemplate

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"
	"sigs.k8s.io/karpenter/pkg/scheduling"

	"github.com/Azure/karpenter-provider-azure/pkg/apis/v1beta1"
	"github.com/Azure/karpenter-provider-azure/pkg/operator/options"
)

func TestGetOpenShiftTemplatePreservesOSDiskSize(t *testing.T) {
	g := NewWithT(t)

	const (
		subnetID = "/subscriptions/subscription/resourceGroups/resource-group/providers/Microsoft.Network/virtualNetworks/vnet/subnets/subnet"
		userData = `{"ignition":{"version":"3.2.0"}}`
		imageURN = "azureopenshift:aro4:aro_422-v2:9.8.20260428"
	)
	ctx := options.ToContext(context.Background(), &options.Options{SubnetID: subnetID})
	nodeClass := &v1beta1.AKSNodeClass{
		Spec: v1beta1.AKSNodeClassSpec{
			OSDiskSizeGB:      lo.ToPtr[int32](120),
			MarketplaceImages: []string{imageURN},
			UserData:          lo.ToPtr(userData),
		},
	}
	nodeClass.Status.Images = []v1beta1.NodeImage{{
		ID: imageURN,
		Requirements: []corev1.NodeSelectorRequirement{
			{Key: corev1.LabelArchStable, Operator: corev1.NodeSelectorOpIn, Values: []string{"amd64"}},
			{Key: v1beta1.LabelSKUHyperVGeneration, Operator: corev1.NodeSelectorOpIn, Values: []string{"2"}},
		},
	}}
	nodeClass.StatusConditions().SetTrue(v1beta1.ConditionTypeImagesReady)
	instanceType := &cloudprovider.InstanceType{Requirements: scheduling.NewRequirements(
		scheduling.NewRequirement(corev1.LabelArchStable, corev1.NodeSelectorOpIn, "amd64"),
		scheduling.NewRequirement(v1beta1.LabelSKUHyperVGeneration, corev1.NodeSelectorOpIn, "2"),
	)}

	template, err := (&Provider{}).getOpenShiftTemplate(ctx, nodeClass, instanceType)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(template.SubnetID).To(Equal(subnetID))
	g.Expect(template.StorageProfileSizeGB).To(Equal(int32(120)))
	g.Expect(template.MarketplaceImage).To(Equal(&v1beta1.MarketplaceImage{
		Publisher: "azureopenshift",
		Offer:     "aro4",
		SKU:       "aro_422-v2",
		Version:   "9.8.20260428",
	}))
	g.Expect(template.OpenShiftUserData).To(Equal(userData))
}

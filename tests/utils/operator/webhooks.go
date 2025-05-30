/*
Copyright © contributors to CloudNativePG, established as
CloudNativePG a Series of LF Projects, LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

SPDX-License-Identifier: Apache-2.0
*/

package operator

import (
	"bytes"
	"context"
	"fmt"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudnative-pg/cloudnative-pg/internal/cmd/manager/controller"
	"github.com/cloudnative-pg/cloudnative-pg/tests/utils/objects"
)

// GetMutatingWebhookByName get the MutatingWebhook filtered by the name of one
// of the webhooks
func GetMutatingWebhookByName(
	ctx context.Context,
	crudClient client.Client,
	name string,
) (
	*admissionregistrationv1.MutatingWebhookConfiguration, int, error,
) {
	var mWebhooks admissionregistrationv1.MutatingWebhookConfigurationList
	err := objects.List(ctx, crudClient, &mWebhooks)
	if err != nil {
		return nil, 0, err
	}

	for i, item := range mWebhooks.Items {
		for i2, webhook := range item.Webhooks {
			if webhook.Name == name {
				return &mWebhooks.Items[i], i2, nil
			}
		}
	}
	return nil, 0, fmt.Errorf("mutating webhook not found")
}

// UpdateMutatingWebhookConf update MutatingWebhookConfiguration object
func UpdateMutatingWebhookConf(
	ctx context.Context,
	kubeInterface kubernetes.Interface,
	wh *admissionregistrationv1.MutatingWebhookConfiguration,
) error {
	_, err := kubeInterface.AdmissionregistrationV1().
		MutatingWebhookConfigurations().Update(ctx, wh, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// getCNPGsValidatingWebhookConf get the ValidatingWebhook linked to the operator
func getCNPGsValidatingWebhookConf(kubeInterface kubernetes.Interface) (
	*admissionregistrationv1.ValidatingWebhookConfiguration, error,
) {
	ctx := context.Background()
	validatingWebhookConfig, err := kubeInterface.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(
		ctx, controller.ValidatingWebhookConfigurationName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return validatingWebhookConfig, nil
}

// GetValidatingWebhookByName get ValidatingWebhook by the name of one
// of the webhooks
func GetValidatingWebhookByName(
	ctx context.Context,
	crudClient client.Client,
	name string,
) (
	*admissionregistrationv1.ValidatingWebhookConfiguration, int, error,
) {
	var vWebhooks admissionregistrationv1.ValidatingWebhookConfigurationList
	err := objects.List(ctx, crudClient, &vWebhooks)
	if err != nil {
		return nil, 0, err
	}

	for i, item := range vWebhooks.Items {
		for i2, webhook := range item.Webhooks {
			if webhook.Name == name {
				return &vWebhooks.Items[i], i2, nil
			}
		}
	}
	return nil, 0, fmt.Errorf("validating webhook not found")
}

// UpdateValidatingWebhookConf update the ValidatingWebhook object
func UpdateValidatingWebhookConf(
	ctx context.Context,
	kubeInterface kubernetes.Interface,
	wh *admissionregistrationv1.ValidatingWebhookConfiguration,
) error {
	_, err := kubeInterface.AdmissionregistrationV1().
		ValidatingWebhookConfigurations().Update(ctx, wh, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// checkWebhookReady ensures that the operator has finished the webhook setup.
func checkWebhookReady(
	ctx context.Context,
	crudClient client.Client,
	kubeInterface kubernetes.Interface,
	namespace string,
) error {
	// Check CA
	secret := &corev1.Secret{}
	secretNamespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      controller.WebhookSecretName,
	}
	err := objects.Get(ctx, crudClient, secretNamespacedName, secret)
	if err != nil {
		return err
	}

	ca := secret.Data["tls.crt"]

	mutatingWebhookConfig, err := getCNPGsMutatingWebhookConf(ctx, kubeInterface)
	if err != nil {
		return err
	}

	for _, webhook := range mutatingWebhookConfig.Webhooks {
		if !bytes.Equal(webhook.ClientConfig.CABundle, ca) {
			return fmt.Errorf("secret %+v not match with ca bundle in %v: %v is not equal to %v",
				controller.MutatingWebhookConfigurationName, secret, string(ca), string(webhook.ClientConfig.CABundle))
		}
	}

	validatingWebhookConfig, err := getCNPGsValidatingWebhookConf(kubeInterface)
	if err != nil {
		return err
	}

	for _, webhook := range validatingWebhookConfig.Webhooks {
		if !bytes.Equal(webhook.ClientConfig.CABundle, ca) {
			return fmt.Errorf("secret not match with ca bundle in %v",
				controller.ValidatingWebhookConfigurationName)
		}
	}

	return nil
}

// getCNPGsMutatingWebhookConf get the MutatingWebhook linked to the operator
func getCNPGsMutatingWebhookConf(
	ctx context.Context,
	kubeInterface kubernetes.Interface,
) (
	*admissionregistrationv1.MutatingWebhookConfiguration, error,
) {
	return kubeInterface.AdmissionregistrationV1().
		MutatingWebhookConfigurations().
		Get(ctx, controller.MutatingWebhookConfigurationName, metav1.GetOptions{})
}

/*
Copyright 2019 The OpenEBS Authors.

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

package k8s

import (
	"github.com/ghodss/yaml"
	config "github.com/openebs/velero-plugin/tests/config"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	// for GCP
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// KubeClient interface for k8s API
type KubeClient struct {
	kubernetes.Interface
}

// Client for KubeClient
var Client *KubeClient

var (
	scName string
	cfg    *rest.Config
)

func init() {
	var err error
	cfg, err = config.GetClusterConfig()
	if err != nil {
		panic(err)
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}
	Client = &KubeClient{client}
}

// CreatePVC creates the PVC from given yaml
func (k *KubeClient) CreatePVC(pvc corev1.PersistentVolumeClaim) error {
	_, err := k.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(&pvc)
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}

	_, err = k.waitForPVCBound(pvc.Name, pvc.Namespace)
	return err
}

// DeletePVC creates the PVC from given yaml
func (k *KubeClient) DeletePVC(pvc corev1.PersistentVolumeClaim) error {
	err := k.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(pvc.Name, &metav1.DeleteOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			err = nil
		}
	}

	return err
}

// CreateStorageClass creates storageClass from given yaml
func (k *KubeClient) CreateStorageClass(scYAML string) error {
	var sc storagev1.StorageClass
	if err := yaml.Unmarshal([]byte(scYAML), &sc); err != nil {
		return err
	}

	scName = sc.Name

	_, err := k.StorageV1().StorageClasses().Create(&sc)
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
		return nil
	}

	return nil
}

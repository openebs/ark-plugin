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

package openebs

import (
	"fmt"
	"time"

	v1alpha1 "github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	k8s "github.com/openebs/velero-plugin/tests/k8s"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	cVRPVLabel         = "openebs.io/persistent-volume"
	cVRLabel           = "cstorpool.openebs.io/uid"
	cstorID            = "OPENEBS_IO_CSTOR_ID"
	cstorPoolContainer = "cstor-pool"
)

func (c *ClientSet) waitForHealthyCVR(pvc *v1.PersistentVolumeClaim) error {
	for {
		if healthy := c.CheckCVRStatus(pvc.Name,
			pvc.Namespace,
			v1alpha1.CVRStatusOnline); healthy {
			break
		}
		time.Sleep(5 * time.Second)
		fmt.Printf("Waiting for %s/%s's CVR\n", pvc.Namespace, pvc.Name)
	}
	return nil
}

// CheckCVRStatus check CVR status for given PVC
func (c *ClientSet) CheckCVRStatus(pvc, ns string, status v1alpha1.CStorVolumeReplicaPhase) bool {
	var match bool

	cvrlist, err := c.getPVCCVRList(pvc, ns)
	if err != nil {
		return match
	}

	if len(cvrlist.Items) == 0 {
		return match
	}

	match = true
	for _, v := range cvrlist.Items {
		if v.Status.Phase != status {
			match = false
		}
	}

	return match
}

func (c *ClientSet) getPVCCVRList(pvc, ns string) (*v1alpha1.CStorVolumeReplicaList, error) {
	vol, err := c.getPVCVolumeName(pvc, ns)
	if err != nil {
		return nil, err
	}

	return c.OpenebsV1alpha1().
		CStorVolumeReplicas(OpenEBSNs).
		List(metav1.ListOptions{
			LabelSelector: cVRPVLabel + "=" + vol,
		})
}

func (c *ClientSet) getPVCVolumeName(pvc, ns string) (string, error) {
	o, err := k8s.Client.
		CoreV1().
		PersistentVolumeClaims(ns).
		Get(pvc, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	if len(o.Spec.VolumeName) == 0 {
		return "", errors.Errorf("Volume name is empty")
	}
	return o.Spec.VolumeName, nil
}

// CheckSnapshot check if given snapshot is created or not
func (c *ClientSet) CheckSnapshot(pvc, pvcNs, snapshot string) (bool, error) {
	var availabel bool

	podList, err := k8s.Client.CoreV1().Pods(OpenEBSNs).List(metav1.ListOptions{
		LabelSelector: string(v1alpha1.StoragePoolClaimCPK) + "=" + SPCName,
	})
	if err != nil {
		return availabel, err
	}

	cvrlist, err := c.getPVCCVRList(PVCName, pvcNs)
	if err != nil {
		return availabel, err
	}

	for _, k := range cvrlist.Items {
		for _, p := range podList.Items {
			v := getEnvValueFromName(p.Spec.Containers[0].Env, cstorID)
			if v == k.Labels[cVRLabel] {
				cmd := "zfs list -t all " +
					getPoolNameFromCVR(k) +
					"/" +
					getVolumeNameFromCVR(k) +
					"@" +
					snapshot
				_, e, err := k8s.Client.Exec(cmd, p.Name, cstorPoolContainer, p.Namespace)
				if err != nil || len(e) != 0 {
					return false, errors.Errorf("Error occurred for %v/%v@%v.. stderr:%v err:%v",
						getPoolNameFromCVR(k), getVolumeNameFromCVR(k), snapshot, e, err)
				}
				availabel = true
				continue
			}
		}
	}
	return availabel, nil
}

func getEnvValueFromName(env []v1.EnvVar, name string) string {
	for _, l := range env {
		if l.Name == name {
			return l.Value
		}
	}
	return ""
}

func getPoolNameFromCVR(k v1alpha1.CStorVolumeReplica) string {
	return "cstor-" + k.Labels[cVRLabel]
}

func getVolumeNameFromCVR(k v1alpha1.CStorVolumeReplica) string {
	return k.Labels[cVRPVLabel]
}

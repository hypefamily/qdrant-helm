package test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
)

func TestStorageClassAndPersistentVolume(t *testing.T) {
	t.Parallel()

	helmChartPath, err := filepath.Abs("../charts/qdrant")
	require.NoError(t, err)

	releaseName := "qdrant"
	namespaceName := "qdrant-" + strings.ToLower(random.UniqueId())

	options := &helm.Options{
		SetValues: map[string]string{
			"localStorage.enabled":          "true",
			"localStorage.name":             "local-storage",
			"localStorage.pvPrefix":         "local-pv-qdrant",
			"localStorage.storageSize":      "100Gi",
			"localStorage.localPath":        "/mnt",
			"localStorage.startIndex":       "0",
			"localStorage.step":             "2",
			"localStorage.workerNodePrefix": "server-",
			"replicaCount":                  "2",
		},
		KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
	}

	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, []string{"templates/localstorage.yaml"})

	var storageClass storagev1.StorageClass
	helm.UnmarshalK8SYaml(t, output, &storageClass)

	require.Equal(t, "local-storage", storageClass.Name, "StorageClass name is incorrect")
	require.Equal(t, "kubernetes.io/no-provisioner", storageClass.Provisioner, "StorageClass provisioner is incorrect")
	require.Equal(t, storagev1.VolumeBindingWaitForFirstConsumer, *storageClass.VolumeBindingMode, "StorageClass volumeBindingMode is incorrect")

	output = helm.RenderTemplate(t, options, helmChartPath, releaseName, []string{"templates/persistencevolume.yaml"})

	var pv corev1.PersistentVolume
	helm.UnmarshalK8SYaml(t, output, &pv)

	require.Equal(t, "local-pv-qdrant-worker-0", pv.Name, "PersistentVolume name is incorrect")
	require.Equal(t, "100Gi", pv.Spec.Capacity.Storage().String(), "PersistentVolume capacity is incorrect")
	require.Equal(t, corev1.ReadWriteOnce, pv.Spec.AccessModes[0], "PersistentVolume access mode is incorrect")
	require.Equal(t, "local-storage", pv.Spec.StorageClassName, "PersistentVolume storageClassName is incorrect")
	require.Equal(t, "/mnt", pv.Spec.Local.Path, "PersistentVolume local path is incorrect")

	require.NotNil(t, pv.Spec.NodeAffinity, "NodeAffinity should be set")
	require.Equal(t, "kubernetes.io/hostname", pv.Spec.NodeAffinity.Required.NodeSelectorTerms[0].MatchExpressions[0].Key, "NodeAffinity key is incorrect")
	require.Equal(t, "server-0", pv.Spec.NodeAffinity.Required.NodeSelectorTerms[0].MatchExpressions[0].Values[0], "NodeAffinity value is incorrect for worker 0")
}

/*
Copyright 2018 The Kubernetes Authors.

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

package kubeadm

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/version"
)

// ConfigData is supplied to the kubeadm config template, with values populated
// by the cluster package
type ConfigData struct {
	ClusterName       string
	KubernetesVersion string
	// The ControlPlaneEndpoint, that is the address of the external loadbalancer, if defined
	ControlPlaneEndpoint string
	// The Local API Server port
	APIBindPort int
	// The Token for TLS bootstrap
	Token string
	// DerivedConfigData is populated by Derive()
	// These auto-generated fields are available to Config templates,
	// but not meant to be set by hand
	DerivedConfigData
}

// DerivedConfigData fields are automatically derived by
// ConfigData.Derive if they are not specified / zero valued
type DerivedConfigData struct {
	// DockerStableTag is automatically derived from KubernetesVersion
	DockerStableTag string
}

// Derive automatically derives DockerStableTag if not specified
func (c *ConfigData) Derive() {
	if c.DockerStableTag == "" {
		c.DockerStableTag = strings.Replace(c.KubernetesVersion, "+", "_", -1)
	}
}

// See docs for these APIs at:
// https://godoc.org/k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm#pkg-subdirectories
// EG:
// https://godoc.org/k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1

// ConfigTemplateAlphaV2 is the kubadm config template for v1alpha2
const ConfigTemplateAlphaV2 = `# config generated by kind
apiVersion: kubeadm.k8s.io/v1alpha2
kind: MasterConfiguration
metadata:
  name: config
kubernetesVersion: {{.KubernetesVersion}}
clusterName: "{{.ClusterName}}"
# we use a well know token for TLS bootstrap
bootstrapTokens:
- token: "{{ .Token }}"
{{ if .ControlPlaneEndpoint -}}
controlPlaneEndpoint: {{ .ControlPlaneEndpoint }}
{{- end }}
# we use a well know port for making the API server discoverable inside docker network. 
# from the host machine such port will be accessible via a random local port instead.
api:
  bindPort: {{.APIBindPort}}
# we need nsswitch.conf so we use /etc/hosts
# https://github.com/kubernetes/kubernetes/issues/69195
apiServerExtraVolumes:
- name: nsswitch
  mountPath: /etc/nsswitch.conf
  hostPath: /etc/nsswitch.conf
  writeable: false
  pathType: FileOrCreate
# on docker for mac we have to expose the api server via port forward,
# so we need to ensure the cert is valid for localhost so we can talk
# to the cluster after rewriting the kubeconfig to point to localhost
apiServerCertSANs: [localhost]
kubeletConfiguration:
  baseConfig:
    # disable disk resource management by default
    # kubelet will see the host disk that the inner container runtime
    # is ultimately backed by and attempt to recover disk space.
    # we don't want that.
    imageGCHighThresholdPercent: 100
    evictionHard:
      nodefs.available: "0%"
      nodefs.inodesFree: "0%"
      imagefs.available: "0%"
controllerManagerExtraArgs:
  enable-hostpath-provisioner: "true"
nodeRegistration:
  criSocket: "/run/containerd/containerd.sock"
---
apiVersion: kubeadm.k8s.io/v1alpha2
kind: NodeConfiguration
metadata:
  name: config
nodeRegistration:
  criSocket: "/run/containerd/containerd.sock"
`

// ConfigTemplateAlphaV3 is the kubadm config template for API version v1alpha3
const ConfigTemplateAlphaV3 = `# config generated by kind
apiVersion: kubeadm.k8s.io/v1alpha3
kind: ClusterConfiguration
metadata:
  name: config
kubernetesVersion: {{.KubernetesVersion}}
clusterName: "{{.ClusterName}}"
{{ if .ControlPlaneEndpoint -}}
controlPlaneEndpoint: {{ .ControlPlaneEndpoint }}
{{- end }}
# we need nsswitch.conf so we use /etc/hosts
# https://github.com/kubernetes/kubernetes/issues/69195
apiServerExtraVolumes:
- name: nsswitch
  mountPath: /etc/nsswitch.conf
  hostPath: /etc/nsswitch.conf
  writeable: false
  pathType: FileOrCreate
# on docker for mac we have to expose the api server via port forward,
# so we need to ensure the cert is valid for localhost so we can talk
# to the cluster after rewriting the kubeconfig to point to localhost
apiServerCertSANs: [localhost]
controllerManagerExtraArgs:
  enable-hostpath-provisioner: "true"
---
apiVersion: kubeadm.k8s.io/v1alpha3
kind: InitConfiguration
metadata:
  name: config
# we use a well know token for TLS bootstrap
bootstrapTokens:
- token: "{{ .Token }}"
# we use a well know port for making the API server discoverable inside docker network. 
# from the host machine such port will be accessible via a random local port instead.
apiEndpoint:
  bindPort: {{.APIBindPort}}
nodeRegistration:
  criSocket: "/run/containerd/containerd.sock"
---
# no-op entry that exists solely so it can be patched
apiVersion: kubeadm.k8s.io/v1alpha3
kind: JoinConfiguration
metadata:
  name: config
nodeRegistration:
  criSocket: "/run/containerd/containerd.sock"
---
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
metadata:
  name: config
# disable disk resource management by default
# kubelet will see the host disk that the inner container runtime
# is ultimately backed by and attempt to recover disk space. we don't want that.
imageGCHighThresholdPercent: 100
evictionHard:
  nodefs.available: "0%"
  nodefs.inodesFree: "0%"
  imagefs.available: "0%"
---
# no-op entry that exists solely so it can be patched
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
metadata:
  name: config
`

// ConfigTemplateBetaV1 is the kubadm config template for API version v1beta1
const ConfigTemplateBetaV1 = `# config generated by kind
apiVersion: kubeadm.k8s.io/v1beta1
kind: ClusterConfiguration
metadata:
  name: config
kubernetesVersion: {{.KubernetesVersion}}
clusterName: "{{.ClusterName}}"
{{ if .ControlPlaneEndpoint -}}
controlPlaneEndpoint: {{ .ControlPlaneEndpoint }}
{{- end }}
# on docker for mac we have to expose the api server via port forward,
# so we need to ensure the cert is valid for localhost so we can talk
# to the cluster after rewriting the kubeconfig to point to localhost
apiServer:
  certSANs: [localhost]
controllerManager:
  extraArgs:
    enable-hostpath-provisioner: "true"
---
apiVersion: kubeadm.k8s.io/v1beta1
kind: InitConfiguration
metadata:
  name: config
# we use a well know token for TLS bootstrap
bootstrapTokens:
- token: "{{ .Token }}"
# we use a well know port for making the API server discoverable inside docker network. 
# from the host machine such port will be accessible via a random local port instead.
localAPIEndpoint:
  bindPort: {{.APIBindPort}}
nodeRegistration:
  criSocket: "/run/containerd/containerd.sock"
---
# no-op entry that exists solely so it can be patched
apiVersion: kubeadm.k8s.io/v1beta1
kind: JoinConfiguration
metadata:
  name: config
nodeRegistration:
  criSocket: "/run/containerd/containerd.sock"
---
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
metadata:
  name: config
# disable disk resource management by default
# kubelet will see the host disk that the inner container runtime
# is ultimately backed by and attempt to recover disk space. we don't want that.
imageGCHighThresholdPercent: 100
evictionHard:
  nodefs.available: "0%"
  nodefs.inodesFree: "0%"
  imagefs.available: "0%"
---
# no-op entry that exists solely so it can be patched
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
metadata:
  name: config
`

// Config returns a kubeadm config generated from config data, in particular
// the kubernetes version
func Config(data ConfigData) (config string, err error) {
	ver, err := version.ParseGeneric(data.KubernetesVersion)
	if err != nil {
		return "", err
	}

	// assume the latest API version, then fallback if the k8s version is too low
	templateSource := ConfigTemplateBetaV1
	if ver.LessThan(version.MustParseSemantic("v1.12.0")) {
		templateSource = ConfigTemplateAlphaV2
	} else if ver.LessThan(version.MustParseSemantic("v1.13.0")) {
		templateSource = ConfigTemplateAlphaV3
	}

	t, err := template.New("kubeadm-config").Parse(templateSource)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse config template")
	}

	// derive any automatic fields if not supplied
	data.Derive()

	// execute the template
	var buff bytes.Buffer
	err = t.Execute(&buff, data)
	if err != nil {
		return "", errors.Wrap(err, "error executing config template")
	}
	return buff.String(), nil
}

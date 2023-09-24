// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License

package slurm_schema

import (
	"bytes"
	"fmt"
	kubeclusterorgv1alpha1 "github.com/kubecluster/apis/kubecluster.org/v1alpha1"
	"github.com/kubecluster/pkg/common"
	"html/template"
	"os"
	"strconv"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

var (
	initContainerTemplate = `
- name: init-slurm
  image: {{.InitContainerImage}}
  imagePullPolicy: IfNotPresent
  resources:
    limits:
      cpu: 100m
      memory: 20Mi
    requests:
      cpu: 50m
      memory: 10Mi
  command: ['sh', '-c', 'err=1;for i in $(seq {{.MaxTries}}); do if nslookup {{.MasterAddr}}; then err=0 && break; fi;echo waiting for master; sleep 2; done; exit $err']`
	onceInitContainer sync.Once
	icGenerator       *initContainerGenerator
)

type initContainerGenerator struct {
	template string
	image    string
	maxTries int
}

func getInitContainerGenerator() *initContainerGenerator {
	onceInitContainer.Do(func() {
		icGenerator = &initContainerGenerator{
			template: getInitContainerTemplateOrDefault(config.SlurmSchemaInitContainerTemplateFile),
			image:    config.SlurmSchemaInitContainerImage,
			maxTries: config.SlurmSchemaInitContainerMaxTries,
		}
	})
	return icGenerator
}

func (i *initContainerGenerator) GetInitContainer(masterAddr string) ([]corev1.Container, error) {
	var buf bytes.Buffer
	tpl, err := template.New("container").Parse(i.template)
	if err != nil {
		return nil, err
	}
	if err := tpl.Execute(&buf, struct {
		MasterAddr         string
		InitContainerImage string
		MaxTries           int
	}{
		MasterAddr:         masterAddr,
		InitContainerImage: i.image,
		MaxTries:           i.maxTries,
	}); err != nil {
		return nil, err
	}

	var result []corev1.Container
	err = yaml.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}

	//should only supply one image
	if len(result) > 1 {
		result = result[0:1]
	}
	return result, nil
}

// getInitContainerTemplateOrDefault returns the init container template file if
// it exists, or return initContainerTemplate by default.
func getInitContainerTemplateOrDefault(file string) string {
	b, err := os.ReadFile(file)
	if err == nil {
		return string(b)
	}
	return initContainerTemplate
}

func setInitContainer(
	kcluster *kubeclusterorgv1alpha1.KubeCluster,
	podTemplate *corev1.PodTemplateSpec,
	rtype kubeclusterorgv1alpha1.ReplicaType,
	index string,
	port int,
	slurmdPort int,
) error {
	g := getInitContainerGenerator()
	controllerAddress := common.GenGeneralName(kcluster.Name, SchemaReplicaTypeController, strconv.Itoa(0))
	initContainers, err := g.GetInitContainer(controllerAddress)
	if err != nil {
		return err
	}

	waitReadyCommand := fmt.Sprintf("while [ \"$(cat %s)\" != \"%s\" ]; do echo \"slurm.conf not ready\"; sleep 5; done && echo \"slurm.conf ready\"",
		fmt.Sprintf("%s/%s", SlurmConfDir, slurmConfKey), configMapReady)
	cpSlurmCmdCommand := fmt.Sprintf("if [ -d %s ] && [ -d %s ];then cp -r %s/* %s;fi", SlurmCmdDir, EmptyVolumeMountPathInInitContainer, SlurmCmdDir, EmptyVolumeMountPathInInitContainer)
	if rtype == SchemaReplicaTypeController {
		initContainers[0].Command = []string{"sh", "-c", strings.Join([]string{waitReadyCommand, cpSlurmCmdCommand}, " && ")}
	} else {
		initContainers[0].Command = []string{"sh", "-c", strings.Join([]string{waitReadyCommand, strings.Join(initContainers[0].Command, " ")}, " && ")}
	}

	//we only need to change tha last
	podTemplate.Spec.InitContainers = append(podTemplate.Spec.InitContainers,
		initContainers...)
	return nil
}

package slurm_schema

import "flag"

// Config is the global configuration for the training operator.
var config struct {
	slurmSchemaInitContainerTemplateFile string
	slurmSchemaInitContainerImage        string
	slurmSchemaInitContainerMaxTries     int
}

const (
	// slurmSchemaInitContainerImageDefault is the default image for the SlurmSchema
	// init container.
	slurmSchemaInitContainerImageDefault = "registry.cn-shanghai.aliyuncs.com/eflops-bcp/slurm-minimal:v1"
	// slurmSchemaInitContainerTemplateFileDefault is the default template file for
	// the SlurmSchema init container.
	slurmSchemaInitContainerTemplateFileDefault = "/etc/config/initContainer.yaml"
	// slurmSchemaInitContainerMaxTriesDefault is the default number of tries for the SlurmSchema init container.
	slurmSchemaInitContainerMaxTriesDefault = 100
)

func init() {
	// SlurmSchema related flags
	flag.StringVar(&config.slurmSchemaInitContainerImage, "SlurmSchema-init-container-image",
		slurmSchemaInitContainerImageDefault, "The image for SlurmSchema init container")
	flag.StringVar(&config.slurmSchemaInitContainerTemplateFile, "SlurmSchema-init-container-template-file",
		slurmSchemaInitContainerTemplateFileDefault, "The template file for SlurmSchema init container")
	flag.IntVar(&config.slurmSchemaInitContainerMaxTries, "SlurmSchema-init-container-max-tries",
		slurmSchemaInitContainerMaxTriesDefault, "The number of tries for the SlurmSchema init container")
}

package slurm_schema

import "flag"

// Config is the global configuration for the training operator.
var config struct {
	SlurmSchemaInitContainerTemplateFile string
	SlurmSchemaInitContainerImage        string
	SlurmSchemaInitContainerMaxTries     int
}

const (
	// SlurmSchemaInitContainerImageDefault is the default image for the SlurmSchema
	// init container.
	SlurmSchemaInitContainerImageDefault = "registry.cn-shanghai.aliyuncs.com/eflops-bcp/slurm-minimal:v1"
	// SlurmSchemaInitContainerTemplateFileDefault is the default template file for
	// the SlurmSchema init container.
	SlurmSchemaInitContainerTemplateFileDefault = "/etc/config/initContainer.yaml"
	// SlurmSchemaInitContainerMaxTriesDefault is the default number of tries for the SlurmSchema init container.
	SlurmSchemaInitContainerMaxTriesDefault = 100
)

func init() {
	// SlurmSchema related flags
	flag.StringVar(&config.SlurmSchemaInitContainerImage, "SlurmSchema-init-container-image",
		SlurmSchemaInitContainerImageDefault, "The image for SlurmSchema init container")
	flag.StringVar(&config.SlurmSchemaInitContainerTemplateFile, "SlurmSchema-init-container-template-file",
		SlurmSchemaInitContainerTemplateFileDefault, "The template file for SlurmSchema init container")
	flag.IntVar(&config.SlurmSchemaInitContainerMaxTries, "SlurmSchema-init-container-max-tries",
		SlurmSchemaInitContainerMaxTriesDefault, "The number of tries for the SlurmSchema init container")
}

package torque_schema

import "flag"

// Config is the global configuration for the training operator.
var config struct {
	TorqueSchemaInitContainerTemplateFile string
	TorqueSchemaInitContainerImage        string
	TorqueSchemaInitContainerMaxTries     int
}

const (
	// TorqueSchemaInitContainerImageDefault is the default image for the TorqueSchema
	// init container.
	TorqueSchemaInitContainerImageDefault = "registry.cn-shanghai.aliyuncs.com/eflops-bcp/pbs-minimal:v1"
	// TorqueSchemaInitContainerTemplateFileDefault is the default template file for
	// the TorqueSchema init container.
	TorqueSchemaInitContainerTemplateFileDefault = "/etc/config/initContainer.yaml"
	// TorqueSchemaInitContainerMaxTriesDefault is the default number of tries for the TorqueSchema init container.
	TorqueSchemaInitContainerMaxTriesDefault = 100
)

func init() {
	// TorqueSchema related flags
	flag.StringVar(&config.TorqueSchemaInitContainerImage, "TorqueSchema-init-container-image",
		TorqueSchemaInitContainerImageDefault, "The image for TorqueSchema init container")
	flag.StringVar(&config.TorqueSchemaInitContainerTemplateFile, "TorqueSchema-init-container-template-file",
		TorqueSchemaInitContainerTemplateFileDefault, "The template file for TorqueSchema init container")
	flag.IntVar(&config.TorqueSchemaInitContainerMaxTries, "TorqueSchema-init-container-max-tries",
		TorqueSchemaInitContainerMaxTriesDefault, "The number of tries for the TorqueSchema init container")
}

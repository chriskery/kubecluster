package pbspro_schema

import "flag"

// Config is the global configuration for the training operator.
var config struct {
	pbsproSchemaInitContainerTemplateFile string
	pbsproSchemaInitContainerImage        string
	pbsproSchemaInitContainerMaxTries     int
}

const (
	// pbsproSchemaInitContainerImageDefault is the default image for the pbsproSchema
	// init container.
	pbsproSchemaInitContainerImageDefault = "registry.cn-shanghai.aliyuncs.com/eflops-bcp/pbs-minimal:v1"
	// pbsproSchemaInitContainerTemplateFileDefault is the default template file for
	// the pbsproSchema init container.
	pbsproSchemaInitContainerTemplateFileDefault = "/etc/config/initContainer.yaml"
	// pbsproSchemaInitContainerMaxTriesDefault is the default number of tries for the pbsproSchema init container.
	pbsproSchemaInitContainerMaxTriesDefault = 100
)

func init() {
	// pbsproSchema related flags
	flag.StringVar(&config.pbsproSchemaInitContainerImage, "pbsproSchema-init-container-image",
		pbsproSchemaInitContainerImageDefault, "The image for pbsproSchema init container")
	flag.StringVar(&config.pbsproSchemaInitContainerTemplateFile, "pbsproSchema-init-container-template-file",
		pbsproSchemaInitContainerTemplateFileDefault, "The template file for pbsproSchema init container")
	flag.IntVar(&config.pbsproSchemaInitContainerMaxTries, "pbsproSchema-init-container-max-tries",
		pbsproSchemaInitContainerMaxTriesDefault, "The number of tries for the pbsproSchema init container")
}

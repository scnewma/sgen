package cmd

type ConfigSource struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	File *struct {
		Path string `yaml:"path"`
	} `yaml:"file"`
	Command *string `yaml:"command"`
}

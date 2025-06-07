package types

import "net"

// Config represents the structure of the YAML configuration file
type Config struct {
	IpAddresses []net.IP `yaml:"ip-addresses"`
	AWS         struct {
		Profile string `yaml:"profile"`
		Region  string `yaml:"region"`
	} `yaml:"aws"`
	Scenarios map[string]Scenario `yaml:"scenarios"`
}

type Scenario struct {
	Provider string `yaml:"provider"`
	Path     string `yaml:"path"`
}

type TerraformOptions struct {
	Allowlist              []net.IP
	AWSConfigPath          string
	AWSCredentialsPath     string
	AwsProfile             string
	AwsRegion              string
	ConfigDir              string
	Destroy                bool
	TerraformCodePath      string
	TerraformStateFilePath string
	TerraformVersion       string
}

type AnsibleInventoryOptions struct {
	ScenarioAnsiblePath string
	ScenarioStatePath   string
	TemporaryDirPath    string
}

type AnsibleOptions struct {
	AnsiblePath   string
	ConfigDir     string
	InventoryPath string
}

type TerraformOutput struct {
	Value     string `json:"value"`
	Type      string `json:"type"`
	Sensitive bool   `json:"sensitive"`
}

type ValidateConfigInputs struct {
	CliInputs       CliInputs
	ConfigDirectory string
	IpAddresses     []net.IP `yaml:"ip-addresses"`
	Config          Config
}

type CliInputs struct {
	AwsProfile string
	AwsRegion  string
}

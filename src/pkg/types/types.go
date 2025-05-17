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

type ContainerOptions struct {
	ConfigDir  string
	HomeDir    string
	AwsProfile string
	AwsRegion  string
}

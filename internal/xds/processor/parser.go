package processor

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"lb/apis/v1alpha1"
	"os"
)

// parseYaml takes in a yaml envoy config and returns a typed version
func parseYaml(file string) (*v1alpha1.EnvoyConfig, error) {
	var config v1alpha1.EnvoyConfig

	yamlFile, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("Error reading YAML file: %s\n", err)
	}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

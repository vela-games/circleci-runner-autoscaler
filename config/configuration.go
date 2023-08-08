package config

import "github.com/kelseyhightower/envconfig"

type Configuration struct {
	KubernetesScalerEnabled bool   `split_words:"true" default:"true"`
	KubernetesNamespace     string `split_words:"true" default:"circleci-runners"`
	CircleToken             string `split_words:"true" required:"true"`
	CircleResourceNamespace string `split_words:"true" required:"true"`
}

func GetConfig() (*Configuration, error) {
	var autoScalerConfig Configuration

	err := envconfig.Process("app", &autoScalerConfig)
	if err != nil {
		return nil, err
	}

	return &autoScalerConfig, nil
}

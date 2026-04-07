package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type kubeConfig struct {
	APIVersion     string        `yaml:"apiVersion"`
	Kind           string        `yaml:"kind"`
	Clusters       []kubeCluster `yaml:"clusters"`
	Users          []kubeUser    `yaml:"users"`
	Contexts       []kubeContext `yaml:"contexts"`
	CurrentContext string        `yaml:"current-context,omitempty"`
}

type kubeCluster struct {
	Name    string          `yaml:"name"`
	Cluster kubeClusterData `yaml:"cluster"`
}

type kubeClusterData struct {
	Server                string `yaml:"server"`
	TLSServerName         string `yaml:"tls-server-name,omitempty"`
	InsecureSkipTLSVerify bool   `yaml:"insecure-skip-tls-verify,omitempty"`
}

type kubeUser struct {
	Name string       `yaml:"name"`
	User kubeUserData `yaml:"user"`
}

type kubeUserData struct {
	Token string `yaml:"token,omitempty"`
}

type kubeContext struct {
	Name    string          `yaml:"name"`
	Context kubeContextData `yaml:"context"`
}

type kubeContextData struct {
	Cluster string `yaml:"cluster"`
	User    string `yaml:"user"`
}

func writeKubeconfig(path string, endpoints []ProxyEndpoint) error {
	cfg := kubeConfig{
		APIVersion: "v1",
		Kind:       "Config",
	}

	for _, ep := range endpoints {
		cfg.Clusters = append(cfg.Clusters, kubeCluster{
			Name: ep.Cluster.Name,
			Cluster: kubeClusterData{
				Server: fmt.Sprintf("http://%s", ep.LocalAddr),
			},
		})
		cfg.Users = append(cfg.Users, kubeUser{
			Name: ep.Cluster.Name,
			User: kubeUserData{Token: "unused"},
		})
		cfg.Contexts = append(cfg.Contexts, kubeContext{
			Name: ep.Cluster.Name,
			Context: kubeContextData{
				Cluster: ep.Cluster.Name,
				User:    ep.Cluster.Name,
			},
		})
	}

	if len(cfg.Contexts) > 0 {
		cfg.CurrentContext = cfg.Contexts[0].Name
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal kubeconfig: %w", err)
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".kubeconfig-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp: %w", err)
	}
	tmp.Close()

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename: %w", err)
	}

	log.Printf("wrote kubeconfig to %s (%d clusters)", path, len(endpoints))
	return nil
}

package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ferret-linux/otter/pkg/config"
)

func TestDefaultConfigValues(t *testing.T) {
	cfg := config.DefaultValues()

	assert.Equal(t, "podman", cfg.ContainerManagerType)
	assert.Equal(t, "sudo", cfg.SudoProgram)
	assert.False(t, cfg.Verbose)
	assert.Equal(t, "docker.io/library/ubuntu:26.04", cfg.DefaultContainerImage)
	assert.Equal(t, "my-box", cfg.DefaultContainerName)
}

package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPodman_CloneAsRoot_PreservesFieldsAndFlipsRoot(t *testing.T) {
	original := newPodman(podmanCommandPodman, false, "doas", true)

	cloned := original.CloneAsRoot()

	require.NotSame(t, original, cloned, "expected a fresh instance")

	clone, ok := cloned.(*Podman)
	require.True(t, ok, "expected *Podman, got %T", cloned)

	assert.True(t, clone.root, "clone should be in root mode")
	assert.Equal(t, original.command, clone.command)
	assert.Equal(t, original.sudoCommand, clone.sudoCommand)
	assert.Equal(t, original.verbose, clone.verbose)

	assert.False(t, original.root, "original should remain non-root")
}

func TestPodman_CloneAsRoot_AlreadyRootStillReturnsCopy(t *testing.T) {
	original := newPodman(podmanCommandPodman, true, "sudo", false)

	cloned := original.CloneAsRoot()

	require.NotSame(t, original, cloned)

	clone, ok := cloned.(*Podman)
	require.True(t, ok)

	assert.True(t, clone.root)
	assert.Equal(t, podmanCommandPodman, clone.command)
}

func TestDocker_CloneAsRoot_PreservesFieldsAndFlipsRoot(t *testing.T) {
	original := NewDocker(false, "doas", true)

	cloned := original.CloneAsRoot()

	require.NotSame(t, original, cloned, "expected a fresh instance")

	clone, ok := cloned.(*Docker)
	require.True(t, ok, "expected *Docker, got %T", cloned)

	assert.True(t, clone.root, "clone should be in root mode")
	assert.Equal(t, original.sudoCommand, clone.sudoCommand)
	assert.Equal(t, original.verbose, clone.verbose)

	assert.False(t, original.root, "original should remain non-root")
}

func TestDocker_CloneAsRoot_AlreadyRootStillReturnsCopy(t *testing.T) {
	original := NewDocker(true, "sudo", false)

	cloned := original.CloneAsRoot()

	require.NotSame(t, original, cloned)

	clone, ok := cloned.(*Docker)
	require.True(t, ok)

	assert.True(t, clone.root)
}

func TestNerdctl_CloneAsRoot_PreservesFieldsAndFlipsRoot(t *testing.T) {
	original := NewNerdctl(false, "doas", true)

	cloned := original.CloneAsRoot()

	require.NotSame(t, original, cloned, "expected a fresh instance")

	clone, ok := cloned.(*Nerdctl)
	require.True(t, ok, "expected *Nerdctl, got %T", cloned)

	assert.True(t, clone.root, "clone should be in root mode")
	assert.Equal(t, original.sudoCommand, clone.sudoCommand)
	assert.Equal(t, original.verbose, clone.verbose)

	assert.False(t, original.root, "original should remain non-root")
}

func TestNerdctl_CloneAsRoot_AlreadyRootStillReturnsCopy(t *testing.T) {
	original := NewNerdctl(true, "sudo", false)

	cloned := original.CloneAsRoot()

	require.NotSame(t, original, cloned)

	clone, ok := cloned.(*Nerdctl)
	require.True(t, ok)

	assert.True(t, clone.root)
}

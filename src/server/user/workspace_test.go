package user

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelfHostedWorkspaceCreationPolicy(t *testing.T) {
	t.Setenv("DEKART_CLOUD", "")
	t.Setenv("DEKART_DEFAULT_WORKSPACE_ADMIN", "admin@example.com")

	t.Run("configured admin can create while default workspace remains enabled", func(t *testing.T) {
		t.Setenv("DEKART_ALLOW_WORKSPACE_CREATION", "")

		require.True(t, CanCreateWorkspace("admin@example.com"))
		require.True(t, ShouldUseDefaultWorkspace("admin@example.com"))
	})

	t.Run("flag enables creation for authenticated users without the default workspace", func(t *testing.T) {
		t.Setenv("DEKART_ALLOW_WORKSPACE_CREATION", "1")

		require.True(t, CanCreateWorkspace("user@example.com"))
		require.False(t, ShouldUseDefaultWorkspace("user@example.com"))
	})

	t.Run("anonymous users cannot create and retain the default workspace", func(t *testing.T) {
		t.Setenv("DEKART_ALLOW_WORKSPACE_CREATION", "1")

		require.False(t, CanCreateWorkspace(UnknownEmail))
		require.True(t, ShouldUseDefaultWorkspace(UnknownEmail))
	})
}

// The Klikpeta staff check of the Data Library admin commands: only the configured instance admin, never an unset one.
func TestIsDefaultWorkspaceAdmin(t *testing.T) {
	t.Setenv("DEKART_DEFAULT_WORKSPACE_ADMIN", "admin@example.com")
	require.True(t, IsDefaultWorkspaceAdmin("admin@example.com"))
	require.False(t, IsDefaultWorkspaceAdmin("other@example.com"))
	require.False(t, IsDefaultWorkspaceAdmin(""))
	require.False(t, IsDefaultWorkspaceAdmin(UnknownEmail))

	t.Setenv("DEKART_DEFAULT_WORKSPACE_ADMIN", "")
	require.False(t, IsDefaultWorkspaceAdmin(""), "an unset setting must match nobody, not an empty email")
	require.False(t, IsDefaultWorkspaceAdmin("admin@example.com"))
}

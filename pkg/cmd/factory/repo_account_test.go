package factory

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/cli/cli/v2/git"
	"github.com/cli/cli/v2/internal/config"
	"github.com/cli/cli/v2/internal/gh"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/stretchr/testify/require"
)

func TestRepoAccountHTTPAuthentication(t *testing.T) {
	cfg, _ := config.NewIsolatedTestConfig(t, `hosts:
  github.com:
    user: active
    oauth_token: active-token
    users:
      active:
        oauth_token: active-token
      personal:
        oauth_token: personal-token
      work:
        oauth_token: work-token
  enterprise.example:
    user: work
    oauth_token: enterprise-work-token
    users:
      work:
        oauth_token: enterprise-work-token
`)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)

	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	newRepo := func(account string) string {
		dir := t.TempDir()
		cmd := exec.Command("git", "init", "-q", dir)
		require.NoError(t, cmd.Run())
		if account != "" {
			cmd = exec.Command("git", "-C", dir, "config", "--local", "github.account", account)
			require.NoError(t, cmd.Run())
		}
		return dir
	}
	personal := newRepo("personal")
	work := newRepo("work")
	fallback := newRepo("")
	unknown := newRepo("missing")

	request := func(dir, host string) error {
		ios, _, _, _ := iostreams.Test()
		client, err := httpClientFunc(func() (gh.Config, error) { return cfg, nil }, ios, "test", "", nil, &git.Client{RepoDir: dir})()
		if err != nil {
			return err
		}
		req, err := http.NewRequest("GET", server.URL, nil)
		if err != nil {
			return err
		}
		req.Host = host
		res, err := client.Do(req)
		if err != nil {
			return err
		}
		return res.Body.Close()
	}

	require.NoError(t, request(personal, "github.com"))
	require.Equal(t, "token personal-token", authorization)
	require.NoError(t, request(work, "github.com"))
	require.Equal(t, "token work-token", authorization)
	require.NoError(t, request(personal, "github.com"))
	require.Equal(t, "token personal-token", authorization)
	require.NoError(t, request(fallback, "github.com"))
	require.Equal(t, "token active-token", authorization)
	require.NoError(t, request(work, "enterprise.example"))
	require.Equal(t, "token enterprise-work-token", authorization)

	err := request(unknown, "github.com")
	require.ErrorContains(t, err, `github.account "missing" has no stored token for github.com`)

	t.Setenv("GH_TOKEN", "env-token")
	require.NoError(t, request(personal, "github.com"))
	require.Equal(t, "token env-token", authorization)
	require.NoError(t, request(unknown, "github.com"))
	require.Equal(t, "token env-token", authorization)
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "github-env-token")
	require.NoError(t, request(work, "github.com"))
	require.Equal(t, "token github-env-token", authorization)
	t.Setenv("GH_ENTERPRISE_TOKEN", "enterprise-env-token")
	require.NoError(t, request(work, "enterprise.example"))
	require.Equal(t, "token enterprise-env-token", authorization)

	// The account setting and the globally active account are unchanged.
	account, err := repoAccount(&git.Client{RepoDir: personal})
	require.NoError(t, err)
	require.Equal(t, "personal", account)
	active, err := cfg.Authentication().ActiveUser("github.com")
	require.NoError(t, err)
	require.Equal(t, "active", active)
	_, err = os.Stat(filepath.Join(personal, ".git", "config"))
	require.NoError(t, err)
}

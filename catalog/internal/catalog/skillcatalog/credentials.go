package skillcatalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// loadRepoCredentials reads the git token for repo's credentialRef from dir.
// credentialRef is validated as a plain filename at config-parse time so it cannot
// escape the directory; the file is re-read on each call to pick up hot-reloaded
// Secrets. The token reaches git only via GIT_ASKPASS (see RepoResolver.gitEnv),
// never argv, a URL, or a log line.
func loadRepoCredentials(dir string, repo SkillRepository) (*Credentials, error) {
	if repo.CredentialRef == "" {
		return nil, nil
	}
	data, err := os.ReadFile(filepath.Join(dir, repo.CredentialRef))
	if err != nil {
		return nil, fmt.Errorf("reading git token for credentialRef %q: %w", repo.CredentialRef, err)
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return nil, fmt.Errorf("git token for credentialRef %q is empty", repo.CredentialRef)
	}
	return &Credentials{Username: gitTokenUsername, Token: token}, nil
}

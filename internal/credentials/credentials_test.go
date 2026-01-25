package credentials

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetCurrentUser(t *testing.T) {
	u, err := GetCurrentUser()
	if err != nil {
		t.Fatalf("GetCurrentUser() error: %v", err)
	}

	if u.Username == "" {
		t.Error("Username should not be empty")
	}
	if u.HomeDir == "" {
		t.Error("HomeDir should not be empty")
	}
}

func TestParseGitService(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantName string
		wantHost string
		wantErr  bool
	}{
		{
			name:     "github https",
			url:      "https://github.com/user/repo.git",
			wantName: "github",
			wantHost: "github.com",
		},
		{
			name:     "github ssh",
			url:      "git@github.com:user/repo.git",
			wantName: "github",
			wantHost: "github.com",
		},
		{
			name:     "gitlab https",
			url:      "https://gitlab.com/user/repo.git",
			wantName: "gitlab",
			wantHost: "gitlab.com",
		},
		{
			name:     "gitlab ssh",
			url:      "git@gitlab.com:user/repo.git",
			wantName: "gitlab",
			wantHost: "gitlab.com",
		},
		{
			name:     "self-hosted gitlab",
			url:      "https://gitlab.mycompany.com/team/repo.git",
			wantName: "gitlab",
			wantHost: "gitlab.mycompany.com",
		},
		{
			name:     "bitbucket https",
			url:      "https://bitbucket.org/user/repo.git",
			wantName: "bitbucket",
			wantHost: "bitbucket.org",
		},
		{
			name:     "local path",
			url:      "/home/user/repos/myproject",
			wantName: "local",
			wantHost: "localhost",
		},
		{
			name:     "unknown service",
			url:      "https://git.mycompany.com/repo.git",
			wantName: "other",
			wantHost: "git.mycompany.com",
		},
		{
			name:    "empty url",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := ParseGitService(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseGitService(%q) expected error, got nil", tt.url)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseGitService(%q) error: %v", tt.url, err)
			}
			if service.Name != tt.wantName {
				t.Errorf("ParseGitService(%q).Name = %q, want %q", tt.url, service.Name, tt.wantName)
			}
			if service.Host != tt.wantHost {
				t.Errorf("ParseGitService(%q).Host = %q, want %q", tt.url, service.Host, tt.wantHost)
			}
		})
	}
}

func TestCheckNetrc(t *testing.T) {
	// Create temp directory with test .netrc
	tmpDir := t.TempDir()
	netrcPath := filepath.Join(tmpDir, ".netrc")

	// Write test netrc content
	content := `machine github.com login testuser password testtoken
machine gitlab.com login anotheruser password anothertoken
`
	if err := os.WriteFile(netrcPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test netrc: %v", err)
	}

	tests := []struct {
		name     string
		host     string
		wantAvail bool
	}{
		{
			name:      "github found",
			host:      "github.com",
			wantAvail: true,
		},
		{
			name:      "gitlab found",
			host:      "gitlab.com",
			wantAvail: true,
		},
		{
			name:      "bitbucket not found",
			host:      "bitbucket.org",
			wantAvail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := checkNetrc(tmpDir, tt.host)
			if status.Available != tt.wantAvail {
				t.Errorf("checkNetrc(%q).Available = %v, want %v", tt.host, status.Available, tt.wantAvail)
			}
		})
	}
}

func TestCheckNetrc_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	status := checkNetrc(tmpDir, "github.com")
	if status.Available {
		t.Error("checkNetrc should return Available=false when .netrc doesn't exist")
	}
}

func TestValidateCredentials_LocalRepo(t *testing.T) {
	u := &User{
		Username: "testuser",
		HomeDir:  t.TempDir(),
	}
	service := &GitService{
		Name:     "local",
		Host:     "localhost",
		IsCustom: true,
	}

	result := ValidateCredentials(u, service)
	if !result.Valid {
		t.Error("Local repos should always validate as true")
	}
	if len(result.FixOptions) != 0 {
		t.Error("Local repos should not have fix options")
	}
}

func TestGetFixOptions(t *testing.T) {
	tests := []struct {
		name        string
		service     *GitService
		wantMinOpts int
	}{
		{
			name:        "github",
			service:     &GitService{Name: "github", Host: "github.com"},
			wantMinOpts: 2, // gh auth, netrc
		},
		{
			name:        "gitlab",
			service:     &GitService{Name: "gitlab", Host: "gitlab.com"},
			wantMinOpts: 2, // glab auth, netrc
		},
		{
			name:        "self-hosted gitlab",
			service:     &GitService{Name: "gitlab", Host: "gitlab.mycompany.com", IsCustom: true},
			wantMinOpts: 2, // netrc, ssh
		},
		{
			name:        "other",
			service:     &GitService{Name: "other", Host: "git.example.com"},
			wantMinOpts: 2, // netrc, ssh
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := getFixOptions(tt.service)
			if len(options) < tt.wantMinOpts {
				t.Errorf("getFixOptions(%s) returned %d options, want at least %d", tt.name, len(options), tt.wantMinOpts)
			}
		})
	}
}

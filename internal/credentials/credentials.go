// Package credentials handles user identity detection and git credential validation.
// Follows AI-friendly design: solve don't punt, self-documenting, no magic values.
package credentials

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
)

// User represents the current system user
type User struct {
	Username string `json:"username"`
	HomeDir  string `json:"home_dir"`
	UID      string `json:"uid,omitempty"`
}

// GitService represents a git hosting service (GitHub, GitLab, etc.)
type GitService struct {
	Name     string `json:"name"`     // "github", "gitlab", "bitbucket", "other"
	Host     string `json:"host"`     // "github.com", "gitlab.com", etc.
	IsCustom bool   `json:"is_custom"` // true if self-hosted
}

// CredentialStatus represents the validation status for a single credential source
type CredentialStatus struct {
	Source    string `json:"source"`    // "gh_auth", "glab_auth", "netrc", "ssh_agent"
	Available bool   `json:"available"` // true if credentials found
	Message   string `json:"message"`   // Human-readable status
}

// ValidationResult contains the full credential validation result
type ValidationResult struct {
	User       User               `json:"user"`
	Service    GitService         `json:"service"`
	Valid      bool               `json:"valid"`      // true if any valid credential found
	Statuses   []CredentialStatus `json:"statuses"`   // Individual source statuses
	FixOptions []FixOption        `json:"fix_options,omitempty"` // Only populated if invalid
}

// FixOption provides actionable remediation steps
type FixOption struct {
	Command     string `json:"command,omitempty"`     // Shell command to run
	Description string `json:"description"`           // What this fix does
	Manual      string `json:"manual,omitempty"`      // Manual steps if not a command
}

// GetCurrentUser returns the current system user
// Uses os/user.Current() which works on all platforms
func GetCurrentUser() (*User, error) {
	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to detect current user: %w", err)
	}

	return &User{
		Username: u.Username,
		HomeDir:  u.HomeDir,
		UID:      u.Uid,
	}, nil
}

// ParseGitService extracts the git service from a URL or local path
// Handles: https://github.com/..., git@github.com:..., /local/path
func ParseGitService(remoteURL string) (*GitService, error) {
	if remoteURL == "" {
		return nil, fmt.Errorf("empty git remote URL")
	}

	// Handle local paths - need to get remote from the repo
	if strings.HasPrefix(remoteURL, "/") || strings.HasPrefix(remoteURL, "~") {
		return &GitService{
			Name:     "local",
			Host:     "localhost",
			IsCustom: true,
		}, nil
	}

	// Handle SSH format: git@github.com:user/repo.git
	if strings.HasPrefix(remoteURL, "git@") {
		re := regexp.MustCompile(`^git@([^:]+):`)
		if matches := re.FindStringSubmatch(remoteURL); matches != nil {
			return parseHost(matches[1])
		}
	}

	// Handle HTTPS format: https://github.com/user/repo.git
	parsed, err := url.Parse(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("invalid git URL: %w", err)
	}

	if parsed.Host != "" {
		return parseHost(parsed.Host)
	}

	return nil, fmt.Errorf("could not parse git service from URL: %s", remoteURL)
}

// parseHost categorizes a hostname into a known git service
func parseHost(host string) (*GitService, error) {
	host = strings.ToLower(host)

	// Known services
	switch {
	case strings.Contains(host, "github.com"):
		return &GitService{Name: "github", Host: "github.com", IsCustom: false}, nil
	case strings.Contains(host, "gitlab.com"):
		return &GitService{Name: "gitlab", Host: "gitlab.com", IsCustom: false}, nil
	case strings.Contains(host, "bitbucket.org"):
		return &GitService{Name: "bitbucket", Host: "bitbucket.org", IsCustom: false}, nil
	case strings.Contains(host, "gitlab"):
		// Self-hosted GitLab (contains "gitlab" in hostname)
		return &GitService{Name: "gitlab", Host: host, IsCustom: true}, nil
	default:
		return &GitService{Name: "other", Host: host, IsCustom: true}, nil
	}
}

// ValidateCredentials checks if the user has valid credentials for the git service
// Returns a ValidationResult with detailed status and fix options
func ValidateCredentials(u *User, service *GitService) *ValidationResult {
	result := &ValidationResult{
		User:    *u,
		Service: *service,
		Valid:   false,
	}

	switch service.Name {
	case "github":
		result.Statuses = validateGitHub(u, service)
	case "gitlab":
		result.Statuses = validateGitLab(u, service)
	case "bitbucket":
		result.Statuses = validateBitbucket(u, service)
	case "local":
		// Local paths don't need credential validation
		result.Valid = true
		result.Statuses = []CredentialStatus{
			{Source: "local", Available: true, Message: "Local repository - no remote credentials needed"},
		}
		return result
	default:
		// For unknown services, check SSH and netrc
		result.Statuses = validateGeneric(u, service)
	}

	// Check if any credential source is valid
	for _, status := range result.Statuses {
		if status.Available {
			result.Valid = true
			break
		}
	}

	// Add fix options if invalid
	if !result.Valid {
		result.FixOptions = getFixOptions(service)
	}

	return result
}

// validateGitHub checks GitHub credentials via gh auth and netrc
func validateGitHub(u *User, service *GitService) []CredentialStatus {
	var statuses []CredentialStatus

	// Check gh CLI auth status
	ghStatus := checkGhAuth()
	statuses = append(statuses, ghStatus)

	// Check ~/.netrc
	netrcStatus := checkNetrc(u.HomeDir, "github.com")
	statuses = append(statuses, netrcStatus)

	// Check SSH agent for github.com
	sshStatus := checkSSHAgentForHost("github.com")
	statuses = append(statuses, sshStatus)

	return statuses
}

// validateGitLab checks GitLab credentials via glab auth and netrc
func validateGitLab(u *User, service *GitService) []CredentialStatus {
	var statuses []CredentialStatus

	// Check glab CLI auth status (only for gitlab.com, not self-hosted)
	if !service.IsCustom {
		glabStatus := checkGlabAuth()
		statuses = append(statuses, glabStatus)
	}

	// Check ~/.netrc
	netrcStatus := checkNetrc(u.HomeDir, service.Host)
	statuses = append(statuses, netrcStatus)

	// Check SSH agent
	sshStatus := checkSSHAgentForHost(service.Host)
	statuses = append(statuses, sshStatus)

	return statuses
}

// validateBitbucket checks Bitbucket credentials
func validateBitbucket(u *User, service *GitService) []CredentialStatus {
	var statuses []CredentialStatus

	// Check ~/.netrc
	netrcStatus := checkNetrc(u.HomeDir, "bitbucket.org")
	statuses = append(statuses, netrcStatus)

	// Check SSH agent
	sshStatus := checkSSHAgentForHost("bitbucket.org")
	statuses = append(statuses, sshStatus)

	return statuses
}

// validateGeneric checks generic git credentials (SSH, netrc)
func validateGeneric(u *User, service *GitService) []CredentialStatus {
	var statuses []CredentialStatus

	// Check ~/.netrc
	netrcStatus := checkNetrc(u.HomeDir, service.Host)
	statuses = append(statuses, netrcStatus)

	// Check SSH agent
	sshStatus := checkSSHAgentForHost(service.Host)
	statuses = append(statuses, sshStatus)

	return statuses
}

// checkGhAuth checks GitHub CLI authentication status
func checkGhAuth() CredentialStatus {
	cmd := exec.Command("gh", "auth", "status")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Check if gh is installed
		if _, lookErr := exec.LookPath("gh"); lookErr != nil {
			return CredentialStatus{
				Source:    "gh_auth",
				Available: false,
				Message:   "GitHub CLI (gh) not installed",
			}
		}
		return CredentialStatus{
			Source:    "gh_auth",
			Available: false,
			Message:   "Not logged in to GitHub CLI",
		}
	}

	// Parse output to get username
	outputStr := string(output)
	if strings.Contains(outputStr, "Logged in to") {
		return CredentialStatus{
			Source:    "gh_auth",
			Available: true,
			Message:   "Logged in via GitHub CLI",
		}
	}

	return CredentialStatus{
		Source:    "gh_auth",
		Available: false,
		Message:   "GitHub CLI status unclear",
	}
}

// checkGlabAuth checks GitLab CLI authentication status
func checkGlabAuth() CredentialStatus {
	cmd := exec.Command("glab", "auth", "status")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Check if glab is installed
		if _, lookErr := exec.LookPath("glab"); lookErr != nil {
			return CredentialStatus{
				Source:    "glab_auth",
				Available: false,
				Message:   "GitLab CLI (glab) not installed",
			}
		}
		return CredentialStatus{
			Source:    "glab_auth",
			Available: false,
			Message:   "Not logged in to GitLab CLI",
		}
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "Logged in to") || strings.Contains(outputStr, "Token:") {
		return CredentialStatus{
			Source:    "glab_auth",
			Available: true,
			Message:   "Logged in via GitLab CLI",
		}
	}

	return CredentialStatus{
		Source:    "glab_auth",
		Available: false,
		Message:   "GitLab CLI status unclear",
	}
}

// checkNetrc checks ~/.netrc for credentials for a host
func checkNetrc(homeDir, host string) CredentialStatus {
	netrcPath := filepath.Join(homeDir, ".netrc")

	file, err := os.Open(netrcPath)
	if err != nil {
		return CredentialStatus{
			Source:    "netrc",
			Available: false,
			Message:   fmt.Sprintf("~/.netrc not found or not readable"),
		}
	}
	defer file.Close()

	// Parse netrc format: machine <host> login <user> password <token>
	scanner := bufio.NewScanner(file)
	inMachine := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)

		for i := 0; i < len(fields); i++ {
			if fields[i] == "machine" && i+1 < len(fields) {
				if strings.Contains(fields[i+1], host) {
					inMachine = true
				} else {
					inMachine = false
				}
			}
			if inMachine && (fields[i] == "password" || fields[i] == "login") {
				return CredentialStatus{
					Source:    "netrc",
					Available: true,
					Message:   fmt.Sprintf("Entry found in ~/.netrc for %s", host),
				}
			}
		}
	}

	return CredentialStatus{
		Source:    "netrc",
		Available: false,
		Message:   fmt.Sprintf("No entry for %s in ~/.netrc", host),
	}
}

// checkSSHAgentForHost checks if SSH agent has keys
// The host parameter is accepted for interface consistency but not used
// (SSH agent keys work for any host)
func checkSSHAgentForHost(host string) CredentialStatus {
	// Check if SSH_AUTH_SOCK is set
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		return CredentialStatus{
			Source:    "ssh_agent",
			Available: false,
			Message:   "SSH agent not running (SSH_AUTH_SOCK not set)",
		}
	}

	// Check if ssh-add -l returns any keys
	cmd := exec.Command("ssh-add", "-l")
	output, err := cmd.Output()

	if err != nil {
		return CredentialStatus{
			Source:    "ssh_agent",
			Available: false,
			Message:   "No SSH keys loaded in agent",
		}
	}

	if strings.TrimSpace(string(output)) != "" && !strings.Contains(string(output), "no identities") {
		return CredentialStatus{
			Source:    "ssh_agent",
			Available: true,
			Message:   "SSH agent has loaded keys",
		}
	}

	return CredentialStatus{
		Source:    "ssh_agent",
		Available: false,
		Message:   "No SSH keys loaded in agent",
	}
}

// getFixOptions returns remediation options for a git service
func getFixOptions(service *GitService) []FixOption {
	var options []FixOption

	switch service.Name {
	case "github":
		options = append(options,
			FixOption{
				Command:     "gh auth login",
				Description: "Authenticate with GitHub CLI (recommended)",
			},
			FixOption{
				Description: "Add entry to ~/.netrc",
				Manual:      fmt.Sprintf("machine %s login <username> password <personal-access-token>", service.Host),
			},
			FixOption{
				Command:     "ssh-add ~/.ssh/id_ed25519",
				Description: "Add SSH key to agent (if using SSH)",
			},
		)
	case "gitlab":
		if service.IsCustom {
			options = append(options,
				FixOption{
					Description: "Add entry to ~/.netrc",
					Manual:      fmt.Sprintf("machine %s login <username> password <personal-access-token>", service.Host),
				},
				FixOption{
					Command:     "ssh-add ~/.ssh/id_ed25519",
					Description: "Add SSH key to agent (if using SSH)",
				},
			)
		} else {
			options = append(options,
				FixOption{
					Command:     "glab auth login",
					Description: "Authenticate with GitLab CLI (recommended)",
				},
				FixOption{
					Description: "Add entry to ~/.netrc",
					Manual:      fmt.Sprintf("machine %s login <username> password <personal-access-token>", service.Host),
				},
				FixOption{
					Command:     "ssh-add ~/.ssh/id_ed25519",
					Description: "Add SSH key to agent (if using SSH)",
				},
			)
		}
	default:
		options = append(options,
			FixOption{
				Description: "Add entry to ~/.netrc",
				Manual:      fmt.Sprintf("machine %s login <username> password <token>", service.Host),
			},
			FixOption{
				Command:     "ssh-add ~/.ssh/id_ed25519",
				Description: "Add SSH key to agent (if using SSH)",
			},
		)
	}

	return options
}

// GetGitRemoteFromRepo gets the origin remote URL from a git repository
func GetGitRemoteFromRepo(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git remote: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

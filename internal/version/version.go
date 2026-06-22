// internal/version/version.go

package version

var (
    Version   = "dev"
    BuildTime = "unknown"
    GitCommit = "unknown"
)

// GetInfo returns a map of version information
func GetInfo() map[string]string {
    return map[string]string{
        "version":     Version,
        "build_time":  BuildTime,
        "git_commit":  GitCommit,
    }
}
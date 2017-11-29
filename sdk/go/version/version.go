package version

var (
	// Version will get assigned the release number at compile time
	Version string
)

// GetVersion returns the release number if it was assigned by the compiler
// or "dev" otherwise.
func GetVersion() string {
	if Version != "" {
		return Version
	}
	return "dev"
}

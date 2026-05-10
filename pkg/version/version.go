package version

// Version is set at build time via -ldflags.
var Version = "unstable" //nolint:gochecknoglobals // default value for development builds

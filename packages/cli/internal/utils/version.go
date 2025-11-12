package utils

import (
	"fmt"
	"runtime"
)

const (
	Version = "1.0.0"
	BuildDate = "2025-01-01"
)

type VersionInfo struct {
	Version   string
	BuildDate string
	GoVersion string
	OS        string
	Arch      string
	Commit    string
}

func GetVersionInfo(commit string) *VersionInfo {
	return &VersionInfo{
		Version:   Version,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Commit:    commit,
	}
}

func (v *VersionInfo) String() string {
	return fmt.Sprintf("Sentra Lab v%s\nBuild Date: %s\nGo Version: %s\nOS/Arch: %s/%s\nCommit: %s",
		v.Version,
		v.BuildDate,
		v.GoVersion,
		v.OS,
		v.Arch,
		v.Commit,
	)
}
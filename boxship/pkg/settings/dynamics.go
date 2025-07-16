package settings

import (
	"path/filepath"
)

func GetWorkDir(set Settings, elem ...string) string {
	var elems []string

	elems = append(elems, set.GetString(WorkDir))
	elems = append(elems, elem...)

	return filepath.Join(elems...)
}

func GetLogsDir(set Settings, elem ...string) string {
	var elems []string

	elems = append(elems, GetWorkDir(set), "logs")
	elems = append(elems, elem...)

	return filepath.Join(elems...)
}

func GetRepoDir(set Settings, elem ...string) string {
	var elems []string

	elems = append(elems, GetWorkDir(set), "git-repo")
	elems = append(elems, elem...)

	return filepath.Join(elems...)
}

func GetCertsDir(set Settings, elem ...string) string {
	var elems []string

	elems = append(elems, GetWorkDir(set), "certs")
	elems = append(elems, elem...)

	return filepath.Join(elems...)
}

func GetConfigsDir(set Settings, elem ...string) string {
	var elems []string

	elems = append(elems, GetWorkDir(set), "config")
	elems = append(elems, elem...)

	return filepath.Join(elems...)
}

func GetVolumesDir(set Settings, elem ...string) string {
	var elems []string

	elems = append(elems, GetWorkDir(set), "vol")
	elems = append(elems, elem...)

	return filepath.Join(elems...)
}

package utils

import (
	"path"
	"strings"
)

func Basename(p string) string {
	return path.Base(p)
}

func BasenameWithoutExtension(p string) string {
	return strings.TrimSuffix(Basename(p), path.Ext(p))
}

func Extension(p string) string {
	return path.Ext(p)
}

func ParentDirectory(p string) string {
	return path.Dir(p)
}

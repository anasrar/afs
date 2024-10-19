package afs

import (
	"path"
	"strings"
	"unicode"
)

func _IsAttributeInfoValid(attributesOffset, attributesSize, afsFileSize, entryTotal, dataBlockEndOffset uint32) bool {
	if attributesOffset == 0 {
		return false
	}
	if attributesSize == 0 {
		return false
	}

	if attributesSize > afsFileSize-dataBlockEndOffset {
		return false
	}
	if attributesSize < entryTotal*AttributeElementSize {
		return false
	}
	if attributesOffset < dataBlockEndOffset {
		return false
	}
	if attributesOffset > afsFileSize-attributesSize {
		return false
	}

	return true
}

func _Pad(value uint32, alignment uint32) uint32 {
	mod := value % alignment
	if mod != 0 {
		return value + (alignment - mod)
	} else {
		return value
	}
}

func _StringFromBytes(b []byte) string {
	str := string(b)
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, str)

}

func _Basename(p string) string {
	return path.Base(p)
}

func _BasenameWithoutExtension(p string) string {
	return strings.TrimSuffix(_Basename(p), path.Ext(p))
}

func _Extension(p string) string {
	return path.Ext(p)
}

func _ParentDirectory(p string) string {
	return path.Dir(p)
}

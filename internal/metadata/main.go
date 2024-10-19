package metadata

import "github.com/anasrar/afs/pkg/afs"

type MetadataEntry struct {
	IsNull        bool   `json:"is_null"`
	Source        string `json:"source"`
	Name          string `json:"name"`
	LastWriteTime string `json:"last_write_time"`
}

type Metadata struct {
	Version             afs.Version        `json:"version"`
	AttributesInfo      afs.AttributesInfo `json:"attributes_info"`
	EntryBlockAlignment uint32             `json:"entry_block_alignment"`
	EntryTotal          uint32             `json:"entry_total"`
	Entries             []*MetadataEntry   `json:"entries"`
}

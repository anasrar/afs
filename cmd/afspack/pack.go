package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anasrar/afs/internal/metadata"
	"github.com/anasrar/afs/internal/utils"
	"github.com/anasrar/afs/pkg/afs"
)

func pack(
	ctx context.Context,
	metadataPath string,
	onStart,
	onDone func(total uint32, current uint32, name string),
) error {
	metadataBuf, err := os.ReadFile(metadataPath)
	if err != nil {
		return err
	}

	var m metadata.Metadata
	if err := json.Unmarshal(metadataBuf, &m); err != nil {
		return err
	}

	a := afs.New()
	a.Version = m.Version
	a.AttributesInfo = m.AttributesInfo
	a.EntryBlockAlignment = m.EntryBlockAlignment

	parentDir := utils.ParentDirectory(metadataPath)

	for _, entry := range m.Entries {
		if entry.IsNull {
			a.AddNullEntry(entry.Name)
		} else {
			if err := a.AddEntryFromPathWithNameLastWriteTime(
				fmt.Sprintf("%s/%s", parentDir, entry.Source),
				entry.Name,
				entry.LastWriteTime,
			); err != nil {
				return err
			}
		}
	}

	if err := a.Pack(
		ctx,
		fmt.Sprintf("%s/OUTPUT.AFS", parentDir),
		onStart,
		onDone,
	); err != nil {
		return err
	}

	return nil
}

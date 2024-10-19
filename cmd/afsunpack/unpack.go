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

func unpack(
	ctx context.Context,
	afsPath string,
	onStart,
	onDone func(total uint32, current uint32, name string),
) error {
	a := afs.New()
	if err := afs.FromPath(a, afsPath); err != nil {
		return err
	}

	outputMetadataPath := fmt.Sprintf("%s/UNPACK_%s/METADATA.json", utils.ParentDirectory(afsPath), utils.Basename(afsPath))

	md := metadata.Metadata{
		Version:             a.Version,
		AttributesInfo:      a.AttributesInfo,
		EntryBlockAlignment: a.EntryBlockAlignment,
		EntryTotal:          a.EntryTotal,
		Entries:             []*metadata.MetadataEntry{},
	}

	duplicates := map[string]int{}

	for _, entry := range a.Entries {
		name := entry.Name

		count, found := duplicates[entry.Name]
		if found {
			v := count + 1
			duplicates[entry.Name] = v
			entry.Name = fmt.Sprintf("%s_%d%s", utils.BasenameWithoutExtension(entry.Name), v, utils.Extension(entry.Name))
		} else {
			duplicates[entry.Name] = 0
		}

		md.Entries = append(
			md.Entries,
			&metadata.MetadataEntry{
				IsNull:        entry.IsNull,
				Source:        fmt.Sprintf("FILES/%s", entry.Name),
				Name:          name,
				LastWriteTime: entry.LastWriteTime,
			},
		)
	}

	outputFilesDirPath := fmt.Sprintf("%s/UNPACK_%s/FILES", utils.ParentDirectory(afsPath), utils.Basename(afsPath))
	if err := os.MkdirAll(outputFilesDirPath, os.ModePerm); err != nil {
		return err
	}

	if err := a.Unpack(ctx, outputFilesDirPath, onStart, onDone); err != nil {
		return err
	}

	buf, err := json.MarshalIndent(md, "", "\t")
	if err != nil {
		return err
	}

	metadataFile, err := os.OpenFile(outputMetadataPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer metadataFile.Close()

	if _, err := metadataFile.Write(buf); err != nil {
		return err
	}

	return nil
}

package afs

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	Signature                  uint32 = 0x534641
	MinEntryBlockAlignmentSize uint32 = 0x800
	AttributeInfoSize          uint32 = 0x8
	AttributeElementSize       uint32 = 0x30
	MaxEntryNameLength         uint32 = 0x20
	DateLayoutFormat           string = "2006-01-02 15:04:05"
	HeaderSize                 uint32 = 0x8
	EntryInfoElementSize       uint32 = 0x8
	AlignmentSize              uint32 = 0x800
)

type Afs struct {
	Version             Version        `json:"version"`
	AttributesInfo      AttributesInfo `json:"attributes_info"`
	EntryBlockAlignment uint32         `json:"entry_block_alignment"`
	EntryTotal          uint32         `json:"entry_total"`
	Entries             []*Entry       `json:"entries"`
}

func (self *Afs) unmarshal(source string, stream io.ReadWriteSeeker) error {
	size, _ := stream.Seek(0, io.SeekEnd)
	if _, err := stream.Seek(0, io.SeekStart); err != nil {
		return err
	}

	var signature uint32
	if err := binary.Read(stream, binary.LittleEndian, &signature); err != nil {
		return err
	}

	if signature&0x00FFFFFF != Signature {
		return fmt.Errorf("Invalid signature")
	}

	switch (signature & 0xFF000000) >> (3 * 8) {
	case 0x00:
		self.Version = Version00
	case 0x20:
		self.Version = Version20
	default:
		return fmt.Errorf("Invalid version")
	}

	if err := binary.Read(stream, binary.LittleEndian, &self.EntryTotal); err != nil {
		return err
	}

	entryBlockStartOffset := uint32(0)
	entryBlockEndOffset := uint32(0)

	for e := uint32(0); e < self.EntryTotal; e++ {
		entry := &Entry{
			Source: source,
		}
		self.Entries = append(self.Entries, entry)

		if err := binary.Read(stream, binary.LittleEndian, &entry.Offset); err != nil {
			return err
		}
		entry.IsNull = entry.Offset == 0

		if err := binary.Read(stream, binary.LittleEndian, &entry.Size); err != nil {
			return err
		}

		if entry.IsNull {
			continue
		}

		if entryBlockStartOffset == 0 {
			entryBlockStartOffset = entry.Offset
		}
		entryBlockEndOffset = entry.Offset + entry.Size
	}

	position, _ := stream.Seek(0, io.SeekCurrent)

	alignment := MinEntryBlockAlignmentSize
	endInfoBlockOffset := uint32(position) + AttributeInfoSize

	for endInfoBlockOffset+alignment < entryBlockStartOffset {
		alignment <<= 1
	}
	self.EntryBlockAlignment = alignment

	self.AttributesInfo = AttributesInfoNoAttribute

	var (
		attributeDataOffset uint32
		attributeDataSize   uint32
	)

	if err := binary.Read(stream, binary.LittleEndian, &attributeDataOffset); err != nil {
		return err
	}

	if err := binary.Read(stream, binary.LittleEndian, &attributeDataSize); err != nil {
		return err
	}

	isAttributeInfoValid := _IsAttributeInfoValid(attributeDataOffset, attributeDataSize, uint32(size), self.EntryTotal, entryBlockEndOffset)

	if isAttributeInfoValid {
		self.AttributesInfo = AttributesInfoInfoAtStart
	} else {
		if _, err := stream.Seek(int64(entryBlockStartOffset-AttributeInfoSize), io.SeekStart); err != nil {
			return err
		}

		if err := binary.Read(stream, binary.LittleEndian, &attributeDataOffset); err != nil {
			return err
		}

		if err := binary.Read(stream, binary.LittleEndian, &attributeDataSize); err != nil {
			return err
		}

		isAttributeInfoValid = _IsAttributeInfoValid(attributeDataOffset, attributeDataSize, uint32(size), self.EntryTotal, entryBlockEndOffset)

		if isAttributeInfoValid {
			self.AttributesInfo = AttributesInfoInfoAtEnd
		}
	}

	if self.AttributesInfo != AttributesInfoNoAttribute {
		if _, err := stream.Seek(int64(attributeDataOffset), io.SeekStart); err != nil {
			return err
		}

		for _, entry := range self.Entries {
			if entry.IsNull {

				if _, err := stream.Seek(int64(AttributeElementSize), io.SeekCurrent); err != nil {
					return err
				}
				continue

			} else {

				name := make([]byte, MaxEntryNameLength)
				if _, err := stream.Read(name); err != nil {
					return err
				}

				entry.Name = _StringFromBytes(name)

				var (
					year       uint16
					month      uint16
					day        uint16
					hour       uint16
					minute     uint16
					second     uint16
					customData uint32
				)

				if err := binary.Read(stream, binary.LittleEndian, &year); err != nil {
					return err
				}

				if err := binary.Read(stream, binary.LittleEndian, &month); err != nil {
					return err
				}

				if err := binary.Read(stream, binary.LittleEndian, &day); err != nil {
					return err
				}

				if err := binary.Read(stream, binary.LittleEndian, &hour); err != nil {
					return err
				}

				if err := binary.Read(stream, binary.LittleEndian, &minute); err != nil {
					return err
				}

				if err := binary.Read(stream, binary.LittleEndian, &second); err != nil {
					return err
				}

				if err := binary.Read(stream, binary.LittleEndian, &customData); err != nil {
					return err
				}

				entry.LastWriteTime = fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", year, month, day, hour, minute, second)
				entry.CustomData = customData

			}
		}
	} else {
		for i, entry := range self.Entries {
			entry.Name = fmt.Sprintf("%08d", i)
		}
	}

	return nil
}

func (self *Afs) Pack(
	ctx context.Context,
	output string,
	onStart,
	onDone func(total uint32, current uint32, name string),
) error {
	packFile, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer packFile.Close()

	if err := binary.Write(packFile, binary.LittleEndian, Signature); err != nil {
		return err
	}

	if _, err := packFile.Seek(-1, io.SeekCurrent); err != nil {
		return err
	}

	if err := binary.Write(packFile, binary.LittleEndian, self.Version); err != nil {
		return err
	}

	if err := binary.Write(packFile, binary.LittleEndian, self.EntryTotal); err != nil {
		return err
	}

	offsets := []uint32{}

	firstEntryOffset := _Pad(
		HeaderSize+(EntryInfoElementSize*self.EntryTotal)+AttributeInfoSize,
		self.EntryBlockAlignment,
	)
	currentEntryOffset := firstEntryOffset

	for _, entry := range self.Entries {
		if entry.IsNull {
			offsets = append(offsets, 0)
		} else {
			offsets = append(offsets, currentEntryOffset)
			currentEntryOffset += entry.Size
			currentEntryOffset = _Pad(currentEntryOffset, AlignmentSize)
		}
	}

	for i, entry := range self.Entries {
		if entry.IsNull {
			if err := binary.Write(packFile, binary.LittleEndian, uint32(0)); err != nil {
				return err
			}
			if err := binary.Write(packFile, binary.LittleEndian, uint32(0)); err != nil {
				return err
			}
		} else {
			if err := binary.Write(packFile, binary.LittleEndian, offsets[i]); err != nil {
				return err
			}
			if err := binary.Write(packFile, binary.LittleEndian, entry.Size); err != nil {
				return err
			}
		}
	}

	position, err := packFile.Seek(int64(HeaderSize+(self.EntryTotal*EntryInfoElementSize)), io.SeekStart)
	if err != nil {
		return err
	}

	if _, err := packFile.Write(make([]byte, firstEntryOffset-uint32(position))); err != nil {
		return err
	}

	attributesInfoOffset := currentEntryOffset

	if self.AttributesInfo != AttributesInfoNoAttribute {
		if self.AttributesInfo == AttributesInfoInfoAtStart {
			if _, err := packFile.Seek(int64(HeaderSize+(self.EntryTotal*EntryInfoElementSize)), io.SeekStart); err != nil {
				return err
			}
		} else if self.AttributesInfo == AttributesInfoInfoAtEnd {
			if _, err := packFile.Seek(int64(firstEntryOffset-AttributeInfoSize), io.SeekStart); err != nil {
				return err
			}
		}

		if err := binary.Write(packFile, binary.LittleEndian, attributesInfoOffset); err != nil {
			return err
		}
		if err := binary.Write(packFile, binary.LittleEndian, self.EntryTotal*AttributeElementSize); err != nil {
			return err
		}
	}

	for i, entry := range self.Entries {
		onStart(self.EntryTotal, uint32(i+1), entry.Name)
		if !entry.IsNull {
			if _, err := packFile.Seek(int64(offsets[i]), io.SeekStart); err != nil {
				return err
			}

			entryFile, err := os.Open(entry.Source)
			if err != nil {
				return err
			}
			defer entryFile.Close()

			buf := make([]byte, entry.Size)

			if _, err := entryFile.Read(buf); err != nil {
				return err
			}

			if _, err := packFile.Write(buf); err != nil {
				return err
			}
		}
		onDone(self.EntryTotal, uint32(i+1), entry.Name)

		select {
		case <-ctx.Done():
			return fmt.Errorf("Canceled")
		default:
		}
	}

	if self.AttributesInfo != AttributesInfoNoAttribute {
		if _, err := packFile.Seek(int64(attributesInfoOffset), io.SeekStart); err != nil {
			return err
		}

		for _, entry := range self.Entries {
			if entry.IsNull {
				if _, err := packFile.Seek(int64(AttributeElementSize), io.SeekCurrent); err != nil {
					return err
				}
			} else {
				buf := []byte(entry.Name)

				if _, err := packFile.Write(buf); err != nil {
					return err
				}

				if _, err := packFile.Seek(int64(MaxEntryNameLength-uint32(len(entry.Name))), io.SeekCurrent); err != nil {
					return err
				}

				lastWrite, err := time.Parse(DateLayoutFormat, entry.LastWriteTime)
				if err != nil {
					return err
				}

				if err := binary.Write(packFile, binary.LittleEndian, uint16(lastWrite.Year())); err != nil {
					return err
				}

				if err := binary.Write(packFile, binary.LittleEndian, uint16(lastWrite.Month())); err != nil {
					return err
				}

				if err := binary.Write(packFile, binary.LittleEndian, uint16(lastWrite.Day())); err != nil {
					return err
				}

				if err := binary.Write(packFile, binary.LittleEndian, uint16(lastWrite.Hour())); err != nil {
					return err
				}

				if err := binary.Write(packFile, binary.LittleEndian, uint16(lastWrite.Minute())); err != nil {
					return err
				}

				if err := binary.Write(packFile, binary.LittleEndian, uint16(lastWrite.Second())); err != nil {
					return err
				}

				if err := binary.Write(packFile, binary.LittleEndian, entry.CustomData); err != nil {
					return err
				}

			}
		}
	}

	{
		currentPosition, _ := packFile.Seek(0, io.SeekCurrent)
		endOfFile := _Pad(uint32(currentPosition), AlignmentSize)

		if _, err := packFile.Write(make([]byte, endOfFile-uint32(currentPosition))); err != nil {
			return err
		}
	}

	return nil
}

func (self *Afs) Unpack(
	ctx context.Context,
	dir string,
	onStart,
	onDone func(total uint32, current uint32, name string),
) error {
	total := len(self.Entries)
	for i, entry := range self.Entries {
		if entry.IsNull {
			continue
		}

		onStart(uint32(total), uint32(i+1), entry.Name)

		sourceFile, err := os.Open(entry.Source)
		if err != nil {
			return err
		}
		defer sourceFile.Close()

		if _, err := sourceFile.Seek(int64(entry.Offset), io.SeekStart); err != nil {
			return err
		}

		b := make([]byte, entry.Size)

		if _, err := sourceFile.Read(b); err != nil {
			return err
		}

		unpackFile, err := os.OpenFile(fmt.Sprintf("%s/%s", dir, entry.Name), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer unpackFile.Close()

		if _, err := unpackFile.Write(b); err != nil {
			return err
		}

		onDone(uint32(total), uint32(i+1), entry.Name)

		select {
		case <-ctx.Done():
			return fmt.Errorf("Canceled")
		default:
		}
	}

	return nil
}

func (self *Afs) AddNullEntry(name string) {
	self.Entries = append(
		self.Entries,
		&Entry{
			Source:        "",
			Offset:        0,
			Name:          name,
			Size:          0,
			LastWriteTime: "2000-01-01 00:00:00",
			CustomData:    0,
			IsNull:        true,
		},
	)

	self.EntryTotal += 1
}

func (self *Afs) AddEntryFromPath(source string) error {
	return self.AddEntryFromPathWithNameLastWriteTime(
		source,
		_Basename(source),
		time.Now().Format(DateLayoutFormat),
	)
}

func (self *Afs) AddEntryFromPathWithName(source string, name string) error {
	return self.AddEntryFromPathWithNameLastWriteTime(
		source,
		name,
		time.Now().Format(DateLayoutFormat),
	)
}

func (self *Afs) AddEntryFromPathWithNameLastWriteTime(source string, name string, lastWriteTime string) error {
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()

	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	self.Entries = append(
		self.Entries,
		&Entry{
			Source:        source,
			Offset:        0,
			Name:          name,
			Size:          uint32(size),
			LastWriteTime: lastWriteTime,
			CustomData:    uint32(size),
			IsNull:        size == 0,
		},
	)

	self.EntryTotal += 1

	return nil
}

func New() *Afs {
	result := Afs{
		Version:             Version00,
		AttributesInfo:      AttributesInfoInfoAtStart,
		EntryBlockAlignment: 0x800,
		EntryTotal:          0,
		Entries:             []*Entry{},
	}

	return &result
}

func FromPath(afs *Afs, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return afs.unmarshal(filePath, file)
}

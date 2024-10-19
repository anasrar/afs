# AFS

Tool for unpack and pack AFS file. Simple rewrite in Go from https://github.com/MaikelChan/AFSLib .

## Usage

Run executable without flag will open up GUI.

### GUI

Just drag and drop the file.

![Preview](https://github.com/user-attachments/assets/87561f9b-4f7f-4a53-89c3-c1596fcc5ce4)

### CLI

```bash
afsunpack --afspath <path to AFS>
afspack --metadatapath <path to METADATA.json>
```

## Built With

- https://github.com/MaikelChan/AFSLib
- https://go.dev/
- https://www.raylib.com/
- https://github.com/gen2brain/raylib-go

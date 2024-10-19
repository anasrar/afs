package afs

type Entry struct {
	Source        string `json:"source"`
	Offset        uint32 `json:"offset"`
	Name          string `json:"name"`
	Size          uint32 `json:"size"`
	LastWriteTime string `json:"last_write_time"`
	CustomData    uint32 `json:"custom_data"`
	IsNull        bool   `json:"is_null"`
}

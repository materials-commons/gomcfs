package mcapi

type File struct {
	ID          int    `json:"id"`
	Path        string `json:"path"`
	DirectoryID int    `json:"directory_id"`
	Size        int64  `json:"size"`
	Checksum    string `json:"checksum"`
	MimeType    string `json:"mime_type"`
}

type MCFile struct {
	File
	Directory File `json:"directory"`
}

func (f File) IsFile() bool {
	return f.MimeType != "directory"
}

func (f File) IsDir() bool {
	return f.MimeType == "directory"
}

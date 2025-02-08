package tools

type FileAction struct {
	Folder     string `json:"folder"`
	FilePath   string `json:"file_path"`
	Directory  string `json:"directory"`
	NewContent string `json:"new_content"`
	Content    string `json:"content"`
}

type SearchAction struct {
	FilePath  string `json:"file_path"`
	Pattern   string `json:"pattern"`
	Recursive bool   `json:"recursive"`
}

type SearchResult struct {
	File       string `json:"file"`
	LineNumber int    `json:"line_number"`
	Line       string `json:"line"`
}

type CopyAction struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

type ExtractAction struct {
	HTML     string `json:"html"`
	URL      string `json:"url"`
	FilePath string `json:"file_path"`
}

type MoveAction struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

type AppendAction struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

type CreateDirectoryAction struct {
	DirectoryPath string `json:"directory_path"`
}

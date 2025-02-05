package tools

const (
	write_file  = "write_file"
	read_file   = "read_file"
	edit_file   = "edit_file"
	delete_file = "delete_file"
	list_files  = "list_files"
)

type Tool struct {
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	Parameters  Parameter                   `json:"parameters"`
	HandlerFunc func(ToolTask) (any, error) `json:"-"`
}

type Parameter struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
	Required   []string       `json:"required"`
}

type ToolTask struct {
	Key        string         `json:"key"`
	Parameters map[string]any `json:"parameters"`
}

var WorkerTools = map[string]Tool{
	write_file: {
		Name:        write_file,
		Description: "Use this action to create a new file or overwrite existing content.",
		Parameters: Parameter{
			Type: "object",
			Properties: map[string]any{
				"file_path": map[string]any{
					"type":        "string",
					"description": "The path of the file to write to.",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "The content to write into the file.",
				},
			},
			Required: []string{"file_path", "content"},
		},
		HandlerFunc: ExecuteFileAction,
	},
	read_file: {
		Name:        read_file,
		Description: "Use this action to retrieve the content of an existing file. Useful for verifying file content or analyzing data.",
		Parameters: Parameter{
			Type: "object",
			Properties: map[string]any{
				"file_path": map[string]any{
					"type":        "string",
					"description": "The path of the file to read.",
				},
			},
			Required: []string{"file_path"},
		},
		HandlerFunc: ExecuteFileAction,
	},
	edit_file: {
		Name:        edit_file,
		Description: "Use this action to modify an existing file without losing its original content. Specify the target file path and the additional or changed content.",
		Parameters: Parameter{
			Type: "object",
			Properties: map[string]any{
				"file_path": map[string]any{
					"type":        "string",
					"description": "The path of the file to edit.",
				},
				"new_content": map[string]any{
					"type":        "string",
					"description": "The new content to replace in the file.",
				},
			},
			Required: []string{"file_path", "new_content"},
		},
		HandlerFunc: ExecuteFileAction,
	},
	delete_file: {
		Name:        delete_file,
		Description: "Use this action to remove an existing file from the system. Once removed, this action is irreversible.",
		Parameters: Parameter{
			Type: "object",
			Properties: map[string]any{
				"file_path": map[string]any{
					"type":        "string",
					"description": "The path of the file to delete.",
				},
			},
			Required: []string{"file_path"},
		},
		HandlerFunc: ExecuteFileAction,
	},
	list_files: {
		Name:        list_files,
		Description: "Use this action to generate a tree listing of files in a specified directory.",
		Parameters: Parameter{
			Type: "object",
			Properties: map[string]any{
				"directory": map[string]any{
					"type":        "string",
					"description": "The directory to list files from.",
				},
			},
			Required: []string{"directory"},
		},
		HandlerFunc: ExecuteFileAction,
	},
}

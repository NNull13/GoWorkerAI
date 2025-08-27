package tools

import (
	"errors"
	"log"
	"os"

	"GoWorkerAI/app/utils"
)

// Presets
const (
	PresetApprover        = "approver"
	PresetMinimal         = "minimal"
	PresetReadOnly        = "readonly"
	PresetFileOpsBasic    = "file_basic"
	PresetFileOpsExtended = "file_extended"
	PresetScraperBasic    = "scraper_basic"
	PresetScraperFull     = "scraper_full"
	PresetAll             = "all"
	Custom                = "custom"
)

// Tools
const (
	true_or_false        = "true_or_false"
	run_go_command       = "run_go_command"
	write_file           = "write_file"
	read_file            = "read_file"
	delete_file          = "delete_file"
	list_files           = "list_files"
	copy_file            = "copy_file"
	move_file            = "move_file"
	append_file          = "append_file"
	search_file          = "search_file"
	create_directory     = "create_directory"
	fetch_html_content   = "fetch_html_content"
	extract_links_html   = "extract_links_html"
	extract_text_content = "extract_text_content"
	extract_meta_tags    = "extract_meta_tags"
)

var (
	workerFolder = os.Getenv("WORKER_FOLDER")
)

type Tool struct {
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	Parameters  Parameter                      `json:"parameters"`
	HandlerFunc func(ToolTask) (string, error) `json:"-"`
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

var allTools = map[string]Tool{
	true_or_false: {
		Name:        "true_or_false",
		Description: "Binary decision with a brief reason.",
		Parameters: Parameter{
			Type: "object",
			Properties: map[string]any{
				"answer": map[string]any{
					"type": "string",
					"enum": []string{"true", "false"},
				},
				"reason": map[string]any{
					"type":      "string",
					"maxLength": 333,
				},
			},
			Required: []string{"answer", "reason"},
		},
		HandlerFunc: executeReviewerAction,
	},
	run_go_command: {
		Name:        run_go_command,
		Description: "Use this action to run only GO (Golang) commands in the terminal. One command at a time.",
		Parameters: Parameter{
			Type: "object",
			Properties: map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "The command to run.",
				},
			},
			Required: []string{"command"},
		},
		HandlerFunc: executeCommandAction,
	},
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
		HandlerFunc: executeFileAction,
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
		HandlerFunc: executeFileAction,
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
		HandlerFunc: executeFileAction,
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
		HandlerFunc: executeFileAction,
	},
	copy_file: {
		Name:        copy_file,
		Description: "Use this action to copy a file or an entire directory to a specified destination while maintaining the original structure.",
		Parameters: Parameter{
			Type: "object",
			Properties: map[string]any{
				"source": map[string]any{
					"type":        "string",
					"description": "The source file or directory path.",
				},
				"destination": map[string]any{
					"type":        "string",
					"description": "The destination path where the source will be copied.",
				},
			},
			Required: []string{"source", "destination"},
		},
		HandlerFunc: executeFileAction,
	},
	move_file: {
		Name:        move_file,
		Description: "Use this action to move or rename a file/directory from a source path to a destination path.",
		Parameters: Parameter{
			Type: "object",
			Properties: map[string]any{
				"source": map[string]any{
					"type":        "string",
					"description": "The path of the file or directory to move.",
				},
				"destination": map[string]any{
					"type":        "string",
					"description": "The new path or directory where the file/directory will be placed or renamed.",
				},
			},
			Required: []string{"source", "destination"},
		},
		HandlerFunc: executeFileAction,
	},
	append_file: {
		Name:        append_file,
		Description: "Use this action to append content to an existing file. If the file does not exist, it should be created.",
		Parameters: Parameter{
			Type: "object",
			Properties: map[string]any{
				"file_path": map[string]any{
					"type":        "string",
					"description": "The file to append content to.",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "The content to be appended at the end of the file.",
				},
			},
			Required: []string{"file_path", "content"},
		},
		HandlerFunc: executeFileAction,
	},
	search_file: {
		Name:        search_file,
		Description: "Use this action to search for a text pattern or regex in a file or directory. If it's a directory, optionally search recursively.",
		Parameters: Parameter{
			Type: "object",
			Properties: map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "The file or directory path where to search.",
				},
				"pattern": map[string]any{
					"type":        "string",
					"description": "The pattern (plaintext or regex) to look for.",
				},
				"recursive": map[string]any{
					"type":        "boolean",
					"description": "Whether to search in subdirectories if path is a directory.",
				},
			},
			Required: []string{"path", "pattern"},
		},
		HandlerFunc: executeFileAction,
	},
	create_directory: {
		Name:        create_directory,
		Description: "Use this action to create a new directory (and parent directories if necessary) at the given path.",
		Parameters: Parameter{
			Type: "object",
			Properties: map[string]any{
				"directory_path": map[string]any{
					"type":        "string",
					"description": "The path of the new directory to create.",
				},
			},
			Required: []string{"directory_path"},
		},
		HandlerFunc: executeFileAction,
	},
	fetch_html_content: {
		Name:        fetch_html_content,
		Description: "Fetches raw HTML content from the given URL. Optionally, save it into file_path if provided.",
		Parameters: Parameter{
			Type: "object",
			Properties: map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "The URL to fetch the HTML content from.",
				},
				"file_path": map[string]any{
					"type":        "string",
					"description": "Optional. If provided, the fetched HTML will be saved to this file path.",
				},
			},
			Required: []string{"url"},
		},
		HandlerFunc: executeWebAction,
	},
	extract_links_html: {
		Name:        extract_links_html,
		Description: "Extracts all links from the provided HTML content. Optionally saves them in file_path if provided.",
		Parameters: Parameter{
			Type: "object",
			Properties: map[string]any{
				"html": map[string]any{
					"type":        "string",
					"description": "The HTML content to extract links from.",
				},
				"file_path": map[string]any{
					"type":        "string",
					"description": "Optional. If provided, the extracted links will be saved to this file path.",
				},
			},
			Required: []string{"html"},
		},
		HandlerFunc: executeWebAction,
	},
	extract_text_content: {
		Name:        extract_text_content,
		Description: "Extracts all visible text content from the provided HTML. Optionally saves it in file_path if provided.",
		Parameters: Parameter{
			Type: "object",
			Properties: map[string]any{
				"html": map[string]any{
					"type":        "string",
					"description": "The HTML content to extract text from.",
				},
				"file_path": map[string]any{
					"type":        "string",
					"description": "Optional. If provided, the extracted text will be saved to this file path.",
				},
			},
			Required: []string{"html"},
		},
		HandlerFunc: executeWebAction,
	},
	extract_meta_tags: {
		Name:        extract_meta_tags,
		Description: "Extracts meta tags and the title from the provided HTML. Optionally saves them in file_path if provided.",
		Parameters: Parameter{
			Type: "object",
			Properties: map[string]any{
				"html": map[string]any{
					"type":        "string",
					"description": "The HTML content to extract meta tags from.",
				},
				"file_path": map[string]any{
					"type":        "string",
					"description": "Optional. If provided, the extracted meta info will be saved to this file path.",
				},
			},
			Required: []string{"html"},
		},
		HandlerFunc: executeWebAction,
	},
}

func NewToolkitFromPreset(preset string) map[string]Tool {
	switch preset {
	case PresetApprover:
		return pick(
			true_or_false,
		)
	case PresetMinimal:
		return pick(
			read_file,
			list_files,
		)
	case PresetReadOnly:
		return pick(
			read_file,
			list_files,
			search_file,
		)
	case PresetFileOpsBasic:
		return pick(
			read_file,
			write_file,
			append_file,
			create_directory,
			list_files,
			delete_file,
		)
	case PresetFileOpsExtended:
		return pick(
			read_file,
			write_file,
			append_file,
			create_directory,
			list_files,
			copy_file,
			move_file,
			delete_file,
			search_file,
			run_go_command,
		)
	case PresetScraperBasic:
		return pick(
			fetch_html_content,
			extract_text_content,
			extract_links_html,
			extract_meta_tags,
		)
	case PresetScraperFull:
		return pick(
			fetch_html_content,
			extract_text_content,
			extract_links_html,
			extract_meta_tags,
			write_file,
			create_directory,
			list_files,
		)
	case Custom:
		return make(map[string]Tool)
	case PresetAll:
		fallthrough
	default:
		keys := make([]string, 0, len(allTools))
		for k := range allTools {
			keys = append(keys, k)
		}
		return pick(keys...)
	}
}

func pick(names ...string) map[string]Tool {
	m := make(map[string]Tool, len(names))
	for _, n := range names {
		if t, ok := allTools[n]; ok {
			m[n] = t
		}
	}
	return m
}

func withParsed[T any](params any, op string, f func(T) (string, error)) (string, error) {
	v, err := utils.CastAny[T](params)
	if err != nil {
		log.Printf("❌ Error parsing %s action: %v\n", op, err)
		return "", err
	}
	if v == nil {
		log.Printf("❌ %s action is nil\n", op)
		return "", errors.New("action is nil")
	}
	return f(*v)
}

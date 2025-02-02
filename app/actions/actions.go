package actions

import "GoWorkerAI/app/models"

type Action struct {
	Key         string `json:"key"`
	Description string `json:"description"`
	HandlerFunc func(action *models.ActionTask, folder string) (result string, err error)
}

const (
	write_file  = "write_file"
	read_file   = "read_file"
	edit_file   = "edit_file"
	delete_file = "delete_file"
	list_files  = "list_files"
)

var WorkerActions = []Action{
	{
		Key:         write_file,
		HandlerFunc: ExecuteFileAction,
		Description: "Use this action to create a new file or overwrite existing content. " +
			"Double-check for unwanted formatting or escape sequences.\n\n" +
			"**Example**:\n```json\n{\n  \"action\": \"write_file\",\n  \"filename\": \"src/example.go\",\n  \"content\": \"package main\\n\\nfunc main() {\\n    println(\\\"Hello World\\\")\\n}\\n\"\n}\n```\n\n" +
			"When to use:\n" +
			"- You need to add or overwrite a file with new code or configuration.\n" +
			"- You know the exact file path and the content to be generated.\n",
	},
	{
		Key:         read_file,
		HandlerFunc: ExecuteFileAction,
		Description: "Use this action to retrieve the content of an existing file always after the action list files. " +
			"Useful for verifying file content or analyzing data.\n\n" +
			"**Example**:\n```json\n{\n  \"action\": \"read_file\",\n  \"filename\": \"src/example.go\",\n  \"content\": \"\"\n}\n```\n\n" +
			"When to use:\n" +
			"- You need to see the current content of a file.\n" +
			"- You want to reference existing code or data before making edits.\n",
	},
	{
		Key:         edit_file,
		HandlerFunc: ExecuteFileAction,
		Description: "Use this action to modify an existing file without losing its original content. " +
			"Specify the target file path and the additional or changed content.\n\n" +
			"**Example**:\n```json\n{\n  \"action\": \"edit_file\",\n  \"filename\": \"src/example.go\",\n  \"content\": \"// Added a new comment\\nfunc NewFeature() {}\"\n}\n```\n\n" +
			"When to use:\n" +
			"- You must add or alter lines within a file but not overwrite everything.\n" +
			"- You have read the file and know exactly what changes are needed.\n",
	},
	{
		Key:         delete_file,
		HandlerFunc: ExecuteFileAction,
		Description: "Use this action to remove an existing file from the system. Try to not using this action " +
			"Once removed, this action is irreversible.\n\n" +
			"**Example**:\n```json\n{\n  \"action\": \"delete_file\",\n  \"filename\": \"src/old_file.go\",\n  \"content\": \"\"\n}\n```\n\n" +
			"When to use:\n" +
			"- A file is obsolete or incorrect.\n" +
			"- You need to ensure it is no longer required before deleting.\n",
	},
	{
		Key:         list_files,
		HandlerFunc: ExecuteFileAction,
		Description: "Use this action to generate a tree listing of files in a specified directory. " +
			"Supply the path in 'filename'. The output may include subfolders.\n\n" +
			"**Example**:\n```json\n{\n  \"action\": \"list_files\",\n  \"filename\": \"src\",\n  \"content\": \"\"\n}\n```\n\n" +
			"When to use:\n" +
			"- You want an overview of all files and folders.\n" +
			"- You need to decide which file to read or edit next based on existing structure.\n",
	},
}

package attachments

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StatUploadFile validates an upload path and returns its file information.
func StatUploadFile(filePath string) (os.FileInfo, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("file: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory", filePath)
	}
	return info, nil
}

func contentTypeForFile(fileName string) string {
	extension := strings.ToLower(filepath.Ext(fileName))
	return contentTypesByExtension[extension]
}

// contentTypesByExtension contains known browser-compatible MIME hints.
// Empty content types intentionally let the server infer from the extension.
var contentTypesByExtension = map[string]string{
	".bmp":        "image/bmp",
	".c":          "",
	".copilotmd":  "",
	".cpp":        "",
	".cpuprofile": "",
	".cs":         "",
	".css":        "text/css",
	".csv":        "text/csv",
	".debug":      "",
	".dmp":        "",
	".doc":        "application/msword",
	".docx":       "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".drawio":     "",
	".eml":        "message/rfc822",
	".fodg":       "application/vnd.oasis.opendocument.graphics",
	".fodp":       "application/vnd.oasis.opendocument.presentation",
	".fods":       "application/vnd.oasis.opendocument.spreadsheet",
	".fodt":       "application/vnd.oasis.opendocument.text",
	".gif":        "image/gif",
	".gz":         "application/x-gzip",
	".htm":        "text/html",
	".html":       "text/html",
	".ipynb":      "",
	".java":       "",
	".jpeg":       "image/jpeg",
	".jpg":        "image/jpeg",
	".js":         "text/javascript",
	".json":       "application/json",
	".jsonc":      "",
	".log":        "",
	".md":         "text/markdown",
	".mov":        "video/quicktime",
	".mp3":        "audio/mpeg",
	".mp4":        "video/mp4",
	".msg":        "",
	".odf":        "application/vnd.oasis.opendocument.formula",
	".odg":        "application/vnd.oasis.opendocument.graphics",
	".odp":        "application/vnd.oasis.opendocument.presentation",
	".ods":        "application/vnd.oasis.opendocument.spreadsheet",
	".odt":        "application/vnd.oasis.opendocument.text",
	".patch":      "",
	".pdb":        "application/octet-stream",
	".pdf":        "application/pdf",
	".php":        "text/php",
	".png":        "image/png",
	".pptx":       "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".py":         "text/x-python-script",
	".rtf":        "text/rtf",
	".sh":         "text/x-sh",
	".sql":        "",
	".svg":        "image/svg+xml",
	".tgz":        "application/gzip",
	".tif":        "image/tiff",
	".tiff":       "image/tiff",
	".ts":         "",
	".tsv":        "text/tab-separated-values",
	".tsx":        "",
	".txt":        "text/plain",
	".wav":        "audio/wav",
	".webm":       "video/webm",
	".webp":       "image/webp",
	".xls":        "application/vnd.ms-excel",
	".xlsm":       "application/vnd.ms-excel.sheet.macroenabled.12",
	".xlsx":       "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".xml":        "text/xml",
	".yaml":       "application/x-yaml",
	".yml":        "application/x-yaml",
	".zip":        "application/zip",
}

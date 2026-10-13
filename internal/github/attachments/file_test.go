package attachments

import "testing"

func TestContentTypeForFile(t *testing.T) {
	tests := map[string]string{
		".bmp":  "image/bmp",
		".css":  "text/css",
		".csv":  "text/csv",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".eml":  "message/rfc822",
		".fodg": "application/vnd.oasis.opendocument.graphics",
		".fodp": "application/vnd.oasis.opendocument.presentation",
		".fods": "application/vnd.oasis.opendocument.spreadsheet",
		".fodt": "application/vnd.oasis.opendocument.text",
		".gif":  "image/gif",
		".gz":   "application/x-gzip",
		".htm":  "text/html",
		".html": "text/html",
		".jpeg": "image/jpeg",
		".jpg":  "image/jpeg",
		".js":   "text/javascript",
		".json": "application/json",
		".md":   "text/markdown",
		".mov":  "video/quicktime",
		".mp3":  "audio/mpeg",
		".mp4":  "video/mp4",
		".odf":  "application/vnd.oasis.opendocument.formula",
		".odg":  "application/vnd.oasis.opendocument.graphics",
		".odp":  "application/vnd.oasis.opendocument.presentation",
		".ods":  "application/vnd.oasis.opendocument.spreadsheet",
		".odt":  "application/vnd.oasis.opendocument.text",
		".pdb":  "application/octet-stream",
		".pdf":  "application/pdf",
		".php":  "text/php",
		".png":  "image/png",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".py":   "text/x-python-script",
		".rtf":  "text/rtf",
		".sh":   "text/x-sh",
		".svg":  "image/svg+xml",
		".tgz":  "application/gzip",
		".tif":  "image/tiff",
		".tiff": "image/tiff",
		".tsv":  "text/tab-separated-values",
		".txt":  "text/plain",
		".wav":  "audio/wav",
		".webm": "video/webm",
		".webp": "image/webp",
		".xls":  "application/vnd.ms-excel",
		".xlsm": "application/vnd.ms-excel.sheet.macroenabled.12",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".xml":  "text/xml",
		".yaml": "application/x-yaml",
		".yml":  "application/x-yaml",
		".zip":  "application/zip",
	}

	for extension, want := range tests {
		t.Run(extension, func(t *testing.T) {
			got := contentTypeForFile("file" + extension)
			if got != want {
				t.Fatalf("contentTypeForFile() = %q, want %q", got, want)
			}
		})
	}
	if got := contentTypeForFile("FILE.TXT"); got != "text/plain" {
		t.Fatalf("contentTypeForFile() = %q, want %q", got, "text/plain")
	}
}

func TestContentTypeForFile_ServerInferredExtensionsAreExplicit(t *testing.T) {
	extensions := []string{
		".c", ".copilotmd", ".cpp", ".cpuprofile", ".cs", ".debug", ".dmp", ".drawio", ".ipynb",
		".java", ".jsonc", ".log", ".msg", ".patch", ".sql", ".ts", ".tsx",
	}

	for _, extension := range extensions {
		t.Run(extension, func(t *testing.T) {
			contentType, ok := contentTypesByExtension[extension]
			if !ok || contentType != "" {
				t.Fatalf("contentTypesByExtension[%q] = %q, %v; want empty, true", extension, contentType, ok)
			}
		})
	}
}

func TestContentTypeForFile_UnknownExtensionUsesServerInference(t *testing.T) {
	for _, fileName := range []string{"archive.rar", "README"} {
		if got := contentTypeForFile(fileName); got != "" {
			t.Errorf("contentTypeForFile(%q) = %q, want empty", fileName, got)
		}
	}
}

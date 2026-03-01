package templates

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	echo "github.com/labstack/echo/v5"
)

func GetTemplateRender(rootDir string) (*TemplateRenderer, error) {
	t, err := FindAndParseTemplates(rootDir, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates from %s: %w", rootDir, err)
	}

	return &TemplateRenderer{
		templates: t,
	}, nil
}

func Render(c *echo.Context, code int, name string, data map[string]interface{}) error {

	return c.Render(code, name, data)
}

type TemplateRenderer struct {
	templates *template.Template
}

// FolderExists checks if the folder part of a given file path exists.
func FolderExists(filePath string) bool {
	dirPath := filepath.Dir(filePath)

	// Check if the directory path is empty (e.g., just a filename in the current directory).
	if dirPath == "." || dirPath == "" {
		return true // The current directory always exists
	}

	_, err := os.Stat(dirPath)
	if err == nil {
		// The directory exists
		return true
	}
	if os.IsNotExist(err) {
		// The directory does not exist
		return false
	}
	// Other errors might indicate permission issues or other problems.
	// In this case, we'll consider the folder as not existing for simplicity.
	fmt.Println("Error checking folder:", err)
	return false
}
func (t *TemplateRenderer) Render(c *echo.Context, w io.Writer, name string, data any) error {
	return t.templates.ExecuteTemplate(w, name, data)
}
func FindAndParseTemplates(rootDir string, funcMap template.FuncMap) (*template.Template, error) {
	if !FolderExists(rootDir) {
		return template.New(""), nil
	}

	cleanRoot := filepath.Clean(rootDir)
	pfx := len(cleanRoot) + 1
	root := template.New("")

	err := filepath.Walk(cleanRoot, func(path string, info os.FileInfo, e1 error) error {
		if info == nil {
			return e1
		}
		if !info.IsDir() && strings.HasSuffix(path, ".tpl") {
			if e1 != nil {
				return e1
			}

			b, e2 := ioutil.ReadFile(path)
			if e2 != nil {
				return e2
			}

			name := path[pfx:]
			t := root.New(name).Funcs(funcMap)
			_, e2 = t.Parse(string(b))
			if e2 != nil {
				return e2
			}
		}

		return nil
	})

	return root, err
}

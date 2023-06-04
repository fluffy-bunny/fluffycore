package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	asciitree "github.com/tufin/asciitree"
)

func FilePathWalkDir(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func DumpPath(root string) {
	files, err := FilePathWalkDir(root)
	if err != nil {
		panic(err)
	}
	fmt.Println("==========================================================")
	fmt.Println("Dumping files in " + root)
	fmt.Println("==========================================================")
	tree := asciitree.Tree{}
	for _, file := range files {
		tree.Add("/" + strings.ReplaceAll(file, "\\", "/"))
	}
	tree.Fprint(os.Stdout, true, "")
	fmt.Println("==========================================================")
}

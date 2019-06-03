package utils

import (
	"github.com/sirupsen/logrus"
	"os"
	"path"
)

func OpenFile( root, fileName string) (string, *os.File, error) {
	// Create folder if it doesn't exists
	if !FolderExists(root) {
		_ = os.MkdirAll(root, os.ModePerm)
	}
	fileName = path.Join(root, fileName)

	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm )
	return fileName, f, err
}
func WriteFile(root, fileName, content string) {
	fileName, f, err := OpenFile(root, fileName)

	if err != nil {
		logrus.Errorf("Failed to write file: %s %v", fileName, err)
		return
	}
	_, err = f.WriteString(content)
	if err != nil {
		logrus.Errorf("Failed to write content to file, %v", err)
	}
	_ = f.Close()
}

func FolderExists(root string) bool {
	_, err := os.Stat(root)
	return !os.IsNotExist(err)
}

func ClearFolder(root string) {
	if FolderExists(root) {
		logrus.Infof("Cleaning report folder %s", root)
		_ = os.RemoveAll(root)
	}
	// Create folder, since we delete is already.
	CreateFolders(root)
}

func CreateFolders(root string) {
	err := os.MkdirAll(root, os.ModePerm)
	if err != nil {
		logrus.Errorf("Failed to create folder %s cause %v", root, err)
	}
}
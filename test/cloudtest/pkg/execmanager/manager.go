package execmanager

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"sync"
)

type ExecutionManager interface {
	AddTestLog(category, clusterName, testName, operation, content string)
	AddLog(category, operationName, content string)
}

type executionManagerImpl struct {
	root string
	step int
	sync.Mutex
}

// write file 'clusters/GKE/create'
// write file 'clusters/GKE/tests/testname/output'
// write file 'clusters/GKE/tests/testname/kubectl_logs'
func (mgr *executionManagerImpl) AddTestLog(category, clusterName, testName, operation, content string) {
	mgr.Lock()
	defer mgr.Unlock()
	mgr.writeFile(path.Join(category, clusterName, testName), fmt.Sprintf("%s.log",operation), content)
}

func (mgr *executionManagerImpl) AddLog(category, operationName, content string) {
	mgr.Lock()
	defer mgr.Unlock()
	mgr.step++

	mgr.writeFile(category, fmt.Sprintf("%v-%s.log", mgr.step, operationName), content)
}

func (mgr *executionManagerImpl) writeFile(rootFolder, operation, content string) {
	// Create folder if it doesn't exists
	rootFolder = path.Join(mgr.root, rootFolder)
	if _, err := os.Stat(rootFolder); os.IsNotExist(err) {
		_ = os.MkdirAll(rootFolder, os.ModePerm)
	}
	fileName := path.Join(rootFolder, operation)

	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm )
	if err != nil {
		logrus.Errorf("Failed to write file:  %s %v", fileName, err)
		return
	}
	_, err = f.WriteString(content)
	if err != nil {
		logrus.Errorf("Failed to write content to file, %v", err)
	}
	_ = f.Close()
}

func NewExecutionManager(root string) ExecutionManager {
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		logrus.Infof("Cleaning report folder %s", root)

		_ = os.RemoveAll(root)
	}
	// Create folder, since we delete is already.
	err := os.MkdirAll(root, os.ModePerm)
	if err != nil {
		logrus.Errorf("")
	}
	return &executionManagerImpl{
		root: root,
		step: 0,
	}
}

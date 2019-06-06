package execmanager

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"sync"
)

type ExecutionManager interface {
	AddTestLog(category, clusterName, testName, operation, content string)
	OpenFileTest(category, clusterName, testname, operation string) (string, *os.File, error)
	AddLog(category, operationName, content string)
	OpenFile(category, operationName string) (string, *os.File, error)
	GetRoot(root string) string
	AddFile(fileName string, bytes []byte)
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
	mgr.step++
	utils.WriteFile(path.Join(mgr.root, category), fmt.Sprintf("%d-%s-%s-%s.log", mgr.step, testName, clusterName, operation), content)
}

func (mgr *executionManagerImpl) AddFile(fileName string, bytes []byte) {
	mgr.Lock()
	defer mgr.Unlock()
	mgr.step++
	fileName, f, err := utils.OpenFile(mgr.root, fileName)

	if err != nil {
		logrus.Errorf("Failed to write file: %s %v", fileName, err)
		return
	}
	_, err = f.Write(bytes)
	if err != nil {
		logrus.Errorf("Failed to write content to file, %v", err)
	}
	_ = f.Close()
}

func (mgr *executionManagerImpl) OpenFile(category, operationName string) (string, *os.File, error) {
	mgr.Lock()
	defer mgr.Unlock()
	mgr.step++
	return utils.OpenFile(path.Join(mgr.root, category), fmt.Sprintf("%d-%s.log", mgr.step, operationName))
}
func (mgr *executionManagerImpl) OpenFileTest(category, clusterName, testName, operation string) (string, *os.File, error) {
	mgr.Lock()
	defer mgr.Unlock()
	mgr.step++
	return utils.OpenFile(path.Join(mgr.root, category), fmt.Sprintf("%d-%s-%s-%s.log", mgr.step, testName, clusterName, operation))
}

func (mgr *executionManagerImpl) AddLog(category, operationName, content string) {
	mgr.Lock()
	defer mgr.Unlock()
	mgr.step++

	utils.WriteFile(path.Join(mgr.root, category), fmt.Sprintf("%d-%s.log", mgr.step, operationName), content)
}

func (mgr *executionManagerImpl) GetRoot(root string) string {
	mgr.Lock()
	defer mgr.Unlock()
	initPath := path.Join(mgr.root, root)
	if !utils.FolderExists(initPath) {
		utils.CreateFolders(initPath)
		return initPath
	} else {
		index := 2
		for {
			initPath := path.Join(mgr.root, fmt.Sprintf("%s-%d", root, index))
			if !utils.FolderExists(initPath) {
				utils.CreateFolders(initPath)
				return initPath
			}
			index++
		}
	}
}

func NewExecutionManager(root string) ExecutionManager {
	utils.ClearFolder(root)
	return &executionManagerImpl{
		root: root,
		step: 0,
	}
}

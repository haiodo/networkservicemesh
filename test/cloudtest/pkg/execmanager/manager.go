package execmanager

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
	"path"
	"sync"
)

type ExecutionManager interface {
	AddTestLog(category, clusterName, testName, operation, content string)
	AddLog(category, operationName, content string)
	GetRoot(root string) string
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
	utils.WriteFile(mgr.root, path.Join(category, clusterName, testName), fmt.Sprintf("%s.log",operation), content)
}

func (mgr *executionManagerImpl) AddLog(category, operationName, content string) {
	mgr.Lock()
	defer mgr.Unlock()
	mgr.step++

	utils.WriteFile(mgr.root, category, fmt.Sprintf("%v-%s.log", mgr.step, operationName), content)
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

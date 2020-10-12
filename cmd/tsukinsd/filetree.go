package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
)

type Tree struct {
	Nodes map[string]*Node
	Version int64
	Removed []*Node
	Conf Namenode
}


type Node struct {
	Address string
	IsDirectory bool
	Childs []*Node
	Parent string
	Removed bool
	Pending map[string]bool
	Chunks []string
}

func InitTree(conf Namenode) *Tree {
	root := &Node{Address: ".", IsDirectory: true, Childs: make([]*Node, 0), Parent: ""}
	tree := &Tree{Nodes: map[string]*Node{".": root}, Conf: conf}

	return tree
}

func (t *Tree) String() string {
	return fmt.Sprintf("Tree{FServers: %v}", t.Nodes)
}

func (node *Node) String() string {
	return fmt.Sprintf("Node{Address: %q, IsDirectory: %v, Childs: %v, Parent: %q, Removed: %v, Chunks: %v}", node.Address, node.IsDirectory, node.Childs, node.Parent, node.Removed, node.Chunks)
}

func (t *Tree) CreateFile(fileName string) (*Node, error) {
	fileName, matched := CleanAddress(fileName)

	if !matched {
		return nil, fmt.Errorf("wrong file name format")
	}

	_, fileExists := t.Nodes[fileName]

	if fileExists {
		return nil, fmt.Errorf("/%s file already exists", fileName)
	}


	dirPath := path.Dir(fileName)
	dirExists := t.DirectoryExists(dirPath)

	if !dirExists {
		return nil, fmt.Errorf("/%s/ directory does not exist", dirPath)
	}
	dir, _ := t.GetNodeByAddress(dirPath)

	newFile := &Node{
		Address: fileName,
		IsDirectory: false,
		Childs: nil,
		Parent: dir.Address,
		Pending: map[string]bool{},
	}

	dir.Childs = append(dir.Childs, newFile)
	t.Nodes[fileName] = newFile

	t.CommitUpdate("touch", fileName)

	return newFile, nil
}

func (t *Tree) GetFile(address string) (*Node, error) {
	address = path.Clean(address)

	if !t.FileExists(address) {
		return nil, fmt.Errorf("/%s file does not exist", address)
	}

	node, ok := t.GetNodeByAddress(address)

	if !ok {
		return nil, fmt.Errorf("/%s path does not exist", address)
	}

	return node, nil
}

func (t *Tree) RemoveFile(address string) (*Node, error) {
	address, matched := CleanAddress(address)

	if !matched {
		return nil, fmt.Errorf("/%s wrong file name format", address)
	}

	exists, isDirectory := t.PathExists(address)
	if !exists {
		return nil, fmt.Errorf("/%s file does not exist", address)
	} else if isDirectory {
		return nil, fmt.Errorf("/%s/ cannot remove directory; use rmdir instead", address)
	}

	removed := t.Nodes[address]
	t.Nodes[address].Removed = true // lazy removing
	t.Removed = append(t.Removed, t.Nodes[address])

	delete(t.Nodes, address)

	t.CommitUpdate("rmfile", address)

	return removed, nil
}

func (t *Tree) CreateDirectory(address string) error {
	address, matched := CleanAddress(address)

	if !matched {
		return fmt.Errorf("/%s wrong file name format", address)
	}

	exists, _ := t.PathExists(address)

	if exists {
		return fmt.Errorf("/%s the path already exists", address)
	}

	dirPath := path.Dir(address)
	dirExists := t.DirectoryExists(dirPath)
	if !dirExists {
		return fmt.Errorf("/%s the parent directory (%s) does not exist", address, dirPath)
	}
	dir := t.Nodes[dirPath]

	newDir := &Node{
		Address: address,
		IsDirectory: true,
		Childs: nil,
		Parent: dir.Address,
	}

	dir.Childs = append(dir.Childs, newDir)
	t.Nodes[address] = newDir

	t.CommitUpdate("mkdir", address)

	return nil
}

func (t *Tree) RemoveDirectory(address string) error {
	address, matched := CleanAddress(address)

	if !matched {
		return fmt.Errorf("wrong file name format")
	}
	if !t.DirectoryExists(address) {
		return fmt.Errorf("/%s/ directory does not exist", address)
	}

	node, _ := t.GetNodeByAddress(address)
	node.Removed = true // lazy removing; will be removed later
	t.Removed = append(t.Removed, node)
	delete(t.Nodes, address)

	t.CommitUpdate("rmdir", address)

	return nil
}

func (t *Tree) GetNodeByAddress(address string) (*Node, bool) {
	address,_ = CleanAddress(address)
	node, ok := t.Nodes[address]

	return node, ok
}

func (t *Tree) CD(address string) (string, error) {
	address, matched := CleanAddress(address)

	if !matched {
		return "", fmt.Errorf("wrong file name format")
	}

	exists, isDirectory := t.PathExists(address)

	if !exists {
		return "", fmt.Errorf("/%s/ directory does not exist", address)
	} else if !isDirectory {
		return "", fmt.Errorf("/%s not a directory", address)
	}
	return address, nil
}

func (t *Tree) CopyFile(fileToCopy string, copyTo string) error {
	fileToCopy, fileToCopyMatched := CleanAddress(fileToCopy)
	copyTo, copyToMatched := CleanAddress(copyTo)

	if !fileToCopyMatched {
		return fmt.Errorf("/%s wrong file name format", fileToCopy)
	}
	if !copyToMatched {
		return fmt.Errorf("/%s wrong file name format", copyTo)
	}

	var fullFilePath string

	fileToCopyExists, fileToCopyIsDirectory := t.PathExists(fileToCopy)

	if !fileToCopyExists {
		return fmt.Errorf("/%s file does not exist", fileToCopy)
	} else if fileToCopyIsDirectory {
		return fmt.Errorf("/%s/ cannot copy directory", fileToCopy)
	}

	if t.DirectoryExists(copyTo) {
		fullFilePath = path.Join(copyTo, path.Base(fileToCopy))
	} else if t.FileExists(copyTo) {
		return fmt.Errorf("%s the file already exists", fileToCopy)
	} else {
		fullFilePath = copyTo
	}

	if t.FileExists(fullFilePath) {
		return fmt.Errorf("/%s the file already exists", fullFilePath)
	}

	parentDir := t.Nodes[path.Dir(fullFilePath)]

	copiedFile := *t.Nodes[fileToCopy]
	copiedFile.Parent = parentDir.Address
	copiedFile.Address = fullFilePath

	t.Nodes[fullFilePath] = &copiedFile
	parentDir.Childs = append(parentDir.Childs, &copiedFile)

	t.CommitUpdate("copy", fileToCopy, copyTo)

	return nil
}

func (t *Tree) MoveFile(fileToMove string, moveTo string) error {
	fileToMove, fileToMoveMatched := CleanAddress(fileToMove)
	moveTo, moveToMatched := CleanAddress(moveTo)

	if !fileToMoveMatched {
		return fmt.Errorf("/%s wrong file name format", fileToMove)
	}
	if !moveToMatched {
		return fmt.Errorf("/%s wrong file name format", moveTo)

	}


	err := t.CopyFile(fileToMove, moveTo)

	if err != nil {
		return fmt.Errorf("impossible to move file: %e", err)
	}

	_, _ = t.RemoveFile(fileToMove)

	return nil
}

func (t *Tree) LS(address string) ([]string, error) {
	address, matched := CleanAddress(address)

	if !matched {
		return nil, fmt.Errorf("/%s wrong file name format", address)
	}
	if !t.DirectoryExists(address) {
		return nil, fmt.Errorf("/%s directory does not exist", address)
	}

	dir, _ := t.GetNodeByAddress(address)
	var list = []string{}

	for _, node := range dir.Childs {
		if !t.Exists(node.Address) || node.Removed {
			continue
		}
		name := path.Base(node.Address)
		if node.IsDirectory {
			name += "/"
		}
		list = append(list, name)
	}

	return list, nil
}


func (t *Tree) PathExists(address string) (exists bool, isDirectory bool) {
	address, _ = CleanAddress(address)
	node, ok := t.Nodes[address]

	if ok {
		return ok && len(node.Pending) == 0 && !t.Nodes[address].Removed && t.ParentsExist(node), t.Nodes[address].IsDirectory
	}
	return ok, false
}

func (t *Tree) Exists(address string) bool {
	exists, _ := t.PathExists(address)
	return exists
}

func (t *Tree) ParentsExist(node *Node) bool {
	if node.Address == "." {
		return true
	}

	parent, _ := t.GetNodeByAddress(node.Parent)
	if parent.Removed {
		return false
	}

	result := t.ParentsExist(parent)

	if result == false {
		node.Removed = true
	}

	return result
}
func (t *Tree) FileExists(address string) bool {
	exists, isDirectory := t.PathExists(address)

	return exists && !isDirectory
}

func (t *Tree) DirectoryExists(address string) bool {
	exists, isDirectory := t.PathExists(address)

	return exists && isDirectory
}


func (t *Tree) ParentNode(node *Node) *Node {
	parentNode, _ := t.GetNodeByAddress(node.Parent)
	return parentNode
}


func (t *Tree) SaveTree(saveTo string) bool {
	file, _ := os.Create(saveTo)
	defer file.Close()
	encoder := gob.NewEncoder(file)

	encoder.Encode(t)
	return true
}

func LoadTree(openFrom string) *Tree {
	file, _ := os.Open(openFrom)
	defer file.Close()

	decoder := gob.NewDecoder(file)

	var tree Tree
	decoder.Decode(&tree)

	return &tree
}

func (t *Tree) CommitUpdate(command string, args ...string) {
	f, err := os.OpenFile(t.Conf.TreeLogName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	if _, err := f.WriteString(fmt.Sprintf("%d\t%s\t%s\n", t.Version, command, strings.Join(args, "\t"))); err != nil {
		log.Println(err)
	}

	t.Version += 1

	if t.Version % 100 == t.Conf.TreeUpdatePeriod {
		t.ClearRemoved()
		t.SaveTree(t.Conf.TreeGobName)

		os.Remove(t.Conf.TreeLogName)
	}
}

func (t *Tree) PrintTreeStruct() {
	PrintDir(0, t.Nodes["."])
}

func (t *Tree) ClearRemoved() {
	for _, node := range t.Removed {
		parent := t.ParentNode(node)

		toRemoveInd := -1
		for i, parentChild := range parent.Childs {
			if parentChild.Address == node.Address {
				toRemoveInd = i
				break
			}
		}

		if toRemoveInd >= 0 {
			parent.Childs[toRemoveInd] = parent.Childs[len(parent.Childs)-1]
			parent.Childs[len(parent.Childs)-1] = nil
			parent.Childs = parent.Childs[:len(parent.Childs)-1]
			// garbage collector should work here even for directories
		}
	}

	// clear removed
	t.Removed = nil
}

func PrintDir(depth int, dir *Node) {
	fmt.Printf("%s├── %s\n", strings.Repeat("│   ", depth), path.Base(dir.Address) + "/")
	for _, c := range dir.Childs {
		if c.Removed {
			continue
		}

		if c.IsDirectory {
			PrintDir(depth + 1, c)
		} else {
			fmt.Printf("%s├── %s\n", strings.Repeat("│   ", depth + 1), path.Base(c.Address))
		}
	}
}

func CleanAddress(address string) (string, bool) {
	matched, _ := regexp.MatchString(`[a-zA-Z0-9/_\-.]+`, address)

	cleaned := path.Clean(address)

	//fmt.Printf("Matched: %s, %v\n", address, matched)

	if cleaned[0] == '/' {
		cleaned = cleaned[1:]
	}

	return cleaned, matched
}

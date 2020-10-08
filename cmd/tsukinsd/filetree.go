package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"path"
	"strings"
)

type Tree struct {
	Nodes map[string]*Node
}


type Node struct {
	Address string
	IsDirectory bool
	Childs []*Node
	Parent string
	Removed bool
}

func InitTree() *Tree {
	root := &Node{Address: ".", IsDirectory: true, Childs: make([]*Node, 0), Parent: ""}
	tree := &Tree{map[string]*Node{".": root}}

	return tree
}

func (t *Tree) String() string {
	return fmt.Sprintf("Tree{Nodes: %v}", t.Nodes)
}

func (node *Node) String() string {
	return fmt.Sprintf("Node{Address: %q, IsDirectory: %v, Childs: %v, Parent: %q}", node.Address, node.IsDirectory, node.Childs, node.Parent)
}

func (t *Tree) CreateFile(fileName string) error {
	fileName = CleanAddress(fileName)

	_, fileExists := t.Nodes[fileName]

	if fileExists {
		return fmt.Errorf("The file already exists")
	}


	dirPath := path.Dir(fileName)
	dir, dirExists := t.Nodes[dirPath]

	if !dirExists {
		return fmt.Errorf("The directory does not exist")
	}

	newFile := &Node{
		Address: fileName,
		IsDirectory: false,
		Childs: nil,
		Parent: dir.Address,
	}

	dir.Childs = append(dir.Childs, newFile)
	t.Nodes[fileName] = newFile

	return nil
}

func (t *Tree) RemoveFile(address string) error {
	address = CleanAddress(address)
	exists, isDirectory := t.PathExists(address)
	if !exists {
		return fmt.Errorf("file does not exist")
	} else if isDirectory {
		return fmt.Errorf("cannot remove directory")
	}

	t.Nodes[address].Removed = true // lazy removing
	delete(t.Nodes, address)
	return nil
}

func (t *Tree) CreateDirectory(address string) error {
	address = CleanAddress(address)
	exists, _ := t.PathExists(address)

	if exists {
		return fmt.Errorf("the path already exists")
	}

	dirPath := path.Dir(address)
	dirExists := t.DirectoryExists(dirPath)
	if !dirExists {
		return fmt.Errorf("the parent directory (%s) does not exist", dirPath)
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

	return nil
}

func (t *Tree) RemoveDirectory(address string) error {
	address = CleanAddress(address)
	if !t.DirectoryExists(address) {
		return fmt.Errorf("directory does not exist")
	}

	node, _ := t.GetNodeByAddress(address)
	node.Removed = true // lazy removing; will be removed later
	delete(t.Nodes, address)

	return nil
}

func (t *Tree) GetNodeByAddress(address string) (*Node, bool) {
	address = CleanAddress(address)
	node, ok := t.Nodes[address]

	return node, ok
}

func (t *Tree) CD(address string) error {
	address = CleanAddress(address)
	exists, isDirectory := t.PathExists(address)

	if !exists {
		return fmt.Errorf("directory does not exist")
	} else if !isDirectory {
		return fmt.Errorf("not a directory")
	}
	return nil
}

func (t *Tree) CopyFile(fileToCopy string, copyTo string) error {
	fileToCopy = CleanAddress(fileToCopy)
	copyTo = CleanAddress(copyTo)

	var fullFilePath string

	fileToCopyExists, fileToCopyIsDirectory := t.PathExists(fileToCopy)

	if !fileToCopyExists {
		return fmt.Errorf("file does not exist")
	} else if fileToCopyIsDirectory {
		return fmt.Errorf("cannot copy directory")
	}

	if t.DirectoryExists(copyTo) {
		fullFilePath = path.Join(copyTo, path.Base(fileToCopy))
	} else if t.FileExists(copyTo) {
		return fmt.Errorf("the file already exists")
	} else {
		fullFilePath = copyTo
	}

	if t.FileExists(fullFilePath) {
		return fmt.Errorf("the file already exists")
	}

	parentDir := t.Nodes[path.Dir(fullFilePath)]

	copiedFile := *t.Nodes[fileToCopy]
	copiedFile.Parent = parentDir.Address
	copiedFile.Address = fullFilePath

	t.Nodes[fullFilePath] = &copiedFile
	parentDir.Childs = append(parentDir.Childs, &copiedFile)

	return nil
}

func (t *Tree) MoveFile(fileToMove string, moveTo string) error {
	fileToMove = CleanAddress(fileToMove)
	moveTo = CleanAddress(moveTo)
	err := t.CopyFile(fileToMove, moveTo)

	if err != nil {
		return fmt.Errorf("impossible to move file, %e", err)
	}

	_ = t.RemoveFile(fileToMove)

	return nil
}

func (t *Tree) LS(address string) ([]string, error) {
	address = CleanAddress(address)
	if !t.DirectoryExists(address) {
		return nil, fmt.Errorf("directory does not exist")
	}

	dir, _ := t.GetNodeByAddress(address)
	var list []string

	for _, node := range dir.Childs {
		name := path.Base(node.Address)
		if node.IsDirectory {
			name += "/"
		}
		list = append(list, name)
	}

	return list, nil
}


func (t *Tree) PathExists(address string) (exists bool, isDirectory bool) {
	address = CleanAddress(address)
	node, ok := t.Nodes[address]

	if ok {
		return ok && !t.Nodes[address].Removed && t.ParentsExist(node), t.Nodes[address].IsDirectory
	}
	return ok, false
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


func (t *Tree) SaveTree(saveTo string) bool {
	file, _ := os.Create(saveTo)
	defer file.Close()
	encoder := gob.NewEncoder(file)

	encoder.Encode(t)
	return true
}

func (t *Tree) PrintTreeStruct() {
	PrintDir(0, t.Nodes["."])
}

func PrintDir(depth int, dir *Node) {
	fmt.Printf("d %s├── %s\n", strings.Repeat("│   ", depth), path.Base(dir.Address))
	for _, c := range dir.Childs {
		if c.Removed {
			continue
		}

		if c.IsDirectory {
			PrintDir(depth + 1, c)
		} else {
			fmt.Printf("f %s├── %s\n", strings.Repeat("│   ", depth + 1), path.Base(c.Address))
		}
	}
}

func CleanAddress(address string) string {
	cleaned := path.Clean(address)

	if cleaned[0] == '/' {
		cleaned = cleaned[1:]
	}

	return cleaned
}

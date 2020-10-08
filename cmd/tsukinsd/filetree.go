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

func initTree() *Tree {
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
	fileName = path.Clean(fileName)

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
	address = path.Clean(address)
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
	address = path.Clean(address)
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

func (t *Tree) GetNodeByAddress(address string) (*Node, bool) {
	address = path.Clean(address)
	node, ok := t.Nodes[address]

	return node, ok
}

func (t *Tree) CD(address string) error {
	address = path.Clean(address)
	exists, isDirectory := t.PathExists(address)

	if !exists {
		return fmt.Errorf("directory does not exist")
	} else if !isDirectory {
		return fmt.Errorf("not a directory")
	}
	return nil
}

func (t *Tree) CopyFile(fileToCopy string, copyTo string) error {
	fileToCopy = path.Clean(fileToCopy)
	copyTo = path.Clean(copyTo)

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


func (t *Tree) PathExists(address string) (exists bool, isDirectory bool) {
	address = path.Clean(address)
	_, ok := t.Nodes[address]
	if ok {
		return ok && !t.Nodes[address].Removed, t.Nodes[address].IsDirectory
	}
	return ok, false
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


// func LoadTree() *Tree {
// 	tree := Tree()
// }
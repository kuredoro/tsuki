package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"path"
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

func (t *Tree) CopyFile(oldAddress string, newAddress string) error {
	oldAddress = path.Clean(oldAddress)
	newBase := path.Base(newAddress)
	newAddress = path.Dir(newAddress)
	exists, isDirectory := t.PathExists(oldAddress)

	if !exists {
		return fmt.Errorf("file does not exist")
	} else if isDirectory {
		return fmt.Errorf("cannot copy directory")
	}

	file := *t.Nodes[oldAddress]
	file.Parent = newAddress
	file.Address = path.Join(newAddress, newBase)

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


// func LoadTree() *Tree {
// 	tree := Tree()
// }
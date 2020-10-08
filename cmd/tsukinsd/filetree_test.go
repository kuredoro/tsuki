package main

import (
	"fmt"
	"reflect"
	"testing"
)

func TestTree_CreateFile(t *testing.T) {
	cases := []struct {
		filename string
		want     Node
	}{
		{"hello.txt", Node{"hello.txt", false, nil, ".", false}},
		{".lala.txt", Node{".lala.txt", false, nil, ".", false}},
		{"ohmydog.tar.gz", Node{"ohmydog.tar.gz", false, nil, ".", false}},
	}
	for _, test := range cases {
		t.Run(fmt.Sprintf("Creating %v", test.filename),
			func(t *testing.T) {
				tree := initTree()
				tree.CreateFile(test.filename)
				got, _ := tree.GetNodeByAddress(test.filename)

				if !reflect.DeepEqual(*got, test.want) {
					t.Errorf("got %v, want %v", got, test.want)
				}
			})
	}
}

func TestTree_CreateDirectory(t *testing.T) {
	cases := []struct {
		filename string
		want     *Node
	}{
		{"hello.txt", &Node{"hello.txt", true, nil, ".", false}},
		{".lala.txt", &Node{".lala.txt", true, nil, ".", false}},
		{"ohmydog.tar.gz", &Node{"ohmydog.tar.gz", true, nil, ".", false}},
		{"notexist/ohmydog.tar.gz", nil},
	}
	for _, test := range cases {
		t.Run(fmt.Sprintf("Creating %v", test.filename),
			func(t *testing.T) {
				tree := initTree()
				tree.CreateDirectory(test.filename)
				got, _ := tree.GetNodeByAddress(test.filename)

				if test.want == nil || got == nil {
					if test.want != got {
						t.Errorf("Non nil")
					}
				} else if !reflect.DeepEqual(*got, *test.want) {
					t.Errorf("got %v, want %v", got, test.want)
				}
			})
	}
}

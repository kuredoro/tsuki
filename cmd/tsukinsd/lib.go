package main

import (
	"crypto/rand"
	"github.com/google/uuid"
)

func generateToken() string {
	tokenBytes := make([]byte, 16)
	rand.Read(tokenBytes)
	token, _ := uuid.FromBytes(tokenBytes)

	return token.String()
}

func Min(x int, y int) int {
	if x < y {
		return x
	} else {
		return y
	}
}

func Abs(x int) int {
	if x < 0 {
		return -x
	} else {
		return x
	}

}

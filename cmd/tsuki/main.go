package main

import (
    "fmt"

    "github.com/kureduro/tsuki"
)

func main() {
    var a, b int
    fmt.Print("Please, enter two numbers.\na=")
    fmt.Scanf("%d", &a)
    fmt.Print("b=")
    fmt.Scanf("%d", &b)

    fmt.Printf("a+b=%d\nBye.", tsuki.Sum(a, b))
}

package tsuki_test

import (
	"fmt"
	"testing"

    "github.com/kureduro/tsuki"
)

func TestSum(t *testing.T) {
    cases := []struct {
        a, b int
        want int
    }{
        { 0, 0, 0 },
        { 1, 0, 1 },
        { 0, 1, 1 },
        { -100, 0, -100 },
        { 0, -99, -99 },
        { 2, 2, 4 },
        { 2, -2, 0 },
        { -2, 3, 1 },
        { -4, -3, -7 },
    }

    for _, test := range cases {
        t.Run(fmt.Sprintf("%v+%v", test.a, test.b),
        func(t *testing.T) {
            got := tsuki.Sum(test.a, test.b)

            if got != test.want {
                t.Errorf("got %d, want %d", got, test.want)
            }
        })
    }
}

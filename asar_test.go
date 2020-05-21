package asar // import "github.com/jaygooby/goasar"

import (
	"io/ioutil"
	"testing"
)

func TestBuilder_empty(t *testing.T) {
	var builder Builder
	if _, err := builder.Root().EncodeTo(ioutil.Discard); err != nil {
		t.Fatal(err)
	}
}

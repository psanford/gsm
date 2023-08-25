package mms

import (
	"fmt"
	"os"
	"testing"
)

func TestMms(t *testing.T) {
	packet, err := os.ReadFile("../examples/mms.apple-with-attachment")
	if err != nil {
		t.Fatal(err)
	}

	msg, err := Unmarshal(packet)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("msg: %+v\n", msg)
}

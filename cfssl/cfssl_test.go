package cfssl

import (
	"testing"
)

func TestSigne(t *testing.T) {
	signe, err := Signe("/tmp/ca_test", "192.168.104.126", "www.test.com")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(signe)
}

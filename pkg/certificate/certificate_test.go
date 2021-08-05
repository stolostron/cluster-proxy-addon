package certificate

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

func TestCreateCA(t *testing.T) {
	caKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	_, err := createCABytes("test_cn", caKey)
	if err != nil {
		t.Fatal("create ca failed", err)
	}
}

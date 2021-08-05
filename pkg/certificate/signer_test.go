package certificate

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	certificatesv1 "k8s.io/api/certificates/v1"
	"testing"
)

func createCSR() []byte {
	keys, _ := rsa.GenerateKey(rand.Reader, 2048)
	var csrTemplate = x509.CertificateRequest{
		Subject: pkix.Name{
			Country: []string{"US"},
		},
		SignatureAlgorithm: x509.SHA512WithRSA,
	}
	csrCertificate, _ := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, keys)
	csr := pem.EncodeToMemory(&pem.Block{
		Type: "CERTIFICATE REQUEST", Bytes: csrCertificate,
	})
	return csr
}

func TestSignCSR(t *testing.T) {
	csr := &certificatesv1.CertificateSigningRequest{
		Spec: certificatesv1.CertificateSigningRequestSpec{
			Request: createCSR(),
			Usages:  []certificatesv1.KeyUsage{certificatesv1.UsageCertSign, certificatesv1.UsageClientAuth},
		},
	}

	caKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	caBytes, _ := createCABytes("test-cn", caKey)
	caCerts, _ := x509.ParseCertificates(caBytes.certBytes)

	if SignCSR(csr, caCerts[0], caKey) == nil {
		t.Fatal("Failed to sign CSR")
	}
}

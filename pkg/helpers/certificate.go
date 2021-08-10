package helpers

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/cloudflare/cfssl/config"
	"github.com/cloudflare/cfssl/signer"
	"github.com/cloudflare/cfssl/signer/local"
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
)

func GetCert(caSecret *corev1.Secret) (*x509.Certificate, *rsa.PrivateKey, error) {
	tlsCrt, ok := caSecret.Data["tls.crt"]
	if !ok {
		return nil, nil, fmt.Errorf("no tls.crt in caSecret Data")
	}

	tlsKey, ok := caSecret.Data["tls.key"]
	if !ok {
		return nil, nil, fmt.Errorf("no tls.key in caSecret Data")
	}

	blockTlsCrt, _ := pem.Decode(tlsCrt) // note: the second return value is not error for pem.Decode; it's ok to omit it.
	certs, err := x509.ParseCertificates(blockTlsCrt.Bytes)
	if err != nil {
		return nil, nil, err
	}

	blockTlsKey, _ := pem.Decode(tlsKey)
	key, err := x509.ParsePKCS1PrivateKey(blockTlsKey.Bytes)
	if err != nil {
		return nil, nil, err
	}
	return certs[0], key, nil
}

func SignCSR(csr *certificatesv1.CertificateSigningRequest, caCert *x509.Certificate, caKey *rsa.PrivateKey) ([]byte, error) {
	var usages []string
	for _, usage := range csr.Spec.Usages {
		usages = append(usages, string(usage))
	}

	certExpiryDuration := 365 * 24 * time.Hour
	durationUntilExpiry := time.Until(caCert.NotAfter)
	if durationUntilExpiry <= 0 {
		return nil, fmt.Errorf("signer has expired, expired time: %v", caCert.NotAfter)
	}
	if durationUntilExpiry < certExpiryDuration {
		certExpiryDuration = durationUntilExpiry
	}
	policy := &config.Signing{
		Default: &config.SigningProfile{
			Usage:        usages,
			Expiry:       certExpiryDuration,
			ExpiryString: certExpiryDuration.String(),
		},
	}

	cfs, err := local.NewSigner(caKey, caCert, signer.DefaultSigAlgo(caKey), policy)
	if err != nil {
		return nil, err
	}
	signedCert, err := cfs.Sign(signer.SignRequest{
		Request: string(csr.Spec.Request),
	})
	if err != nil {
		return nil, err
	}
	return signedCert, nil
}

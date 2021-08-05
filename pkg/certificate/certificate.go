package certificate

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
	"math/big"
	"net"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var (
	serialNumberLimit = new(big.Int).Lsh(big.NewInt(1), 128)
)

func GetCert(c corev1listers.SecretLister, name, namespace string) (*x509.Certificate, *rsa.PrivateKey, error) {
	caSecret, err := c.Secrets(namespace).Get(name)
	if err != nil {
		return nil, nil, err
	}

	block1, _ := pem.Decode(caSecret.Data["tls.crt"])
	certs, err := x509.ParseCertificates(block1.Bytes)
	if err != nil {
		return nil, nil, err
	}

	block2, _ := pem.Decode(caSecret.Data["tls.key"])
	key, err := x509.ParsePKCS1PrivateKey(block2.Bytes)
	if err != nil {
		return nil, nil, err
	}
	return certs[0], key, nil
}

func createCABytes(cn string, caKey *rsa.PrivateKey) (c *crtPariBytes, err error) {
	sn, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	ca := &x509.Certificate{
		SerialNumber: sn,
		Subject: pkix.Name{
			Organization: []string{"Red Hat, Inc."},
			Country:      []string{"US"},
			CommonName:   cn,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365 * 5),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caCertBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caKey.PublicKey, caKey)
	if err != nil {
		klog.ErrorS(err, "Failed to create certificate", "cn", cn)
		return nil, err
	}
	caKeyBytes := x509.MarshalPKCS1PrivateKey(caKey)
	return &crtPariBytes{
		caKeyBytes, caCertBytes,
	}, nil
}

func createSecret(ctx context.Context, c client.Client, caCertBytes []byte, cb *crtPariBytes, name, namespace string) (err error) {
	certPEM, keyPEM := cb.pemEncode()
	crtSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"ca.crt":  caCertBytes,
			"tls.crt": certPEM.Bytes(),
			"tls.key": keyPEM.Bytes(),
		},
	}
	return c.Create(ctx, crtSecret)
}

func updateSecret(ctx context.Context, c client.Client, caCertBytes []byte, crtSecret *corev1.Secret, cb *crtPariBytes) (err error) {
	certPEM, keyPEM := cb.pemEncode()
	crtSecret.Data["ca.crt"] = caCertBytes
	crtSecret.Data["tls.crt"] = certPEM.Bytes()
	crtSecret.Data["tls.key"] = keyPEM.Bytes()
	return c.Update(ctx, crtSecret)
}

func createCertificate(cn string, dns []string, ips []net.IP,
	caCert *x509.Certificate, caKey *rsa.PrivateKey, key *rsa.PrivateKey) (*crtPariBytes, error) {
	sn, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	cert := &x509.Certificate{
		SerialNumber: sn,
		Subject: pkix.Name{
			Organization: []string{"Red Hat, Inc."},
			Country:      []string{"US"},
			CommonName:   cn,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(time.Hour * 24 * 365),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		DNSNames:    dns,
		IPAddresses: ips,
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, cert, caCert, &key.PublicKey, caKey)
	if err != nil {
		return nil, err
	}
	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	return &crtPariBytes{
		keyBytes, caBytes,
	}, nil
}

type crtPariBytes struct {
	keyBytes  []byte
	certBytes []byte
}

func (c *crtPariBytes) pemEncode() (*bytes.Buffer, *bytes.Buffer) {
	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: c.certBytes,
	})

	keyPEM := new(bytes.Buffer)
	pem.Encode(keyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: c.keyBytes,
	})

	return certPEM, keyPEM
}

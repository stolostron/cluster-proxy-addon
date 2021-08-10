package helpers

import (
	"crypto/rsa"
	"crypto/x509"
	"reflect"
	"testing"

	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestGetCert(t *testing.T) {
	type args struct {
		caSecret *corev1.Secret
	}
	tests := []struct {
		name    string
		args    args
		want    *x509.Certificate
		want1   *rsa.PrivateKey
		wantErr bool
	}{
		// TODO add unit tests
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := GetCert(tt.args.caSecret)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCert() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("GetCert() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestSignCSR(t *testing.T) {
	type args struct {
		csr    *certificatesv1.CertificateSigningRequest
		caCert *x509.Certificate
		caKey  *rsa.PrivateKey
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SignCSR(tt.args.csr, tt.args.caCert, tt.args.caKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("SignCSR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SignCSR() = %v, want %v", got, tt.want)
			}
		})
	}
}

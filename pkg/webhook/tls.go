package webhook

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const tlsSecretName = "logcloak-tls"

// TLSBundle holds PEM-encoded cert material.
type TLSBundle struct {
	CACert  []byte
	TLSCert []byte
	TLSKey  []byte
}

// EnsureTLS reads TLS cert from the logcloak-tls Secret; generates a new one if absent.
// Returns a *tls.Config ready for the HTTPS server and the CA cert for patching the webhook.
func EnsureTLS(ctx context.Context, kube kubernetes.Interface, namespace, serviceName string) (*tls.Config, []byte, error) {
	secret, err := kube.CoreV1().Secrets(namespace).Get(ctx, tlsSecretName, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return nil, nil, fmt.Errorf("get tls secret: %w", err)
	}

	var bundle TLSBundle
	if err == nil && len(secret.Data["tls.crt"]) > 0 {
		bundle = TLSBundle{
			CACert:  secret.Data["ca.crt"],
			TLSCert: secret.Data["tls.crt"],
			TLSKey:  secret.Data["tls.key"],
		}
	} else {
		bundle, err = generateSelfSigned(serviceName, namespace)
		if err != nil {
			return nil, nil, fmt.Errorf("generate cert: %w", err)
		}
		if storeErr := storeTLSSecret(ctx, kube, namespace, bundle); storeErr != nil {
			return nil, nil, fmt.Errorf("store tls secret: %w", storeErr)
		}
	}

	cert, err := tls.X509KeyPair(bundle.TLSCert, bundle.TLSKey)
	if err != nil {
		return nil, nil, fmt.Errorf("load key pair: %w", err)
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
	return cfg, bundle.CACert, nil
}

func generateSelfSigned(serviceName, namespace string) (TLSBundle, error) {
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return TLSBundle{}, err
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "logcloak-ca"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return TLSBundle{}, err
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		return TLSBundle{}, err
	}

	srvKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return TLSBundle{}, err
	}
	dnsNames := []string{
		serviceName,
		fmt.Sprintf("%s.%s", serviceName, namespace),
		fmt.Sprintf("%s.%s.svc", serviceName, namespace),
		fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, namespace),
	}
	srvTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: dnsNames[2]},
		DNSNames:     dnsNames,
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	srvDER, err := x509.CreateCertificate(rand.Reader, srvTemplate, caCert, &srvKey.PublicKey, caKey)
	if err != nil {
		return TLSBundle{}, err
	}

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srvDER})
	keyDER, _ := x509.MarshalECPrivateKey(srvKey)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return TLSBundle{CACert: caPEM, TLSCert: certPEM, TLSKey: keyPEM}, nil
}

func storeTLSSecret(ctx context.Context, kube kubernetes.Interface, namespace string, b TLSBundle) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: tlsSecretName, Namespace: namespace},
		Data: map[string][]byte{
			"ca.crt":  b.CACert,
			"tls.crt": b.TLSCert,
			"tls.key": b.TLSKey,
		},
	}
	_, err := kube.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		_, err = kube.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	}
	return err
}

// PatchWebhookCABundle updates the caBundle field in a MutatingWebhookConfiguration.
func PatchWebhookCABundle(ctx context.Context, kube kubernetes.Interface, webhookName string, caCert []byte) error {
	caB64 := base64.StdEncoding.EncodeToString(caCert)
	patch := fmt.Sprintf(`[{"op":"replace","path":"/webhooks/0/clientConfig/caBundle","value":%q}]`, caB64)
	_, err := kube.AdmissionregistrationV1().MutatingWebhookConfigurations().Patch(
		ctx, webhookName, types.JSONPatchType, []byte(patch), metav1.PatchOptions{},
	)
	return err
}

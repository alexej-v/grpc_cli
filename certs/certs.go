package certs

import "crypto/x509"

import "crypto/tls"

import "io/ioutil"

import "errors"

type Certs interface {
	CACert() *x509.CertPool
	Cert() tls.Certificate
	// Set(caCertPath, cert, certKey string) error
	HasCaCert() bool
	HasCert() bool
}

type certs struct {
	caCert    *x509.CertPool
	cert      tls.Certificate
	hasCaCert bool
	hasCert   bool
}

func (c *certs) set(caCertPath, certFile, certFileKey string) error {
	switch {
	case caCertPath != "":
		caCertBody, err := ioutil.ReadFile(caCertPath)
		if err != nil {
			return err
		}
		cp := x509.NewCertPool()
		if !cp.AppendCertsFromPEM(caCertBody) {
			return errors.New("failed to append CACert to cert pool")
		}
		c.caCert = cp
		c.hasCaCert = true
	case certFile != "" && certFileKey != "":
		certificate, err := tls.LoadX509KeyPair(certFile, certFileKey)
		if err != nil {
			return err
		}
		c.cert = certificate
		c.hasCert = true
	}
	return nil
}

func (c *certs) HasCaCert() bool {
	return c.hasCaCert
}

func (c *certs) HasCert() bool {
	return c.hasCert
}

func (c *certs) CACert() *x509.CertPool {
	return c.caCert
}

func (c *certs) Cert() tls.Certificate {
	return c.cert
}

func Define(caCertPath, certFile, certFileKey string) (Certs, error) {
	newCerts := certs{}
	if err := newCerts.set(caCertPath, certFile, certFileKey); err != nil {
		return nil, err
	}
	return &newCerts, nil
}

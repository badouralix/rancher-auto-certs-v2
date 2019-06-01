package main

import (
	"crypto"
	"fmt"

	"github.com/go-acme/lego/certcrypto"
	"github.com/go-acme/lego/certificate"
	"github.com/go-acme/lego/challenge"
	"github.com/go-acme/lego/lego"
	"github.com/go-acme/lego/providers/dns"
	"github.com/go-acme/lego/registration"
)

// We need a user or account type that implements registration.User
type letsencryptUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *letsencryptUser) GetEmail() string {
	return u.Email
}
func (u letsencryptUser) GetRegistration() *registration.Resource {
	return u.Registration
}
func (u *letsencryptUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

// We use a manager to cache letsencrypt clients and their users
type letsencryptManager struct {
	clients map[accountConfig]*lego.Client
}

func newLEManager() (*letsencryptManager, error) {
	return &letsencryptManager{
		clients: make(map[accountConfig]*lego.Client),
	}, nil
}

func (lm *letsencryptManager) GenCertificate(cc *certConfig) (*certificate.Resource, error) {
	// Dispatch supported challenges to dedicated methods
	if cc.Challenge == "dns-01" {
		return lm.GenCertificateWithDNS01(cc)
	}

	// Return an error if no dispatch occurred
	return nil, fmt.Errorf("letsencrypt: challenge %s is not supported", cc.Challenge)
}

func (lm *letsencryptManager) GenCertificateWithDNS01(cc *certConfig) (*certificate.Resource, error) {
	client, err := lm.GetClient(cc)
	if err != nil {
		return nil, err
	}

	provider, err := dns.NewDNSChallengeProviderByName(cc.Provider)
	if err != nil {
		return nil, err
	}

	err = client.Challenge.SetDNS01Provider(provider)
	if err != nil {
		return nil, err
	}
	// As the client is cached, we need to clean it before reusing it
	// This specific challenge may fail for another certificate, and keeping it is an antipattern
	// See https://github.com/go-acme/lego/issues/842
	// TODO: maybe only cache the user instead of the client ( note that the user registration depends on the client )
	defer client.Challenge.Remove(challenge.DNS01)

	request := certificate.ObtainRequest{
		Domains: cc.Domains,
		Bundle:  true,
	}
	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		return nil, err
	}

	// Each certificate comes back with the cert bytes, the bytes of the client's
	// private key, and a certificate URL

	return certificates, nil
}

func (lm *letsencryptManager) GetClient(cc *certConfig) (*lego.Client, error) {
	ac := cc.getAccountConfig()

	if client, ok := lm.clients[ac]; ok {
		return client, nil
	}

	// If not ok, no client has been generated for this specific accountConfig yet

	// Create a user. New accounts need an email and private key to start.
	privateKey, err := cc.getPrivateKey()
	if err != nil {
		return nil, err
	}

	user := letsencryptUser{
		Email: cc.AccountEmail,
		key:   privateKey,
	}

	config := lego.NewConfig(&user)

	// This CA URL is configured for a local dev instance of Boulder running in Docker in a VM.
	config.CADirURL = cc.CA

	// The default key type is RSA2048, but we provide the ability to override it
	if cc.KeyType == "EC256" {
		config.Certificate.KeyType = certcrypto.EC256
	} else if cc.KeyType == "EC384" {
		config.Certificate.KeyType = certcrypto.EC384
	} else if cc.KeyType == "RSA2048" {
		config.Certificate.KeyType = certcrypto.RSA2048
	} else if cc.KeyType == "RSA4096" {
		config.Certificate.KeyType = certcrypto.RSA4096
	} else if cc.KeyType == "RSA8192" {
		config.Certificate.KeyType = certcrypto.RSA8192
	} else {
		return nil, fmt.Errorf("letsencrypt: unsupported key type %s", cc.KeyType)
	}

	// A client facilitates communication with the CA server.
	client, err := lego.NewClient(config)
	if err != nil {
		// lego.NewClient errors are not formatted
		return nil, fmt.Errorf("letsencrypt: %v", err)
	}

	// Client is cached
	lm.clients[ac] = client

	// New users will need to register
	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return nil, err
	}
	user.Registration = reg

	// Client is returned
	return client, nil
}

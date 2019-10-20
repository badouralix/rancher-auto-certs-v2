package main

import (
	"crypto"
	"fmt"
	"os"
	"sync"

	"github.com/go-acme/lego/certcrypto"
	"github.com/go-acme/lego/certificate"
	"github.com/go-acme/lego/lego"
	"github.com/go-acme/lego/log"
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
	clients     map[string]*lego.Client
	environment sync.Mutex
}

func newLEManager() (*letsencryptManager, error) {
	return &letsencryptManager{
		clients: make(map[string]*lego.Client),
	}, nil
}

func (lm *letsencryptManager) GenCertificate(cc *certConfig) (*certificate.Resource, error) {
	client, err := lm.GetClient(cc)
	if err != nil {
		return nil, err
	}

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
	// Try to retrieve client from cache
	if client, ok := lm.clients[cc.Name]; ok {
		return client, nil
	}

	// If not ok, no client has been generated for this cert yet

	// Create user: new accounts need an email and private key to start
	privateKey, err := cc.getPrivateKey()
	if err != nil {
		return nil, err
	}

	user := letsencryptUser{
		Email: cc.AccountEmail,
		key:   privateKey,
	}

	// Create config for user
	config := lego.NewConfig(&user)

	// The CA URL can be overridden for development purpose
	config.CADirURL = cc.CA

	// The default key type is RSA2048, but we provide the ability to override it
	switch cc.KeyType {
	case "EC256":
		config.Certificate.KeyType = certcrypto.EC256
	case "EC384":
		config.Certificate.KeyType = certcrypto.EC384
	case "RSA2048":
		config.Certificate.KeyType = certcrypto.RSA2048
	case "RSA4096":
		config.Certificate.KeyType = certcrypto.RSA4096
	case "RSA8192":
		config.Certificate.KeyType = certcrypto.RSA8192
	default:
		return nil, fmt.Errorf("letsencrypt: unsupported key type %s", cc.KeyType)
	}

	// A client facilitates communication with the CA server.
	client, err := lego.NewClient(config)
	if err != nil {
		// lego.NewClient errors are not formatted
		return nil, fmt.Errorf("letsencrypt: %v", err)
	}

	// New users will need to register
	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return nil, err
	}
	user.Registration = reg

	// Register challenges

	// Lock mutex as we are going to update the environment
	lm.environment.Lock()
	defer lm.environment.Unlock()

	// Dispatch supported challenges to dedicated methods
	switch cc.Challenge {
	case "dns-01":
		// Yolo patch the environment with config credentials
		for envvar, value := range cc.Env {
			err := os.Setenv(envvar, value)
			defer os.Unsetenv(envvar)
			if err != nil {
				return nil, err
			}
		}

		// Build provider
		provider, err := dns.NewDNSChallengeProviderByName(cc.Provider)
		if err != nil {
			return nil, err
		}

		// Register provider
		err = client.Challenge.SetDNS01Provider(provider)
		if err != nil {
			return nil, err
		}
	default:
		// Return an error if no dispatch occurred
		return nil, fmt.Errorf("letsencrypt: challenge %s is not supported", cc.Challenge)
	}

	// Client is cached
	log.Infof("letsencrypt: A new client has been created for %s", cc.Name)
	lm.clients[cc.Name] = client

	// Client is returned
	return client, nil
}

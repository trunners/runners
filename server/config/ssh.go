package config

import (
	"crypto/subtle"
	"fmt"

	"golang.org/x/crypto/ssh"

	"github.com/trunners/runners/keys"
)

func serverConfig(authorizedKeys []ssh.PublicKey) *ssh.ServerConfig {
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			for _, k := range authorizedKeys {
				if keysEqual(k, key) {
					return &ssh.Permissions{}, nil
				}
			}

			return nil, fmt.Errorf("unknown public key for %q", c.User())
		},
	}
	config.AddHostKey(keys.ServerPrivateKey())

	return config
}

func clientConfig() *ssh.ClientConfig {
	config := &ssh.ClientConfig{
		User: "trev",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(keys.ServerPrivateKey()),
		},
		HostKeyCallback: ssh.FixedHostKey(keys.ClientPublicKey()),
	}

	return config
}

func keysEqual(ak, bk ssh.PublicKey) bool {
	// avoid panic if one of the keys is nil, return false instead
	if ak == nil || bk == nil {
		return false
	}

	a := ak.Marshal()
	b := bk.Marshal()
	return (len(a) == len(b) && subtle.ConstantTimeCompare(a, b) == 1)
}

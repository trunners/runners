package config

import (
	"crypto/subtle"
	"fmt"

	"golang.org/x/crypto/ssh"

	"github.com/trunners/runners/keys"
)

func serverConfig() *ssh.ServerConfig {
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if keysEqual(key, keys.ServerPublicKey()) {
				return &ssh.Permissions{}, nil
			}

			return nil, fmt.Errorf("unknown public key for %q", c.User())
		},
	}
	config.AddHostKey(keys.ClientPrivateKey())

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

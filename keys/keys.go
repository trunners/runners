package keys

import (
	_ "embed"

	"golang.org/x/crypto/ssh"
)

//go:embed client/id_ed25519
var clientPrivateKey []byte

//go:embed client/id_ed25519.pub
var clientPublicKey []byte

//go:embed server/id_ed25519
var serverPrivateKey []byte

//go:embed server/id_ed25519.pub
var serverPublicKey []byte

func ClientPrivateKey() ssh.Signer {
	key, err := ssh.ParsePrivateKey(clientPrivateKey)
	if err != nil {
		panic(err)
	}

	return key
}

func ClientPublicKey() ssh.PublicKey {
	key, _, _, _, err := ssh.ParseAuthorizedKey(clientPublicKey)
	if err != nil {
		panic(err)
	}

	return key
}

func ServerPrivateKey() ssh.Signer {
	key, err := ssh.ParsePrivateKey(serverPrivateKey)
	if err != nil {
		panic(err)
	}

	return key
}

func ServerPublicKey() ssh.PublicKey {
	key, _, _, _, err := ssh.ParseAuthorizedKey(serverPublicKey)
	if err != nil {
		panic(err)
	}

	return key
}

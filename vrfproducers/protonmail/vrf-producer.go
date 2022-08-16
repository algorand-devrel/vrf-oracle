package protonmail

import (
	"log"

	"github.com/ProtonMail/go-ecvrf/ecvrf"
)

type ProtonMailVRFProducer struct {
	publicKey  []byte
	privateKey []byte
}

func New(publicKey []byte, privateKey []byte) *ProtonMailVRFProducer {
	return &ProtonMailVRFProducer{
		privateKey: privateKey,
		publicKey:  publicKey,
	}
}

func (v *ProtonMailVRFProducer) Prove(msg []byte) (vrf, proof []byte) {

	privateKey, err := ecvrf.NewPrivateKey(v.privateKey)
	if err != nil {
		log.Fatalf("Failed to proove message: %+v", err)
	}

	vrf, proof, err = privateKey.Prove(msg)
	if err != nil {
		log.Fatalf("Failed to proove message: %+v", err)
	}

	return vrf, proof
}

func (v *ProtonMailVRFProducer) Verify(msg, proof []byte) (verified bool) {

	publicKey, err := ecvrf.NewPublicKey(v.publicKey)
	if err != nil {
		log.Fatalf("Failed to verify: %+v", err)
	}

	verified, _, err = publicKey.Verify(msg, proof)
	if err != nil {
		log.Fatalf("Failed to verify: %+v", err)
	}

	return verified
}

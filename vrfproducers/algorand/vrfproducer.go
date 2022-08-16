package algorand

import (
	"github.com/algorand/go-algorand/crypto"
	"github.com/algorand/go-algorand/protocol"
)

type AlgorandVRFProducer struct {
	PublicKey  crypto.VrfPubkey
	PrivateKey crypto.VrfPrivkey
}

func New(publicKey, privateKey []byte) *AlgorandVRFProducer {
	privKey := [64]byte{}
	copy(privKey[:], privateKey[:])

	pubKey := [32]byte{}
	copy(pubKey[:], publicKey[:])

	return &AlgorandVRFProducer{
		PublicKey:  pubKey,
		PrivateKey: privKey,
	}
}

type Msg []byte

func (m Msg) ToBeHashed() (protocol.HashID, []byte) {
	return protocol.HashID(""), m[:]
}

func (avp *AlgorandVRFProducer) Prove(msg []byte) (vrf []byte, proof []byte) {
	vrfProof, _ := avp.PrivateKey.Prove(Msg(msg))
	vrfHash, _ := vrfProof.Hash()
	return vrfHash[:], vrfProof[:]
}

func (avp *AlgorandVRFProducer) Verify(msg, proof []byte) bool {
	var vrfProof crypto.VrfProof
	copy(vrfProof[:], proof[:])
	ok, _ := avp.PublicKey.Verify(vrfProof, Msg(msg))
	return ok
}

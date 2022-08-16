package vrfproducers

type VRFProducer interface {
	Prove(msg []byte) (vrf []byte, proof []byte)
	Verify(msg []byte, proof []byte) bool
}

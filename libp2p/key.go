package libp2p

import (
	"bytes"
	openpgpcrypto "github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/curltech/go-colla-core/crypto/openpgp"
	"github.com/libp2p/go-libp2p-core/crypto"
	pb "github.com/libp2p/go-libp2p-core/crypto/pb"
)

type OpenPGPPrivateKey struct {
	PrivateKey *openpgpcrypto.Key
	KeyType    pb.KeyType
}

// Bytes returns a serialized, storeable representation of this key
// DEPRECATED in favor of Marshal / Unmarshal
func (this *OpenPGPPrivateKey) Bytes() ([]byte, error) {
	return openpgp.BytePrivateKey(this.PrivateKey, nil), nil
}

// Equals checks whether two PubKeys are the same
func (this *OpenPGPPrivateKey) Equals(key crypto.Key) bool {
	bs1, _ := this.Bytes()
	bs2, err := key.Raw()
	if err != nil {
		panic(err)
	}

	return bytes.Equal(bs1, bs2)
}

// Raw returns the raw bytes of the key (not wrapped in the
// libp2p-crypto protobuf).
//
// This function is the inverse of {Priv,Pub}KeyUnmarshaler.
func (this *OpenPGPPrivateKey) Raw() ([]byte, error) {
	return openpgp.BytePrivateKey(this.PrivateKey, nil), nil
}

// Type returns the protobof key type.
func (this *OpenPGPPrivateKey) Type() pb.KeyType {
	return this.KeyType
}

// PrivateKey
// Cryptographically sign the given bytes
func (this *OpenPGPPrivateKey) Sign(plaintext []byte) ([]byte, error) {
	return openpgp.Sign(this.PrivateKey, nil, plaintext), nil
}

// Return a public key paired with this private key
func (this *OpenPGPPrivateKey) GetPublic() crypto.PubKey {
	pub := openpgp.GetPublicKey(this.PrivateKey)
	publicKey := &OpenPGPPublicKey{PublicKey: pub, KeyType: this.KeyType}

	return publicKey
}

type OpenPGPPublicKey struct {
	PublicKey *openpgpcrypto.Key
	KeyType   pb.KeyType
}

// Bytes returns a serialized, storeable representation of this key
// DEPRECATED in favor of Marshal / Unmarshal
func (this *OpenPGPPublicKey) Bytes() ([]byte, error) {
	return openpgp.BytePublicKey(this.PublicKey), nil
}

// Equals checks whether two PubKeys are the same
func (this *OpenPGPPublicKey) Equals(key crypto.Key) bool {
	bs1, _ := this.Bytes()
	bs2, _ := key.Raw()

	return bytes.Equal(bs1, bs2)
}

// Raw returns the raw bytes of the key (not wrapped in the
// libp2p-crypto protobuf).
//
// This function is the inverse of {Priv,Pub}KeyUnmarshaler.
func (this *OpenPGPPublicKey) Raw() ([]byte, error) {
	return openpgp.BytePublicKey(this.PublicKey), nil
}

// Type returns the protobof key type.
func (this *OpenPGPPublicKey) Type() pb.KeyType {
	return this.KeyType
}

// PublicKeys
// Verify that 'sig' is the signed hash of 'data'
func (this *OpenPGPPublicKey) Verify(data []byte, sig []byte) (bool, error) {
	return openpgp.Verify(this.PublicKey, data, sig), nil
}

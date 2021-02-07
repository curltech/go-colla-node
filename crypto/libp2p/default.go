package libp2p

import (
	"crypto/ed25519"
	"github.com/curltech/go-colla-core/crypto/std"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-core/util/security"
	libp2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"golang.org/x/crypto/openpgp/ecdh"
	"golang.org/x/crypto/rand"
)

func init() {

}

/**
 * 生成随机对称秘钥
 *
 * @param algorithm
 * @return
 * @throws NoSuchAlgorithmException
 */
func GenerateSecretKey(size int) (sessionKey interface{}) {

	return std.GenerateSecretKey(size)
}

/**
对对称密钥用对方公钥加密
*/
func WriteSecretKey(sessionKey []byte, publicKey *ecdh.PublicKey, password []byte) []byte {
	if password == nil {
		ciphertext := Encrypt(publicKey, sessionKey)

		return ciphertext
	} else {
		ciphertext := Encrypt(publicKey, sessionKey)
		ciphertext = Encrypt(password, ciphertext)

		return ciphertext
	}
}

/**
 * 生成指定字符串的对称密码
 */
func BuildSecretKey(keyValue string) string {
	return keyValue
}

func ReadSecretKey(keyPacket []byte, privateKey *ecdh.PrivateKey, password []byte) []byte {
	if password == nil {
		plaintext := Decrypt(privateKey, nil, keyPacket)

		return plaintext
	} else {
		ciphertext := Decrypt(password, nil, keyPacket)
		ciphertext = Decrypt(keyPacket, nil, ciphertext)

		return ciphertext
	}
}

/**
 * 从公钥字符串,Key,KeyRing,表示中还原公钥
 *
 *
 * @param keyValue
 * @return
 * @throws NoSuchAlgorithmException
 * @throws InvalidKeySpecException
 */
func LoadPublicKey(keyValue interface{}) *ecdh.PublicKey {
	pub := &ecdh.PublicKey{}
	if _, ok := keyValue.([]byte); ok {
		bs := keyValue.([]byte)
		message.Unmarshal(bs, pub)

		return pub
	}

	return nil
}

/**
 * 从证书中的私钥二进制字符串表示中还原私钥
 *
 *
 * @param keyValue
 * @return
 */
func LoadPrivateKey(keyValue interface{}, password string) *ecdh.PrivateKey {
	priv := &ecdh.PrivateKey{}
	if _, ok := keyValue.([]byte); ok {
		bs := keyValue.([]byte)
		message.Unmarshal(bs, priv)

		return priv
	}

	return nil
}

/**
 * 生成非对称的密钥对
 *
 * @return
 */
func GenerateKeyPair(keyType string, passphrase []byte, armored bool, name string, email string) interface{} {
	var keypair interface{}
	var err error

	if keyType == "RSA" {
		// RSA, Key struct
		keypair, _, err = libp2pcrypto.GenerateKeyPair(libp2pcrypto.RSA, 4096)
	} else if keyType == "Ed25519" || keyType == "x25519" {
		keypair, _, err = libp2pcrypto.GenerateKeyPair(libp2pcrypto.Ed25519, 0)
	} else if keyType == "ECDSA" {
		// Curve25519, Key struct
		keypair, _, err = libp2pcrypto.GenerateKeyPair(libp2pcrypto.ECDSA, 0)
	} else if keyType == "Secp256k1" {
		// Curve25519, Key struct
		keypair, _, err = libp2pcrypto.GenerateKeyPair(libp2pcrypto.Secp256k1, 0)
	}
	if err != nil {
		panic(err)
	}

	return keypair
}

func GetPrivateKey(keyPair *ecdh.PrivateKey, password []byte) *ecdh.PrivateKey {

	return keyPair
}

func GetPublicKey(keyPair *ecdh.PrivateKey) (key *ecdh.PublicKey) {
	return &keyPair.PublicKey
}

func BytePublicKey(publicKey *ecdh.PublicKey) []byte {
	bs, _ := message.Marshal(publicKey)

	return bs
}

func BytePrivateKey(privateKey *ecdh.PrivateKey, password []byte) []byte {
	bs, _ := message.Marshal(privateKey)

	return bs
}

/**
 * 非对称加密
 *
 * @param key  加密的密钥
 * @param data 待加密的明文数据
 * @return 加密后的数据
 * @throws EncryptException
 */
func Encrypt(keyValue interface{}, plaintext []byte) []byte {
	var (
		testCurveOID    = []byte{0x05, 0x2B, 0x81, 0x04, 0x00, 0x22} // MPI encoded oidCurveP384
		testFingerprint = make([]byte, 20)
	)

	if _, ok := keyValue.(*ecdh.PublicKey); ok {
		pub := keyValue.(*ecdh.PublicKey)
		password := []byte(security.RandString(32))
		vsG, m, err := ecdh.Encrypt(rand.Reader, pub, password, testCurveOID, testFingerprint)
		if err != nil {
			logger.Sugar.Errorf("error encrypting: %s", err)
		}
		cipher := std.EncryptSymmetrical(password, plaintext, "GCM")
		ciphertext := make([]byte, len(vsG)+len(m)+len(cipher))
		ciphertext = append(ciphertext, vsG...)
		ciphertext = append(ciphertext, m...)
		ciphertext = append(ciphertext, cipher...)

		return ciphertext
	} else if _, ok := keyValue.([]byte); ok {
		password := keyValue.([]byte)
		ciphertext := std.EncryptSymmetrical(password, plaintext, "GCM")

		return ciphertext
	}
	panic("")
}

/**
 * 非对称解密
 *
 * @param privkey 解密的密钥
 * @param ciphertext 已经加密的数据
 * @return 解密后的明文
 * @throws EncryptException
 */
func Decrypt(keyValue interface{}, passphrase []byte, ciphertext []byte) []byte {
	var (
		testCurveOID    = []byte{0x05, 0x2B, 0x81, 0x04, 0x00, 0x22} // MPI encoded oidCurveP384
		testFingerprint = make([]byte, 20)
	)

	if _, ok := keyValue.(*ecdh.PrivateKey); ok {
		priv := keyValue.(*ecdh.PrivateKey)
		vsG := ciphertext[:8]
		m := ciphertext[8:42]
		password, err := ecdh.Decrypt(priv, vsG, m, testCurveOID, testFingerprint)
		if err != nil {
			logger.Sugar.Errorf("error decrypting: %s", err)
		}
		ciper := ciphertext[60:]
		plaintext := std.DecryptSymmetrical(password, ciper, "GCM")

		return plaintext
	} else if _, ok := keyValue.([]byte); ok {
		password := keyValue.([]byte)
		plaintext := std.DecryptSymmetrical(password, ciphertext, "GCM")

		return plaintext
	}
	panic("")
}

/**
 * 对称加密
 *
 * @param key  加密的密钥
 * @param data 待加密的明文数据
 * @return 加密后的数据
 * @throws EncryptException
 */
func EncryptSymmetrical(key []byte, plaintext []byte) []byte {
	return Encrypt(key, plaintext)
}

/**
 * 非对称解密
 *
 * @param key 解密的密钥
 * @param raw 已经加密的数据
 * @return 解密后的明文
 * @throws EncryptException
 */
func DecryptSymmetrical(key []byte, ciphertext []byte) []byte {
	return Decrypt(key, nil, ciphertext)
}

func ValidateKey(keyPair *ecdh.PublicKey, password []byte) bool {

	return true
}

func Sign(privateKey *ed25519.PrivateKey, passphrase []byte, plaintext []byte) (ciphertext []byte) {
	sign := ed25519.Sign(*privateKey, plaintext)

	return sign
}

func Verify(publicKey *ed25519.PublicKey, plaintext []byte, ciphertext []byte) (success bool) {
	verify := ed25519.Verify(*publicKey, plaintext, ciphertext)

	return verify
}

func EncryptKey(key []byte, publicKey *ecdh.PublicKey) []byte {
	return Encrypt(publicKey, key)
}

func DecryptKey(keyValue []byte, privateKey *ecdh.PrivateKey) []byte {
	return Decrypt(privateKey, nil, keyValue)
}

func WritePublicKey(publicKey *ecdh.PublicKey) string {
	b := BytePublicKey(publicKey)

	return std.EncodeBase64(b)
}

func WritePrivateKey(privateKey *ecdh.PrivateKey, password []byte) string {
	b := BytePrivateKey(privateKey, password)

	return std.EncodeBase64(b)
}

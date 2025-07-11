package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
)

var encryptionKey = []byte("12345678901234567890123456789012")

// EncryptMessage encrypts a message using AES encryption
func EncryptMessage(message string) (string, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	ciphertext := make([]byte, aes.BlockSize+len(message))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], []byte(message))

	return hex.EncodeToString(ciphertext), nil
}

// DecryptMessage decrypts a message using AES decryption
func DecryptMessage(encrypted string) (string, error) {
	ciphertext, err := hex.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return string(ciphertext), nil
}

// ValidateKey validates the encryption key
func ValidateKey() error {
	if len(encryptionKey) != 32 {
		return errors.New("invalid key size: must be exactly 32 bytes")
	}
	return nil
}

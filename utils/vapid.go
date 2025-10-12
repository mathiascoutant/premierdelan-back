package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
)

// GenerateVAPIDKeys génère une paire de clés VAPID (publique et privée)
func GenerateVAPIDKeys() (publicKey, privateKey string, err error) {
	// Générer une paire de clés ECDSA
	curve := elliptic.P256()
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("erreur lors de la génération de la clé: %w", err)
	}

	// Encoder la clé publique (X et Y)
	pubBytes := elliptic.Marshal(curve, key.PublicKey.X, key.PublicKey.Y)
	publicKey = base64.RawURLEncoding.EncodeToString(pubBytes)

	// Encoder la clé privée
	privBytes := key.D.Bytes()
	// Pad à 32 bytes si nécessaire
	if len(privBytes) < 32 {
		padding := make([]byte, 32-len(privBytes))
		privBytes = append(padding, privBytes...)
	}
	privateKey = base64.RawURLEncoding.EncodeToString(privBytes)

	return publicKey, privateKey, nil
}

// DecodeVAPIDPrivateKey décode une clé privée VAPID depuis base64
func DecodeVAPIDPrivateKey(privateKey string) (*ecdsa.PrivateKey, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(privateKey)
	if err != nil {
		return nil, fmt.Errorf("erreur lors du décodage de la clé privée: %w", err)
	}

	curve := elliptic.P256()
	d := new(big.Int).SetBytes(decoded)
	
	priv := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
		},
		D: d,
	}
	
	priv.PublicKey.X, priv.PublicKey.Y = curve.ScalarBaseMult(decoded)
	
	return priv, nil
}


package hardware

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
)

// ErrKeyGenerationFailed is returned when key generation fails
var ErrKeyGenerationFailed = fmt.Errorf("failed to generate cryptographic key")

// ErrInvalidSignature is returned when signature verification fails
var ErrInvalidSignature = fmt.Errorf("invalid signature")

// EthereumSignature represents an Ethereum-compatible signature with recovery
type EthereumSignature struct {
	R         *big.Int
	S         *big.Int
	V         byte     // Recovery ID for Ethereum compatibility
	Signature []byte   // 65-byte signature (r + s + v)
}

// HardwareManager manages Ethereum-compatible cryptographic operations
type HardwareManager struct {
	mu           sync.RWMutex
	privateKey   *ecdsa.PrivateKey
	publicKey    *ecdsa.PublicKey
	address      []byte // Ethereum address derived from public key
}

// NewHardwareManager creates a new Ethereum-compatible hardware manager
func NewHardwareManager(ctx context.Context) (*HardwareManager, error) {

	// Validate context
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	hw := &HardwareManager{}

	// Generate initial key pair using secp256k1
	if err := hw.generateKeyPair(ctx); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFailed, err)
	}

	return hw, nil
}

// generateKeyPair generates a new secp256k1 key pair (Ethereum standard)
func (hm *HardwareManager) generateKeyPair(ctx context.Context) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Generate secp256k1 private key using Ethereum's crypto package
	privKey, err := crypto.GenerateKey()
	if err != nil {
		return fmt.Errorf("failed to generate secp256k1 key: %w", err)
	}

	// Derive Ethereum address from public key
	address := crypto.PubkeyToAddress(privKey.PublicKey)

	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.privateKey = privKey
	hm.publicKey = &privKey.PublicKey
	hm.address = address.Bytes()

	return nil
}

// GetPublicKey returns a copy of the current secp256k1 public key
func (hm *HardwareManager) GetPublicKey() *ecdsa.PublicKey {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	if hm.publicKey == nil {
		return nil
	}

	// Return a copy to prevent external modification
	return &ecdsa.PublicKey{
		Curve: hm.publicKey.Curve,
		X:     new(big.Int).Set(hm.publicKey.X),
		Y:     new(big.Int).Set(hm.publicKey.Y),
	}
}

// GetEthereumAddress returns the Ethereum address derived from the public key
func (hm *HardwareManager) GetEthereumAddress() []byte {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	if hm.address == nil {
		return nil
	}

	// Return a copy to prevent external modification
	addr := make([]byte, len(hm.address))
	copy(addr, hm.address)
	return addr
}

// GetEthereumAddressHex returns the Ethereum address as a hex string
func (hm *HardwareManager) GetEthereumAddressHex() string {
	addr := hm.GetEthereumAddress()
	if addr == nil {
		return ""
	}
	return fmt.Sprintf("0x%x", addr)
}

// Sign signs a hash using Ethereum-compatible signature format
func (hm *HardwareManager) Sign(ctx context.Context, hash []byte) (*EthereumSignature, error) {
	if len(hash) == 0 {
		return nil, fmt.Errorf("hash cannot be empty")
	}

	// Ethereum typically signs 32-byte hashes
	if len(hash) != 32 {
		return nil, fmt.Errorf("hash must be 32 bytes for Ethereum compatibility, got %d bytes", len(hash))
	}

	hm.mu.RLock()
	defer hm.mu.RUnlock()

	if hm.privateKey == nil {
		return nil, fmt.Errorf("private key not initialized")
	}

	// Sign using Ethereum's crypto package for full compatibility
	signature, err := crypto.Sign(hash, hm.privateKey)
	if err != nil {
		return nil, fmt.Errorf("signing failed: %w", err)
	}

	// Parse the signature components
	if len(signature) != 65 {
		return nil, fmt.Errorf("invalid signature length: expected 65, got %d", len(signature))
	}

	r := new(big.Int).SetBytes(signature[0:32])
	s := new(big.Int).SetBytes(signature[32:64])
	v := signature[64]

	ethSig := &EthereumSignature{
		R:         r,
		S:         s,
		V:         v,
		Signature: signature,
	}

	return ethSig, nil
}

// SignPersonalMessage signs a message using Ethereum's personal message format
func (hm *HardwareManager) SignPersonalMessage(ctx context.Context, message []byte) (*EthereumSignature, error) {
	if len(message) == 0 {
		return nil, fmt.Errorf("message cannot be empty")
	}

	// Create Ethereum personal message hash
	hash := crypto.Keccak256Hash(
		[]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(message))),
		message,
	)

	return hm.Sign(ctx, hash.Bytes())
}

// VerifySignature verifies an Ethereum signature against a hash
func (hm *HardwareManager) VerifySignature(hash []byte, sig *EthereumSignature) (bool, error) {
	if len(hash) != 32 {
		return false, fmt.Errorf("hash must be 32 bytes")
	}

	if sig == nil || len(sig.Signature) != 65 {
		return false, fmt.Errorf("invalid signature")
	}

	// Verify using Ethereum's crypto package
	pubKeyBytes, err := crypto.Ecrecover(hash, sig.Signature)
	if err != nil {
		return false, fmt.Errorf("signature recovery failed: %w", err)
	}

	hm.mu.RLock()
	expectedPubKeyBytes := crypto.FromECDSAPub(hm.publicKey)
	hm.mu.RUnlock()

	// Compare public keys
	if len(pubKeyBytes) != len(expectedPubKeyBytes) {
		return false, nil
	}

	for i := range pubKeyBytes {
		if pubKeyBytes[i] != expectedPubKeyBytes[i] {
			return false, nil
		}
	}

	return true, nil
}

// RecoverPublicKey recovers the public key from a signature and hash
func RecoverPublicKey(hash []byte, sig *EthereumSignature) (*ecdsa.PublicKey, error) {
	if len(hash) != 32 {
		return nil, fmt.Errorf("hash must be 32 bytes")
	}

	if sig == nil || len(sig.Signature) != 65 {
		return nil, fmt.Errorf("invalid signature")
	}

	// Recover public key using Ethereum's crypto package
	pubKeyBytes, err := crypto.Ecrecover(hash, sig.Signature)
	if err != nil {
		return nil, fmt.Errorf("public key recovery failed: %w", err)
	}

	pubKey, err := crypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal public key: %w", err)
	}

	return pubKey, nil
}

// GetKeyInfo returns information about the current key
func (hm *HardwareManager) GetKeyInfo() map[string]interface{} {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	return map[string]interface{}{
		"curve":           "secp256k1",
		"ethereum_address": hm.GetEthereumAddressHex(),
		"has_private_key": hm.privateKey != nil,
		"has_public_key":  hm.publicKey != nil,
	}
}

// Close performs cleanup operations
func (hm *HardwareManager) Close() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Zero out the private key for security
	if hm.privateKey != nil {
		// Clear the private key bytes
		if hm.privateKey.D != nil {
			hm.privateKey.D.SetInt64(0)
		}
		hm.privateKey = nil
	}

	hm.publicKey = nil
	hm.address = nil

	return nil
}

// IsValidEthereumSignature validates an Ethereum signature format
func IsValidEthereumSignature(sig *EthereumSignature) bool {
	if sig == nil {
		return false
	}

	// Check signature length
	if len(sig.Signature) != 65 {
		return false
	}

	// Check recovery ID (v) is valid (0 or 1, or 27/28 for legacy)
	if sig.V != 0 && sig.V != 1 && sig.V != 27 && sig.V != 28 {
		return false
	}

	// Check r and s are not zero
	if sig.R == nil || sig.S == nil {
		return false
	}

	if sig.R.Sign() == 0 || sig.S.Sign() == 0 {
		return false
	}

	// Check r and s are less than secp256k1 curve order
	secp256k1N := crypto.S256().Params().N
	if sig.R.Cmp(secp256k1N) >= 0 || sig.S.Cmp(secp256k1N) >= 0 {
		return false
	}

	return true
}
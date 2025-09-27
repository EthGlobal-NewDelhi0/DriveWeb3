package commands

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"peerdrive/app/pkg/hardware"
	"peerdrive/app/pkg/network"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	ethclient "github.com/ethereum/go-ethereum/ethclient"
)

// CommandResult holds the result of a command execution
type CommandResult struct {
	Success bool
	Message string
	Error   error
}

// CommandHandlers contains all the command handling logic
type CommandHandlers struct {
	netManager    *network.MeshNetworkManager
	cryptoManager *hardware.HardwareManager
	rpcURL        string
	privateKey    string
	tokenAddress  string
}

// NewCommandHandlers creates a new command handlers instance
func NewCommandHandlers(netManager *network.MeshNetworkManager, cryptoManager *hardware.HardwareManager, rpcURL, privateKey, tokenAddress string) *CommandHandlers {
	return &CommandHandlers{
		netManager:    netManager,
		cryptoManager: cryptoManager,
		rpcURL:        rpcURL,
		privateKey:    privateKey,
		tokenAddress:  tokenAddress,
	}
}

// AddPeer connects to a new peer
func (h *CommandHandlers) AddPeer(peerAddress string) CommandResult {
	if peerAddress == "" {
		return CommandResult{Success: false, Message: "No address provided"}
	}
	h.netManager.ConnectToPeer(peerAddress)
	return CommandResult{Success: true, Message: fmt.Sprintf("Attempting to connect to peer: %s", peerAddress)}
}

// RemovePeer disconnects from a peer
func (h *CommandHandlers) RemovePeer(peerID string) CommandResult {
	if peerID == "" {
		return CommandResult{Success: false, Message: "No peer ID provided"}
	}
	h.netManager.DisconnectFromPeer(peerID)
	return CommandResult{Success: true, Message: fmt.Sprintf("Attempting to disconnect from peer: %s", peerID)}
}

// ShowPeers lists all connected peers in a formatted way
func (h *CommandHandlers) ShowPeers() CommandResult {
	peers := h.netManager.Host.Network().Peers()
	if len(peers) == 0 {
		return CommandResult{Success: true, Message: "👥 No connected peers"}
	}

	var message strings.Builder
	message.WriteString(fmt.Sprintf("👥 CONNECTED PEERS (%d)\n", len(peers)))
	message.WriteString(strings.Repeat("=", 40) + "\n\n")

	for i, peer := range peers {
		message.WriteString(fmt.Sprintf("🔗 Peer %d: %s\n", i+1, peer.String()))

		// Get connection info if available
		conns := h.netManager.Host.Network().ConnsToPeer(peer)
		if len(conns) > 0 {
			message.WriteString(fmt.Sprintf("   📡 Connections: %d\n", len(conns)))
			if len(conns[0].RemoteMultiaddr().String()) > 0 {
				message.WriteString(fmt.Sprintf("   🌐 Address: %s\n", conns[0].RemoteMultiaddr().String()))
			}
		}
		message.WriteString("\n")
	}
	return CommandResult{Success: true, Message: message.String()}
}

// ShowAccruedRewards shows the current accrued rewards
func (h *CommandHandlers) ShowAccruedRewards() CommandResult {
	return CommandResult{
		Success: true,
		Message: fmt.Sprintf("Accrued Rewards: %d", h.netManager.AccruedRewards),
	}
}

// ShowNonce shows the current nonce
func (h *CommandHandlers) ShowNonce() CommandResult {
	return CommandResult{
		Success: true,
		Message: fmt.Sprintf("Nonce: %d", h.netManager.Nonce),
	}
}

// ShowLedger displays the current ledger state in a human-readable format
func (h *CommandHandlers) ShowLedger() CommandResult {
	ledger := h.netManager.GetLedger()

	if len(ledger.Ledger) == 0 {
		return CommandResult{Success: true, Message: "📚 Ledger is empty - no vehicles detected"}
	}

	var message strings.Builder
	message.WriteString(fmt.Sprintf("📚 VEHICLE LEDGER (%d vehicles)\n", len(ledger.Ledger)))
	message.WriteString(strings.Repeat("=", 50) + "\n\n")

	for peerID, entry := range ledger.Ledger {
		vehicle := entry.VehicleState

		// Determine vehicle type
		vehicleType := "🚗 Vehicle"
		if vehicle.Priority >= 10 {
			vehicleType = "🚑 Emergency Vehicle"
		}

		// Format position
		message.WriteString(fmt.Sprintf("%s\n", vehicleType))
		message.WriteString(fmt.Sprintf("  ID: %s\n", peerID))
		message.WriteString(fmt.Sprintf("  📍 Position: (%.3f, %.3f)\n", vehicle.Position.Lat, vehicle.Position.Lng))
		message.WriteString(fmt.Sprintf("  🏃 Velocity: (%.3f, %.3f) m/s\n", vehicle.Velocity.Lat, vehicle.Velocity.Lng))
		message.WriteString(fmt.Sprintf("  ⚡ Acceleration: (%.3f, %.3f) m/s²\n", vehicle.Acceleration.Lat, vehicle.Acceleration.Lng))
		message.WriteString(fmt.Sprintf("  ⭐ Priority: %d\n", vehicle.Priority))
		message.WriteString(fmt.Sprintf("  🕒 Last Update: %d ms ago\n", vehicle.Timestamp))
		message.WriteString(fmt.Sprintf("  👥 Neighbor Count: %d\n", entry.Neighbour))
		message.WriteString("\n")
	}

	return CommandResult{Success: true, Message: message.String()}
}

// ShowEthAddress displays the Ethereum address
func (h *CommandHandlers) ShowEthAddress() CommandResult {
	addr := h.cryptoManager.GetEthereumAddressHex()
	return CommandResult{
		Success: true,
		Message: fmt.Sprintf("Ethereum address: %s", strings.ToLower(addr)),
	}
}

// ApproveToken approves token for a spender
func (h *CommandHandlers) ApproveToken(ctx context.Context, spenderAddress string) CommandResult {
	if spenderAddress == "" {
		return CommandResult{Success: false, Message: "No spender address provided"}
	}

	spenderLower := strings.ToLower(spenderAddress)
	owner := strings.ToLower(h.cryptoManager.GetEthereumAddressHex())

	// Sign the message (spenderLower)
	sig, err := h.cryptoManager.SignPersonalMessage(ctx, []byte(spenderLower))
	if err != nil {
		return CommandResult{Success: false, Message: "Failed to sign message", Error: err}
	}

	// Check required environment variables
	if h.rpcURL == "" || h.privateKey == "" || h.tokenAddress == "" {
		return CommandResult{Success: false, Message: "Missing RPC_URL, PRIVATE_KEY, or TOKEN_ADDRESS in environment variables"}
	}

	// Execute transaction in background
	go func() {
		result := h.executeApproveTransaction(ctx, owner, spenderLower, sig.Signature)
		fmt.Printf("Approve transaction result: %s\n", result.Message)
	}()

	return CommandResult{Success: true, Message: "Approve transaction initiated..."}
}

// RedeemRewards redeems accrued rewards
func (h *CommandHandlers) RedeemRewards(ctx context.Context) CommandResult {
	if h.rpcURL == "" || h.privateKey == "" || h.tokenAddress == "" {
		return CommandResult{Success: false, Message: "Missing RPC_URL, PRIVATE_KEY, or TOKEN_ADDRESS in environment variables"}
	}

	accruedRewards := h.netManager.AccruedRewards
	nonceValue := h.netManager.Nonce

	// Prepare the message to sign: "(amount,nonce)"
	message := fmt.Sprintf("(%d,%d)", accruedRewards, nonceValue)

	// Sign the message
	sig, err := h.cryptoManager.SignPersonalMessage(ctx, []byte(message))
	if err != nil {
		return CommandResult{Success: false, Message: "Failed to sign message", Error: err}
	}

	to := strings.ToLower(h.cryptoManager.GetEthereumAddressHex())

	// Execute transaction in background
	go func() {
		result := h.executeRedeemTransaction(ctx, to, accruedRewards, nonceValue, sig.Signature)
		fmt.Printf("Redeem transaction result: %s\n", result.Message)
	}()

	return CommandResult{Success: true, Message: "Redeem transaction initiated..."}
}

// executeApproveTransaction handles the approve transaction
func (h *CommandHandlers) executeApproveTransaction(ctx context.Context, owner, spender string, signature []byte) CommandResult {
	client, err := ethclient.Dial(h.rpcURL)
	if err != nil {
		return CommandResult{Success: false, Message: "Failed to connect to RPC", Error: err}
	}
	defer client.Close()

	pk, err := crypto.HexToECDSA(strings.TrimPrefix(h.privateKey, "0x"))
	if err != nil {
		return CommandResult{Success: false, Message: "Invalid private key", Error: err}
	}

	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return CommandResult{Success: false, Message: "Failed to get chain ID", Error: err}
	}

	auth, err := bind.NewKeyedTransactorWithChainID(pk, chainID)
	if err != nil {
		return CommandResult{Success: false, Message: "Failed to create transactor", Error: err}
	}

	abiJSON := `[{"inputs":[{"internalType":"address","name":"owner","type":"address"},{"internalType":"address","name":"spender","type":"address"},{"internalType":"bytes","name":"signature","type":"bytes"}],"name":"approve","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"}]`
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return CommandResult{Success: false, Message: "Failed to parse ABI", Error: err}
	}

	ownerAddr := common.HexToAddress(owner)
	spenderAddr := common.HexToAddress(spender)

	data, err := parsedABI.Pack("approve", ownerAddr, spenderAddr, signature)
	if err != nil {
		return CommandResult{Success: false, Message: "Failed to pack ABI data", Error: err}
	}

	tokenAddr := common.HexToAddress(h.tokenAddress)
	nonce, err := client.PendingNonceAt(ctx, auth.From)
	if err != nil {
		return CommandResult{Success: false, Message: "Failed to get nonce", Error: err}
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return CommandResult{Success: false, Message: "Failed to get gas price", Error: err}
	}

	msg := ethereum.CallMsg{
		From: auth.From,
		To:   &tokenAddr,
		Data: data,
	}
	gasLimit, err := client.EstimateGas(ctx, msg)
	if err != nil {
		gasLimit = uint64(200000) // fallback
	}

	tx := types.NewTransaction(nonce, tokenAddr, big.NewInt(0), gasLimit, gasPrice, data)
	signedTx, err := auth.Signer(auth.From, tx)
	if err != nil {
		return CommandResult{Success: false, Message: "Failed to sign transaction", Error: err}
	}

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return CommandResult{Success: false, Message: "Failed to send transaction", Error: err}
	}

	return CommandResult{Success: true, Message: fmt.Sprintf("Approve transaction sent: %s", signedTx.Hash().Hex())}
}

// executeRedeemTransaction handles the redeem transaction
func (h *CommandHandlers) executeRedeemTransaction(ctx context.Context, to string, amount, nonce int, signature []byte) CommandResult {
	client, err := ethclient.Dial(h.rpcURL)
	if err != nil {
		return CommandResult{Success: false, Message: "Failed to connect to RPC", Error: err}
	}
	defer client.Close()

	pk, err := crypto.HexToECDSA(strings.TrimPrefix(h.privateKey, "0x"))
	if err != nil {
		return CommandResult{Success: false, Message: "Invalid private key", Error: err}
	}

	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return CommandResult{Success: false, Message: "Failed to get chain ID", Error: err}
	}

	auth, err := bind.NewKeyedTransactorWithChainID(pk, chainID)
	if err != nil {
		return CommandResult{Success: false, Message: "Failed to create transactor", Error: err}
	}

	abiJSON := `[{"inputs":[{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"uint256","name":"nonce","type":"uint256"},{"internalType":"bytes","name":"signature","type":"bytes"}],"name":"mint","outputs":[],"stateMutability":"nonpayable","type":"function"}]`
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return CommandResult{Success: false, Message: "Failed to parse ABI", Error: err}
	}

	toAddr := common.HexToAddress(to)
	amountBig := big.NewInt(int64(amount))
	nonceBig := big.NewInt(int64(nonce))

	data, err := parsedABI.Pack("mint", toAddr, amountBig, nonceBig, signature)
	if err != nil {
		return CommandResult{Success: false, Message: "Failed to pack ABI data", Error: err}
	}

	tokenAddr := common.HexToAddress(h.tokenAddress)
	txNonce, err := client.PendingNonceAt(ctx, auth.From)
	if err != nil {
		return CommandResult{Success: false, Message: "Failed to get nonce", Error: err}
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return CommandResult{Success: false, Message: "Failed to get gas price", Error: err}
	}

	msg := ethereum.CallMsg{
		From: auth.From,
		To:   &tokenAddr,
		Data: data,
	}
	gasLimit, err := client.EstimateGas(ctx, msg)
	if err != nil {
		gasLimit = uint64(200000) // fallback
	}

	tx := types.NewTransaction(txNonce, tokenAddr, big.NewInt(0), gasLimit, gasPrice, data)
	signedTx, err := auth.Signer(auth.From, tx)
	if err != nil {
		return CommandResult{Success: false, Message: "Failed to sign transaction", Error: err}
	}

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return CommandResult{Success: false, Message: "Failed to send transaction", Error: err}
	}

	// Wait for transaction confirmation
	receipt, err := bind.WaitMined(ctx, client, signedTx)
	if err != nil {
		return CommandResult{Success: false, Message: "Error waiting for transaction confirmation", Error: err}
	}

	if receipt.Status != 1 {
		return CommandResult{Success: false, Message: fmt.Sprintf("Transaction failed with status %v", receipt.Status)}
	}

	// Update accrued rewards and nonce after confirmation
	h.netManager.AccruedRewards -= amount
	h.netManager.Nonce += 1

	return CommandResult{Success: true, Message: fmt.Sprintf("Redeem transaction confirmed: %s", signedTx.Hash().Hex())}
}

// GetPeerID returns the current peer ID
func (h *CommandHandlers) GetPeerID() string {
	return h.netManager.GetPeerId()
}

// GetPeerAddress returns the multiaddress for this peer
func (h *CommandHandlers) GetPeerAddress() string {
	if len(h.netManager.Host.Addrs()) > 0 {
		return fmt.Sprintf("%s/p2p/%s", h.netManager.Host.Addrs()[0].String(), h.netManager.Host.ID())
	}
	return h.netManager.Host.ID().String()
}

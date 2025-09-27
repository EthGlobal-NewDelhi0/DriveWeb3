package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"peerdrive/app/pkg/hardware"
	"peerdrive/app/pkg/network"
	"peerdrive/app/pkg/sensor"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	ethclient "github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

func main() {


	// Load .env file before reading environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found or failed to load, proceeding with existing environment variables")
	}
	rpcURL := os.Getenv("RPC_URL")
	privateKey := os.Getenv("PRIVATE_KEY")
	tokenAddress := os.Getenv("TOKEN_ADDRESS")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize crypto hardware manager
	cryptoManager, err := hardware.NewHardwareManager(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize crypto manager: %v", err)
	}

	// 1. Initialize Network Manager
	netManager, err := network.NewMeshNetworkManager(ctx)
	if err != nil {
		log.Fatalf("Failed to create network manager: %v", err)
	}
	netManager.StartMessageHandler(ctx)

	// 2. Initialize Mock Sensor
	sensorManager := sensor.NewMockManager(netManager.GetPeerId())
	vehicleStateChan, err := sensorManager.StartTracking()
	if err != nil {
		log.Fatalf("Failed to start sensor manager: %v", err)
	}

	// 3. Goroutine for user input to connect to peers
	go handleUserInput(ctx, netManager, cryptoManager, rpcURL, privateKey, tokenAddress)

	// 4. Ticker for periodic gossiping and printing
	gossipTicker := time.NewTicker(time.Second/4)
	defer gossipTicker.Stop()

	log.Println("🚀 Application started. Waiting for sensor data...")

	// 5. Main application loop
	for {
		select {
		case stateUpdate := <-vehicleStateChan:
			// A new state was received from our mock sensor
			netManager.UpdateLocalVehicleState(stateUpdate)

		case <-gossipTicker.C:
			netManager.Gossip()
			netManager.StoreLedger()

		case <-ctx.Done():
			log.Println("Shutting down.")
			return
		}
	}
}

func handleUserInput(ctx context.Context, netManager *network.MeshNetworkManager, cryptoManager *hardware.HardwareManager, rpcURL string, privateKey string, tokenAddress string) {

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		switch input {

		case "/addpeer":
			fmt.Print("Enter peer multiaddress: ")
			addr, _ := reader.ReadString('\n')
			addr = strings.TrimSpace(addr)
			if addr != "" {
				netManager.ConnectToPeer(addr)
			} else {
				fmt.Println("No address entered.")
			}

		case "/removepeer":
			fmt.Print("Enter peer ID to disconnect: ")
			peerID, _ := reader.ReadString('\n')
			peerID = strings.TrimSpace(peerID)
			if peerID != "" {
				netManager.DisconnectFromPeer(peerID)
			} else {
				fmt.Println("No peer ID entered.")
			}

		case "/showpeers":
			fmt.Println("Connected peers:")
			fmt.Println(netManager.Host.Network().Peers())

		case "/accruedRewards":
			fmt.Printf("Accrued Rewards: %d\n", netManager.AccruedRewards)

		case "/nonce":
			fmt.Printf("Nonce: %d\n", netManager.Nonce)

		case "/showledger":
			ledger := netManager.GetLedger()
			prettyLedger, _ := json.MarshalIndent(ledger, "", "  ")
			fmt.Println(string(prettyLedger))

		case "/ethAddress":
			addr := cryptoManager.GetEthereumAddressHex()
			fmt.Printf("Ethereum address: %s\n", strings.ToLower(addr))

		case "/approveToken":
			// Prompt for spender address
			fmt.Print("Enter spender address: ")
			spender, _ := reader.ReadString('\n')
			spender = strings.TrimSpace(spender)
			if spender == "" {
				fmt.Println("No spender address entered.")
				break
			}
			spenderLower := strings.ToLower(spender)

			// Get owner address from cryptoManager
			owner := cryptoManager.GetEthereumAddressHex()
			owner = strings.ToLower(owner)

			// Sign the message (spenderLower)
			sig, err := cryptoManager.SignPersonalMessage(ctx, []byte(spenderLower))
			if err != nil {
				fmt.Printf("Failed to sign message: %v\n", err)
				break
			}

			// Prepare signature bytes for contract call
			signatureBytes := sig.Signature

			// Call approve on the token contract
			if rpcURL == "" || privateKey == "" || tokenAddress == "" {
				fmt.Println("Missing RPC_URL, PRIVATE_KEY, or TOKEN_ADDRESS in environment variables.")
				break
			}

			// Use go-ethereum for contract call
			go func() {
				// Import dependencies here to avoid import errors in the rest of the file
				// (in real code, move these to the top)
				// "github.com/ethereum/go-ethereum/accounts/abi"
				// "github.com/ethereum/go-ethereum/accounts/abi/bind"
				// "github.com/ethereum/go-ethereum/common"
				// "github.com/ethereum/go-ethereum/crypto"
				// "github.com/ethereum/go-ethereum/ethclient"
				// "math/big"

				// Connect to RPC
				client, err := ethclient.Dial(rpcURL)
				if err != nil {
					fmt.Printf("Failed to connect to RPC: %v\n", err)
					return
				}
				defer client.Close()

				// Parse private key
				pk, err := crypto.HexToECDSA(strings.TrimPrefix(privateKey, "0x"))
				if err != nil {
					fmt.Printf("Invalid private key: %v\n", err)
					return
				}

				// Get chainID
				chainID, err := client.NetworkID(ctx)
				if err != nil {
					fmt.Printf("Failed to get chain ID: %v\n", err)
					return
				}

				auth, err := bind.NewKeyedTransactorWithChainID(pk, chainID)
				if err != nil {
					fmt.Printf("Failed to create transactor: %v\n", err)
					return
				}

				// Prepare contract ABI
				abiJSON := `[{"inputs":[{"internalType":"address","name":"owner","type":"address"},{"internalType":"address","name":"spender","type":"address"},{"internalType":"bytes","name":"signature","type":"bytes"}],"name":"approve","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"}]`
				parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
				if err != nil {
					fmt.Printf("Failed to parse ABI: %v\n", err)
					return
				}

				// Prepare input arguments
				ownerAddr := common.HexToAddress(owner)
				spenderAddr := common.HexToAddress(spenderLower)

				// Pack the data for the approve function
				data, err := parsedABI.Pack("approve", ownerAddr, spenderAddr, signatureBytes)
				if err != nil {
					fmt.Printf("Failed to pack ABI data: %v\n", err)
					return
				}

				// Prepare transaction
				tokenAddr := common.HexToAddress(tokenAddress)
				nonce, err := client.PendingNonceAt(ctx, auth.From)
				if err != nil {
					fmt.Printf("Failed to get nonce: %v\n", err)
					return
				}
				gasPrice, err := client.SuggestGasPrice(ctx)
				if err != nil {
					fmt.Printf("Failed to get gas price: %v\n", err)
					return
				}

				// Estimate gas
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

				// Sign and send
				signedTx, err := auth.Signer(auth.From, tx)
				if err != nil {
					fmt.Printf("Failed to sign transaction: %v\n", err)
					return
				}
				err = client.SendTransaction(ctx, signedTx)
				if err != nil {
					fmt.Printf("Failed to send transaction: %v\n", err)
					return
				}
				fmt.Printf("Approve transaction sent: %s\n", signedTx.Hash().Hex())
			}()
		case "/redeem":
			// Redeem rewards by calling mint on the token contract
			if rpcURL == "" || privateKey == "" || tokenAddress == "" {
				fmt.Println("Missing RPC_URL, PRIVATE_KEY, or TOKEN_ADDRESS in environment variables.")
				break
			}

			// Get accrued rewards and nonce from the network manager
			accruedRewards := netManager.AccruedRewards
			nonceValue := netManager.Nonce

			// Prepare the message to sign: "(amount,nonce)"
			message := fmt.Sprintf("(%d,%d)", accruedRewards, nonceValue)

			// Sign the message using the crypto manager
			sig, err := cryptoManager.SignPersonalMessage(ctx, []byte(message))
			if err != nil {
				fmt.Printf("Failed to sign message: %v\n", err)
				break
			}
			signatureBytes := sig.Signature

			// Get recipient address (to)
			to := cryptoManager.GetEthereumAddressHex()
			to = strings.ToLower(to)

			go func() {
				// Import dependencies here to avoid import errors in the rest of the file
				// (in real code, move these to the top)
				// "github.com/ethereum/go-ethereum/accounts/abi"
				// "github.com/ethereum/go-ethereum/accounts/abi/bind"
				// "github.com/ethereum/go-ethereum/common"
				// "github.com/ethereum/go-ethereum/crypto"
				// "github.com/ethereum/go-ethereum/ethclient"
				// "math/big"

				client, err := ethclient.Dial(rpcURL)
				if err != nil {
					fmt.Printf("Failed to connect to RPC: %v\n", err)
					return
				}
				defer client.Close()

				pk, err := crypto.HexToECDSA(strings.TrimPrefix(privateKey, "0x"))
				if err != nil {
					fmt.Printf("Invalid private key: %v\n", err)
					return
				}

				chainID, err := client.NetworkID(ctx)
				if err != nil {
					fmt.Printf("Failed to get chain ID: %v\n", err)
					return
				}

				auth, err := bind.NewKeyedTransactorWithChainID(pk, chainID)
				if err != nil {
					fmt.Printf("Failed to create transactor: %v\n", err)
					return
				}

				// Prepare contract ABI for mint
				abiJSON := `[{"inputs":[{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"uint256","name":"nonce","type":"uint256"},{"internalType":"bytes","name":"signature","type":"bytes"}],"name":"mint","outputs":[],"stateMutability":"nonpayable","type":"function"}]`
				parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
				if err != nil {
					fmt.Printf("Failed to parse ABI: %v\n", err)
					return
				}

				toAddr := common.HexToAddress(to)
				amount := big.NewInt(int64(accruedRewards))
				nonceBig := big.NewInt(int64(nonceValue))

				data, err := parsedABI.Pack("mint", toAddr, amount, nonceBig, signatureBytes)
				if err != nil {
					fmt.Printf("Failed to pack ABI data: %v\n", err)
					return
				}

				tokenAddr := common.HexToAddress(tokenAddress)
				txNonce, err := client.PendingNonceAt(ctx, auth.From)
				if err != nil {
					fmt.Printf("Failed to get nonce: %v\n", err)
					return
				}
				gasPrice, err := client.SuggestGasPrice(ctx)
				if err != nil {
					fmt.Printf("Failed to get gas price: %v\n", err)
					return
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
					fmt.Printf("Failed to sign transaction: %v\n", err)
					return
				}
				err = client.SendTransaction(ctx, signedTx)
				if err != nil {
					fmt.Printf("Failed to send transaction: %v\n", err)
					return
				}
				fmt.Printf("Redeem (mint) transaction sent: %s\n", signedTx.Hash().Hex())

				// Wait for transaction confirmation before updating accruedRewards and nonce
				receipt, err := bind.WaitMined(ctx, client, signedTx)
				if err != nil {
					fmt.Printf("Error waiting for transaction confirmation: %v\n", err)
					return
				}
				if receipt.Status != 1 {
					fmt.Printf("Transaction failed with status %v\n", receipt.Status)
					return
				}
				// Update accruedRewards and nonce after confirmation
				netManager.AccruedRewards -= int(amount.Int64())
				netManager.Nonce += 1
			}()
		case "/clear":
			// Clear the terminal screen (works on most Unix terminals)
			fmt.Print("\033[H\033[2J")
		default:
			fmt.Println("Unknown command.")
			fmt.Println("Available commands:")
			fmt.Println("  /addpeer        - Add a peer by multiaddress")
			fmt.Println("  /removepeer     - Remove a peer by peer ID")
			fmt.Println("  /showpeers      - List connected peers")
			fmt.Println("  /showledger     - Show the current ledger")
			fmt.Println("  /ethAddress     - Show your Ethereum address")
			fmt.Println("  /approveToken   - Approve token for a spender")
			fmt.Println("  /redeem         - Redeem accrued rewards")
			fmt.Println("  /accruedRewards - Show accrued rewards")
			fmt.Println("  /nonce          - Show current nonce")
			fmt.Println("  /clear          - Clears terminal")
		}
	}
}
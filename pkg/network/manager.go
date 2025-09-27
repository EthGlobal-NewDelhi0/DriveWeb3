package network

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"peerdrive/app/pkg/models"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	host "github.com/libp2p/go-libp2p/core/host"
	peer "github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

type MeshNetworkManager struct {
	Host           host.Host
	ps             *pubsub.PubSub
	topic          *pubsub.Topic
	sub            *pubsub.Subscription
	ledger         models.Ledger
	AccruedRewards int
	Nonce          int
}

func NewMeshNetworkManager(ctx context.Context) (*MeshNetworkManager, error) {
	// Create a new libp2p Host
	h, err := libp2p.New()
	if err != nil {
		return nil, err
	}

	for _, a := range h.Addrs() {
		fmt.Printf("📡 %s/p2p/%s\n", a.String(), h.ID())
	}

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		log.Fatal(err)
	}
	topic, err := ps.Join("peerdrive")
	if err != nil {
		log.Fatal(err)
	}
	sub, err := topic.Subscribe()
	if err != nil {
		log.Fatal(err)
	}

	m := &MeshNetworkManager{
		Host:           h,
		ps:             ps,
		topic:          topic,
		sub:            sub,
		ledger:         models.Ledger{Ledger: make(map[string]models.NeighbourEntry)},
		AccruedRewards: 0,
		Nonce:          1,
	}

	return m, nil
}

// StartMessageHandler launches a goroutine to read messages from the pubsub topic.
func (m *MeshNetworkManager) StartMessageHandler(ctx context.Context) {
	go func() {
		for {
			msg, err := m.sub.Next(ctx)
			if err != nil {
				log.Println("Subscription closed:", err)
				return
			}
			// Don't process messages we sent ourselves.
			if msg.ReceivedFrom == m.Host.ID() {
				continue
			}
			var incomingLedger models.Ledger
			if err := json.Unmarshal(msg.Data, &incomingLedger); err != nil {
				log.Printf("Error unmarshalling ledger: %v", err)
				continue
			}
			m.handleGossip(incomingLedger)
		}
	}()
}

// UpdateLocalVehicleState updates the state of this specific vehicle in the ledger.
func (m *MeshNetworkManager) UpdateLocalVehicleState(state models.VehicleState) {
	m.ledger.Ledger[m.GetPeerId()] = models.NeighbourEntry{
		Neighbour:    0,
		VehicleState: state,
	}
}

// GetLedger returns a copy of the current ledger for inspection.
func (m *MeshNetworkManager) GetLedger() models.Ledger {
	return m.ledger
}

// updateLedger only updates existing first neighbours (Neighbour == 1) in the ledger, does not add new ones.
func (m *MeshNetworkManager) updateLedger(peerId string, entry models.NeighbourEntry) {
	// Only update if the peer is already in the ledger and is a first neighbour
	existing, exists := m.ledger.Ledger[peerId]
	if exists && existing.Neighbour == 1 && entry.Neighbour == 1 {
		if entry.VehicleState.Timestamp > existing.VehicleState.Timestamp {
			m.ledger.Ledger[peerId] = entry
		}
	}
}

func (m *MeshNetworkManager) handleGossip(ledger models.Ledger) {
	// Only update existing first neighbours from the incoming ledger
	for peerId, incomingEntry := range ledger.Ledger {
		// Ignore self from the incoming ledger
		if peerId == m.GetPeerId() {
			continue
		}

		if localEntry, exists := m.ledger.Ledger[peerId]; exists && localEntry.Neighbour == 1 {
			// Only update if this peer is already in our ledger as a first neighbour
			entryForLocalLedger := models.NeighbourEntry{
				Neighbour:    1,
				VehicleState: incomingEntry.VehicleState,
			}
			m.updateLedger(peerId, entryForLocalLedger)
		} else if !exists && incomingEntry.Neighbour == 0 {
			// If peer not present in the ledger but incomingEntry.Neighbour == 0, add to ledger and call updateLedger
			m.ledger.Ledger[peerId] = models.NeighbourEntry{
				Neighbour:    1,
				VehicleState: incomingEntry.VehicleState,
			}
			m.updateLedger(peerId, models.NeighbourEntry{
				Neighbour:    1,
				VehicleState: incomingEntry.VehicleState,
			})
		}
	}
}

func (m *MeshNetworkManager) Gossip() {
	// gossip the ledger
	ledgerBytes, err := json.Marshal(m.ledger)
	if err != nil {
		// handle error, maybe log it
		fmt.Println("Error marshalling ledger:", err)
		return
	}
	m.topic.Publish(context.Background(), ledgerBytes)
	m.updateRewards(len(m.Host.Network().Peers()) + 1)
}

// connect to a new peer
func (m *MeshNetworkManager) ConnectToPeer(peerID string) {
	// Assume peerID is a string representation of a peer's multiaddress
	ctx := context.Background()
	peerAddr, err := ma.NewMultiaddr(peerID)
	if err != nil {
		fmt.Println("❌ invalid peer multiaddress:", err)
		return
	}
	pinfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
	if err != nil {
		fmt.Println("❌ failed to parse peer address info:", err)
		return
	}

	peerKey := pinfo.ID
	// Check if already connected or attempted
	for _, pid := range m.Host.Network().Peers() {
		if pid == peerKey {
			fmt.Println("🔄 already connected to peer:", peerKey)
			return
		}
	}

	if err := m.Host.Connect(ctx, *pinfo); err != nil {
		fmt.Println("❌ failed to connect to peer:", err)
	} else {
		fmt.Println("✅ connected to peer:", pinfo.ID)
		// Add the peer to the ledger with current timestamp and neighbour as 1
		now := time.Now().Unix()
		m.ledger.Ledger[peerKey.String()] = models.NeighbourEntry{
			Neighbour: 1,
			VehicleState: models.VehicleState{
				Timestamp: now,
			},
		}
	}
}

// disconnect from a peer
func (m *MeshNetworkManager) DisconnectFromPeer(peerID string) {
	// Check if already disconnected or never connected
	isConnected := false
	for _, pid := range m.Host.Network().Peers() {
		if pid.String() == peerID {
			isConnected = true
			break
		}
	}
	if !isConnected {
		fmt.Println("🔄 already disconnected from peer:", peerID)
		return
	}

	pid, err := peer.Decode(peerID) // convert string → peer.ID
	if err != nil {
		fmt.Println("❌ invalid peer ID:", err)
		return
	}

	if err := m.Host.Network().ClosePeer(pid); err != nil {
		fmt.Println("❌ failed to disconnect from peer:", err)
	} else {
		fmt.Println("✅ disconnected from peer:", peerID)
		// Remove the peer from the ledger
		delete(m.ledger.Ledger, peerID)
	}
}

func (m *MeshNetworkManager) StoreLedger() {
	data, err := json.MarshalIndent(m.ledger, "", "  ")
	if err != nil {
		log.Printf("Warning: Failed to marshal ledger: %v", err)
		return
	}

	filename := m.GetPeerId() + ".txt"
	if err := os.WriteFile(filename, data, 0644); err != nil {
		log.Printf("Warning: Failed to write ledger file %s: %v", filename, err)
	}
}

func (m *MeshNetworkManager) GetPeerId() string {
	return m.Host.ID().String()
}

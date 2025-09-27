package network

// This function manages the total rewards for the peer
func (m *MeshNetworkManager) updateRewards(newRewards int) {
	m.AccruedRewards += newRewards
}
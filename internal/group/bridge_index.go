package group

import (
	"crypto/sha256"
	"encoding/hex"
)

type contractLookupIndex struct {
	byUID  map[string]string
	byRef  map[string]string
	byFile map[string][]string
}

func contractNodeID(repoPath string, contractID string, role string, filePath string) string {
	sum := sha256.Sum256([]byte(repoPath + "\x00" + contractID + "\x00" + role + "\x00" + filePath))
	return hex.EncodeToString(sum[:])
}

func createContractLookupIndex() contractLookupIndex {
	return contractLookupIndex{
		byUID:  map[string]string{},
		byRef:  map[string]string{},
		byFile: map[string][]string{},
	}
}

func indexContract(index contractLookupIndex, contract StoredContract, nodeID string) {
	if contract.SymbolUID != "" {
		index.byUID[contractUIDKey(contract.Repo, contract.Role, contract.SymbolUID)] = nodeID
	}
	index.byRef[contractRefKey(contract.Repo, contract.Role, contract.SymbolRef.FilePath, contract.SymbolRef.Name)] = nodeID
	index.byFile[contractFileKey(contract.Repo, contract.Role, contract.SymbolRef.FilePath)] = append(index.byFile[contractFileKey(contract.Repo, contract.Role, contract.SymbolRef.FilePath)], nodeID)
}

func findContractNode(index contractLookupIndex, repoPath string, role string, symbolUID string, filePath string, symbolName string) string {
	if symbolUID != "" {
		if hit := index.byUID[contractUIDKey(repoPath, role, symbolUID)]; hit != "" {
			return hit
		}
	}
	if hit := index.byRef[contractRefKey(repoPath, role, filePath, symbolName)]; hit != "" {
		return hit
	}
	candidates := index.byFile[contractFileKey(repoPath, role, filePath)]
	if len(candidates) == 1 {
		return candidates[0]
	}
	return ""
}

func contractUIDKey(repoPath string, role string, symbolUID string) string {
	return repoPath + "\x00" + role + "\x00" + symbolUID
}

func contractRefKey(repoPath string, role string, filePath string, symbolName string) string {
	return repoPath + "\x00" + role + "\x00" + filePath + "\x00" + symbolName
}

func contractFileKey(repoPath string, role string, filePath string) string {
	return repoPath + "\x00" + role + "\x00" + filePath
}

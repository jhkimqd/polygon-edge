package polybft

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/contract"
)

// ValidatorInfo is data transfer object which holds validator information,
// provided by smart contract
type ValidatorInfo struct {
	Address             ethgo.Address
	Stake               *big.Int
	WithdrawableRewards *big.Int
	IsActive            bool
	IsWhitelisted       bool
}

// SystemState is an interface to interact with the consensus system contracts in the chain
type SystemState interface {
	// GetEpoch retrieves current epoch number from the smart contract
	GetEpoch() (uint64, error)
	// GetNextCommittedIndex retrieves next committed bridge state sync index
	GetNextCommittedIndex() (uint64, error)
	// GetStakeOnValidatorSet retrieves stake of given validator on ValidatorSet contract
	GetStakeOnValidatorSet(validatorAddr types.Address) (*big.Int, error)
}

var _ SystemState = &SystemStateImpl{}

// SystemStateImpl is implementation of SystemState interface
type SystemStateImpl struct {
	validatorContract       *contract.Contract
	sidechainBridgeContract *contract.Contract
}

// NewSystemState initializes new instance of systemState which abstracts smart contracts functions
func NewSystemState(valSetAddr types.Address, stateRcvAddr types.Address, provider contract.Provider) *SystemStateImpl {
	s := &SystemStateImpl{}
	s.validatorContract = contract.NewContract(
		ethgo.Address(valSetAddr),
		contractsapi.ValidatorSet.Abi, contract.WithProvider(provider),
	)
	s.sidechainBridgeContract = contract.NewContract(
		ethgo.Address(stateRcvAddr),
		contractsapi.StateReceiver.Abi,
		contract.WithProvider(provider),
	)

	return s
}

// GetStakeOnValidatorSet retrieves stake of given validator on ValidatorSet contract
func (s *SystemStateImpl) GetStakeOnValidatorSet(validatorAddr types.Address) (*big.Int, error) {
	rawResult, err := s.validatorContract.Call("balanceOf", ethgo.Latest, validatorAddr)
	if err != nil {
		return nil, err
	}

	balance, isOk := rawResult["0"].(*big.Int)
	if !isOk {
		return nil, fmt.Errorf("failed to decode balance")
	}

	return balance, nil
}

// GetEpoch retrieves current epoch number from the smart contract
func (s *SystemStateImpl) GetEpoch() (uint64, error) {
	rawResult, err := s.validatorContract.Call("currentEpochId", ethgo.Latest)
	if err != nil {
		return 0, err
	}

	epochNumber, isOk := rawResult["0"].(*big.Int)

	if !isOk {
		return 0, fmt.Errorf("failed to decode epoch")
	}

	return epochNumber.Uint64(), nil
}

// GetNextCommittedIndex retrieves next committed bridge state sync index
func (s *SystemStateImpl) GetNextCommittedIndex() (uint64, error) {
	rawResult, err := s.sidechainBridgeContract.Call("lastCommittedId", ethgo.Latest)
	if err != nil {
		return 0, err
	}

	nextCommittedIndex, isOk := rawResult["0"].(*big.Int)
	if !isOk {
		return 0, fmt.Errorf("failed to decode next committed index")
	}

	return nextCommittedIndex.Uint64() + 1, nil
}

func buildLogsFromReceipts(entry []*types.Receipt, header *types.Header) []*types.Log {
	var logs []*types.Log

	for _, taskReceipt := range entry {
		for _, taskLog := range taskReceipt.Logs {
			log := new(types.Log)
			*log = *taskLog

			data := map[string]interface{}{
				"Hash":   header.Hash,
				"Number": header.Number,
			}
			log.Data, _ = json.Marshal(&data)
			logs = append(logs, log)
		}
	}

	return logs
}

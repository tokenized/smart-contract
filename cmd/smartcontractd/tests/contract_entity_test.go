package tests

import (
	"bytes"
	"testing"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// TestContractEntities is the entry point for testing contract entity based functions.
func TestContractEntities(t *testing.T) {
	defer tests.Recover(t)

	t.Run("createAssetContract", createAssetContract)
	t.Run("createEntityContract", createEntityContract)
	t.Run("identityContracts", identityContracts)
	t.Run("operatorContracts", operatorContracts)
}

func createAssetContract(t *testing.T) {
	ctx := test.Context

	testSet := []struct {
		name   string
		offer  *actions.ContractOffer
		parent *bitcoin.RawAddress
		valid  bool
	}{
		{
			name: "ValidAsset",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeAsset,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				EntityContract: test.Contract2Key.Address.Bytes(),
			},
			parent: &test.Contract2Key.Address,
			valid:  true,
		},
		{
			name: "EntityWithEntityContract",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeEntity,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				EntityContract: test.Contract2Key.Address.Bytes(),
			},
			parent: &test.Contract2Key.Address,
			valid:  false, // ContractTypeEntity not allowed to include EntityContract field.
		},
		{
			name: "AssetWithoutEntityContract",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeAsset,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
			},
			parent: nil,
			valid:  false, // ContractTypeAsset requires EntityContract field.
		},
		{
			name: "AssetWithServices",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeAsset,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				Services: []*actions.ServiceField{
					&actions.ServiceField{
						Type: 0,
						URL:  "identity.tokenized.com",
					},
				},
			},
			parent: nil,
			valid:  false, // ContractTypeAsset not allowed Services field.
		},
		{
			name: "AssetWithIssuer",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeAsset,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				EntityContract: test.Contract2Key.Address.Bytes(),
				Issuer: &actions.EntityField{
					Name: "Issuer name",
				},
			},
			parent: &test.Contract2Key.Address,
			valid:  false, // ContractTypeAsset not allowed Issuer field.
		},
		{
			name: "EntityWithIssuer",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeEntity,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				Issuer: &actions.EntityField{
					Name: "Issuer name",
				},
			},
			parent: nil,
			valid:  true,
		},
		{
			name: "AssetWithMissingEntity",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeAsset,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				EntityContract: test.Contract2Key.Address.Bytes(),
			},
			parent: nil,
			valid:  false, // Entity contract formation won't exist
		},
		{
			name: "AssetWithInvalidEntityAddress",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeAsset,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				EntityContract: []byte{0x20, 0x30, 0xf8},
			},
			parent: nil,
			valid:  false, // Entity contract address is not valid
		},
	}

	for _, tt := range testSet {
		t.Run(tt.name, func(t *testing.T) {
			if err := resetTest(ctx); err != nil {
				t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
			}

			if tt.parent != nil {
				mockUpOtherContract(t, ctx, *tt.parent, "Test Contract", "I", 1, "John Bitcoin",
					true, true, false, false, false)
			}

			// Create funding tx
			fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100104, issuerKey.Address)

			// Build offer transaction
			offerTx := wire.NewMsgTx(1)

			offerInputHash := fundingTx.TxHash()

			// From issuer (Note: empty sig script)
			offerTx.TxIn = append(offerTx.TxIn, wire.NewTxIn(wire.NewOutPoint(offerInputHash, 0),
				make([]byte, 130)))

			// To contract
			script, _ := test.ContractKey.Address.LockingScript()
			offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(1000, script))

			// Data output
			var err error
			script, err = protocol.Serialize(tt.offer, test.NodeConfig.IsTest)
			if err != nil {
				t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
			}
			offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(0, script))

			offerItx, err := inspector.NewTransactionFromWire(ctx, offerTx, test.NodeConfig.IsTest)
			if err != nil {
				t.Fatalf("\t%s\tFailed to create itx : %v", tests.Failed, err)
			}

			err = offerItx.Promote(ctx, test.RPCNode)
			if err != nil {
				t.Fatalf("\t%s\tFailed to promote itx : %v", tests.Failed, err)
			}

			err = offerItx.Validate(ctx)
			if err != nil {
				t.Fatalf("\t%s\tFailed to validate itx : %v", tests.Failed, err)
			}

			test.RPCNode.SaveTX(ctx, offerTx)

			err = a.Trigger(ctx, "SEE", offerItx)
			if tt.valid {
				if err != nil {
					t.Fatalf("\t%s\tFailed to process contract offer : %s", tests.Failed, err)
				}

				// Check the response
				checkResponse(t, "C2")

				t.Logf("\t%s\tContract offer accepted", tests.Success)

				// Verify data
				cf, err := contract.FetchContractFormation(ctx, test.MasterDB, test.ContractKey.Address,
					test.NodeConfig.IsTest)
				if err != nil {
					t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
				}

				if tt.parent == nil {
					if len(cf.EntityContract) > 0 {
						t.Fatalf("\t%s\tContract parent should be empty : %x", tests.Failed,
							cf.EntityContract)
					}
				} else if !bytes.Equal(cf.EntityContract, tt.parent.Bytes()) {
					t.Fatalf("\t%s\tContract parent incorrect : %x != %x", tests.Failed,
						cf.EntityContract, tt.parent.Bytes())
				}
			} else {
				if err == nil {
					t.Errorf("\t%s\tFailed to reject contract offer", tests.Failed)
				}

				// Check the response
				checkResponse(t, "M2")

				t.Logf("\t%s\tContract offer rejected", tests.Success)
			}
		})
	}
}

func createEntityContract(t *testing.T) {
	ctx := test.Context

	testSet := []struct {
		name  string
		offer *actions.ContractOffer
		valid bool
	}{
		{
			name: "ValidEntity",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeEntity,
				ContractName: "Test Name",
				Issuer: &actions.EntityField{
					Name: "John Bitcoin",
				},
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
			},
			valid: true,
		},
		{
			name: "EntityWithParent",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeEntity,
				ContractName: "Test Name",
				Issuer: &actions.EntityField{
					Name: "John Bitcoin",
				},
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				EntityContract: test.Contract2Key.Address.Bytes(),
			},
			valid: false,
		},
		{
			name: "EntityWithOutIssuer",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeEntity,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
			},
			valid: true,
		},
		{
			name: "EntityWithService",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeEntity,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				Services: []*actions.ServiceField{
					&actions.ServiceField{
						Type:      actions.ServiceTypeIdentityOracle,
						PublicKey: oracleKey.Key.PublicKey().Bytes(),
					},
				},
			},
			valid: true,
		},
		{
			name: "EntityWithInvalidService",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeEntity,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				Services: []*actions.ServiceField{
					&actions.ServiceField{
						Type:      actions.ServiceTypeIdentityOracle,
						PublicKey: []byte{0x03, 0x00, 0xff, 0x1d},
					},
				},
			},
			valid: false,
		},
	}

	for _, tt := range testSet {
		t.Run(tt.name, func(t *testing.T) {
			if err := resetTest(ctx); err != nil {
				t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
			}

			// Create funding tx
			fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100104, issuerKey.Address)

			// Build offer transaction
			offerTx := wire.NewMsgTx(1)

			offerInputHash := fundingTx.TxHash()

			// From issuer (Note: empty sig script)
			offerTx.TxIn = append(offerTx.TxIn, wire.NewTxIn(wire.NewOutPoint(offerInputHash, 0),
				make([]byte, 130)))

			// To contract
			script, _ := test.ContractKey.Address.LockingScript()
			offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(1000, script))

			// Data output
			var err error
			script, err = protocol.Serialize(tt.offer, test.NodeConfig.IsTest)
			if err != nil {
				t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
			}
			offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(0, script))

			offerItx, err := inspector.NewTransactionFromWire(ctx, offerTx, test.NodeConfig.IsTest)
			if err != nil {
				t.Fatalf("\t%s\tFailed to create itx : %v", tests.Failed, err)
			}

			err = offerItx.Promote(ctx, test.RPCNode)
			if err != nil {
				t.Fatalf("\t%s\tFailed to promote itx : %v", tests.Failed, err)
			}

			err = offerItx.Validate(ctx)
			if err != nil {
				t.Fatalf("\t%s\tFailed to validate itx : %v", tests.Failed, err)
			}

			test.RPCNode.SaveTX(ctx, offerTx)

			err = a.Trigger(ctx, "SEE", offerItx)
			if tt.valid {
				if err != nil {
					t.Fatalf("\t%s\tFailed to process contract offer : %s", tests.Failed, err)
				}

				// Check the response
				checkResponse(t, "C2")

				t.Logf("\t%s\tContract offer accepted", tests.Success)

				// Verify data
				cf, err := contract.FetchContractFormation(ctx, test.MasterDB, test.ContractKey.Address,
					test.NodeConfig.IsTest)
				if err != nil {
					t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
				}

				if !tt.offer.Issuer.Equal(cf.Issuer) {
					t.Fatalf("\t%s\tWrong contract issuer : %+v != %+v", tests.Failed, cf.Issuer,
						tt.offer.Issuer)
				}
			} else {
				if err == nil {
					t.Errorf("\t%s\tFailed to reject contract offer", tests.Failed)
				}

				// Check the response
				checkResponse(t, "M2")

				t.Logf("\t%s\tContract offer rejected", tests.Success)
			}
		})
	}
}

func identityContracts(t *testing.T) {
	ctx := test.Context

	idKey, err := bitcoin.GenerateKey(test.NodeConfig.Net)
	if err != nil {
		t.Fatalf("Failed to generate identity key : %s", err)
	}

	idContractKey, err := bitcoin.GenerateKey(test.NodeConfig.Net)
	if err != nil {
		t.Fatalf("Failed to generate identity contract key : %s", err)
	}

	idAddress, err := idContractKey.RawAddress()
	if err != nil {
		t.Fatalf("Failed to generate identity address : %s", err)
	}

	testSet := []struct {
		name          string
		offer         *actions.ContractOffer
		idContractKey *bitcoin.Key
		valid         bool
	}{
		{
			name: "AssetWithIdentity",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeAsset,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				EntityContract: test.Contract2Key.Address.Bytes(),
				Oracles: []*actions.OracleField{
					&actions.OracleField{
						OracleTypes:    []uint32{actions.ServiceTypeIdentityOracle},
						EntityContract: idAddress.Bytes(),
					},
				},
			},
			idContractKey: &idContractKey,
			valid:         true,
		},
		{
			name: "AssetWithMissingIdentity",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeAsset,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				EntityContract: test.Contract2Key.Address.Bytes(),
				Oracles: []*actions.OracleField{
					&actions.OracleField{
						OracleTypes:    []uint32{actions.ServiceTypeIdentityOracle},
						EntityContract: idAddress.Bytes(),
					},
				},
			},
			idContractKey: nil,
			valid:         false, // identity oracle contract won't exist
		},
		{
			name: "AssetWithWrongIdentity",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeAsset,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				EntityContract: test.Contract2Key.Address.Bytes(),
				Oracles: []*actions.OracleField{
					&actions.OracleField{
						OracleTypes:    []uint32{actions.ServiceTypeAuthorityOracle},
						EntityContract: idAddress.Bytes(),
					},
				},
			},
			idContractKey: &idContractKey,
			valid:         false, // oracle contract doesn't support authority oracle type
		},
	}

	for _, tt := range testSet {
		t.Run(tt.name, func(t *testing.T) {
			if err := resetTest(ctx); err != nil {
				t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
			}

			mockUpOtherContract(t, ctx, test.Contract2Key.Address, "Test Contract", "I", 1,
				"John Bitcoin", true, true, false, false, false)

			if tt.idContractKey != nil {
				mockIdentityContract(t, ctx, *tt.idContractKey, idKey.PublicKey(), "C", 1, "John")
			}

			// Create funding tx
			fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100104, issuerKey.Address)

			// Build offer transaction
			offerTx := wire.NewMsgTx(1)

			offerInputHash := fundingTx.TxHash()

			// From issuer (Note: empty sig script)
			offerTx.TxIn = append(offerTx.TxIn, wire.NewTxIn(wire.NewOutPoint(offerInputHash, 0),
				make([]byte, 130)))

			// To contract
			script, _ := test.ContractKey.Address.LockingScript()
			offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(1000, script))

			// Data output
			var err error
			script, err = protocol.Serialize(tt.offer, test.NodeConfig.IsTest)
			if err != nil {
				t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
			}
			offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(0, script))

			offerItx, err := inspector.NewTransactionFromWire(ctx, offerTx, test.NodeConfig.IsTest)
			if err != nil {
				t.Fatalf("\t%s\tFailed to create itx : %v", tests.Failed, err)
			}

			err = offerItx.Promote(ctx, test.RPCNode)
			if err != nil {
				t.Fatalf("\t%s\tFailed to promote itx : %v", tests.Failed, err)
			}

			err = offerItx.Validate(ctx)
			if err != nil {
				t.Fatalf("\t%s\tFailed to validate itx : %v", tests.Failed, err)
			}

			test.RPCNode.SaveTX(ctx, offerTx)

			err = a.Trigger(ctx, "SEE", offerItx)
			if tt.valid {
				if err != nil {
					t.Fatalf("\t%s\tFailed to process contract offer : %s", tests.Failed, err)
				}

				// Check the response
				checkResponse(t, "C2")

				t.Logf("\t%s\tContract offer accepted", tests.Success)

			} else {
				if err == nil {
					t.Errorf("\t%s\tFailed to reject contract offer", tests.Failed)
				}

				// Check the response
				checkResponse(t, "M2")

				t.Logf("\t%s\tChild contract offer rejected", tests.Success)
			}
		})
	}
}

func operatorContracts(t *testing.T) {
	ctx := test.Context

	opServiceKey, err := bitcoin.GenerateKey(test.NodeConfig.Net)
	if err != nil {
		t.Fatalf("Failed to generate operator service key : %s", err)
	}

	opServiceAddress, err := opServiceKey.RawAddress()
	if err != nil {
		t.Fatalf("Failed to generate operator service address : %s", err)
	}

	opContractKey, err := bitcoin.GenerateKey(test.NodeConfig.Net)
	if err != nil {
		t.Fatalf("Failed to generate operator contract key : %s", err)
	}

	opAddress, err := opContractKey.RawAddress()
	if err != nil {
		t.Fatalf("Failed to generate operator address : %s", err)
	}

	otherKey, err := bitcoin.GenerateKey(test.NodeConfig.Net)
	if err != nil {
		t.Fatalf("Failed to generate other key : %s", err)
	}

	otherAddress, err := otherKey.RawAddress()
	if err != nil {
		t.Fatalf("Failed to generate other address : %s", err)
	}

	testSet := []struct {
		name             string
		offer            *actions.ContractOffer
		opContractKey    *bitcoin.Key
		opServiceKey     *bitcoin.Key
		opServiceAddress *bitcoin.RawAddress
		idContractKey    *bitcoin.Key
		valid            bool
	}{
		{
			name: "AssetWithOperator",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeAsset,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				EntityContract:           test.Contract2Key.Address.Bytes(),
				ContractOperatorIncluded: true,
				OperatorEntityContract:   opAddress.Bytes(),
			},
			opContractKey:    &opContractKey,
			opServiceKey:     &opServiceKey,
			opServiceAddress: &opServiceAddress,
			valid:            true,
		},
		{
			name: "EntityWithOperator",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeEntity,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				ContractOperatorIncluded: true,
				OperatorEntityContract:   opAddress.Bytes(),
			},
			opContractKey:    &opContractKey,
			opServiceKey:     &opServiceKey,
			opServiceAddress: &opServiceAddress,
			valid:            true,
		},
		{
			name: "AssetWithMissingOperator",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeAsset,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				EntityContract:           test.Contract2Key.Address.Bytes(),
				ContractOperatorIncluded: true,
				OperatorEntityContract:   opAddress.Bytes(),
			},
			opContractKey: nil,
			opServiceKey:  nil,
			valid:         false, // operator contract won't exist
		},
		{
			name: "EntityWithMissingOperator",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeEntity,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				ContractOperatorIncluded: true,
				OperatorEntityContract:   opAddress.Bytes(),
			},
			opContractKey: nil,
			opServiceKey:  nil,
			valid:         false, // operator contract won't exist
		},
		{
			name: "AssetWithWrongIdentity",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeAsset,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				EntityContract:           test.Contract2Key.Address.Bytes(),
				ContractOperatorIncluded: true,
				OperatorEntityContract:   opAddress.Bytes(),
			},
			idContractKey:    &opContractKey,
			opServiceKey:     &opServiceKey,
			opServiceAddress: &opServiceAddress,
			valid:            false, // operator contract doesn't support operator type
		},
		{
			name: "EntityWithWrongIdentity",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeEntity,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				ContractOperatorIncluded: true,
				OperatorEntityContract:   opAddress.Bytes(),
			},
			idContractKey:    &opContractKey,
			opServiceKey:     &opServiceKey,
			opServiceAddress: &opServiceAddress,
			valid:            false, // operator contract doesn't support operator type
		},
		{
			name: "EntityWrongOperatorInputAddress",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeEntity,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				ContractOperatorIncluded: true,
				OperatorEntityContract:   opAddress.Bytes(),
			},
			opContractKey:    &opContractKey,
			opServiceKey:     &opServiceKey,
			opServiceAddress: &otherAddress,
			valid:            false, // wrong address
		},
		{
			name: "AssetWithWrongIdentity",
			offer: &actions.ContractOffer{
				ContractType: actions.ContractTypeAsset,
				ContractName: "Test Name",
				VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50",
					VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
				EntityContract:           test.Contract2Key.Address.Bytes(),
				ContractOperatorIncluded: true,
				OperatorEntityContract:   opAddress.Bytes(),
			},
			opContractKey:    &opContractKey,
			opServiceKey:     &opServiceKey,
			opServiceAddress: &otherAddress,
			valid:            false, // wrong address
		},
	}

	for _, tt := range testSet {
		t.Run(tt.name, func(t *testing.T) {
			if err := resetTest(ctx); err != nil {
				t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
			}

			mockUpOtherContract(t, ctx, test.Contract2Key.Address, "Test Contract", "I", 1,
				"John Bitcoin", true, true, false, false, false)

			if tt.opContractKey != nil {
				mockOperatorContract(t, ctx, *tt.opContractKey, tt.opServiceKey.PublicKey(), "C", 1,
					"John")
			}

			if tt.idContractKey != nil {
				mockIdentityContract(t, ctx, *tt.idContractKey, tt.opServiceKey.PublicKey(), "C", 1,
					"John")
			}

			// Build offer transaction
			offerTx := wire.NewMsgTx(1)

			// From issuer (Note: empty sig script)
			fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100104, issuerKey.Address)
			offerInputHash := fundingTx.TxHash()
			offerTx.TxIn = append(offerTx.TxIn, wire.NewTxIn(wire.NewOutPoint(offerInputHash, 0),
				make([]byte, 130)))

			// From operator
			if tt.opServiceAddress != nil {
				opFundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100204, *tt.opServiceAddress)
				offerOperatorInputHash := opFundingTx.TxHash()
				offerTx.TxIn = append(offerTx.TxIn, wire.NewTxIn(wire.NewOutPoint(offerOperatorInputHash, 0),
					make([]byte, 130)))
			}

			// To contract
			script, _ := test.ContractKey.Address.LockingScript()
			offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(1000, script))

			// Data output
			var err error
			script, err = protocol.Serialize(tt.offer, test.NodeConfig.IsTest)
			if err != nil {
				t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
			}
			offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(0, script))

			offerItx, err := inspector.NewTransactionFromWire(ctx, offerTx, test.NodeConfig.IsTest)
			if err != nil {
				t.Fatalf("\t%s\tFailed to create itx : %v", tests.Failed, err)
			}

			err = offerItx.Promote(ctx, test.RPCNode)
			if err != nil {
				t.Fatalf("\t%s\tFailed to promote itx : %v", tests.Failed, err)
			}

			err = offerItx.Validate(ctx)
			if err != nil {
				t.Fatalf("\t%s\tFailed to validate itx : %v", tests.Failed, err)
			}

			test.RPCNode.SaveTX(ctx, offerTx)
			t.Logf("Offer Tx : %s", offerTx.String())

			err = a.Trigger(ctx, "SEE", offerItx)
			if tt.valid {
				if err != nil {
					t.Fatalf("\t%s\tFailed to process contract offer : %s", tests.Failed, err)
				}

				// Check the response
				checkResponse(t, "C2")

				t.Logf("\t%s\tContract offer accepted", tests.Success)

				// Verify data
				cf, err := contract.FetchContractFormation(ctx, test.MasterDB, test.ContractKey.Address,
					test.NodeConfig.IsTest)
				if err != nil {
					t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
				}

				if tt.opServiceAddress != nil {
					if !bytes.Equal(cf.OperatorAddress, tt.opServiceAddress.Bytes()) {
						t.Errorf("\t%s\tWrong contract operator : %x != %x", tests.Failed,
							cf.OperatorAddress, opServiceAddress.Bytes())
					}
				}

			} else {
				if err == nil {
					t.Errorf("\t%s\tFailed to reject contract offer", tests.Failed)
				}

				// Check the response
				checkResponse(t, "M2")

				t.Logf("\t%s\tChild contract offer rejected", tests.Success)
			}
		})
	}
}

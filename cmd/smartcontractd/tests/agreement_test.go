package tests

import (
	"context"
	"testing"

	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/smart-contract/internal/agreement"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/permissions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/golang/protobuf/proto"
)

// TestAgreements is the entry point for testing contract based functions.
func TestAgreements(t *testing.T) {
	defer tests.Recover(t)

	t.Run("create", createAgreement)
	t.Run("wrongType", wrongAgreementType)
	t.Run("withDefinitions", agreementWithDefinitions)
	t.Run("missingDefinition", agreementMissingDefinition)
	t.Run("unreferencedDefinition", agreementUnreferencedDefinition)
	t.Run("amend", agreementAmendment)
}

func createAgreement(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}

	mockUpContractForAgreement(t, ctx, "Test Contract", "I", 1, "John Bitcoin", true, true, false,
		false, false)

	// New Body of Agreement Offer
	agreementData := actions.BodyOfAgreementOffer{
		Chapters: []*actions.ChapterField{
			&actions.ChapterField{
				Title: "First Chapter",
			},
		},
		Definitions: []*actions.DefinedTermField{},
	}

	// Create funding tx
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100004, issuerKey.Address)

	// Build offer transaction
	offerTx := wire.NewMsgTx(1)

	offerInputHash := fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	offerTx.TxIn = append(offerTx.TxIn, wire.NewTxIn(wire.NewOutPoint(offerInputHash, 0),
		make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(2000, script))

	// Data output
	var err error
	script, err = protocol.Serialize(&agreementData, test.NodeConfig.IsTest)
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

	t.Logf("Body of agreement offer tx : %s", offerItx.Hash)

	// Resubmit to handler
	if err := a.Trigger(ctx, "SEE", offerItx); err != nil {
		t.Fatalf("\t%s\tFailed to handle agreement offer : %v", tests.Failed, err)
	}

	t.Logf("Body of agreement offer accepted")

	// Check the response
	checkResponse(t, "C7")

	// Verify data
	agree, err := agreement.Retrieve(ctx, test.MasterDB, test.ContractKey.Address)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve agreement : %v", tests.Failed, err)
	}

	if len(agree.Chapters) != 1 {
		t.Fatalf("Wrong chapter count : got %d, want %d", len(agree.Chapters), 1)
	}

	if agree.Chapters[0].Title != "First Chapter" {
		t.Fatalf("Wrong chapter title : got %s, want %s", agree.Chapters[0].Title, "First Chapter")
	}

	if len(agree.Definitions) != 0 {
		t.Fatalf("Wrong definition count : got %d, want %d", len(agree.Definitions), 0)
	}
}

func wrongAgreementType(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}

	mockUpContract(t, ctx, "Test Contract", "I", 1, "John Bitcoin", true, true, false,
		false, false)

	// New Body of Agreement Offer
	agreementData := actions.BodyOfAgreementOffer{
		Chapters: []*actions.ChapterField{
			&actions.ChapterField{
				Title: "First Chapter",
			},
		},
		Definitions: []*actions.DefinedTermField{},
	}

	// Create funding tx
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100004, issuerKey.Address)

	// Build offer transaction
	offerTx := wire.NewMsgTx(1)

	offerInputHash := fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	offerTx.TxIn = append(offerTx.TxIn, wire.NewTxIn(wire.NewOutPoint(offerInputHash, 0),
		make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(2000, script))

	// Data output
	var err error
	script, err = protocol.Serialize(&agreementData, test.NodeConfig.IsTest)
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

	t.Logf("Body of agreement offer tx : %s", offerItx.Hash)

	err = a.Trigger(ctx, "SEE", offerItx)
	if err == nil {
		t.Fatalf("\t%s\tAccepted invalid body of agreement offer", tests.Failed)
	}
	t.Logf("\t%s\tRejected invalid body of agreement offer : %s", tests.Success, err)

	// ********************************************************************************************
	// Check reject response
	if len(responses) != 1 {
		t.Fatalf("\t%s\tHandle body of agreement offer created no reject response", tests.Failed)
	}

	var responseMsg actions.Action
	response := responses[0].Copy()
	responses = nil
	for _, output := range response.TxOut {
		responseMsg, err = protocol.Deserialize(output.PkScript, test.NodeConfig.IsTest)
		if err == nil {
			break
		}
	}
	if responseMsg == nil {
		t.Fatalf("\t%s\tBody of agreement offer response doesn't contain tokenized op return",
			tests.Failed)
	}
	if responseMsg.Code() != "M2" {
		t.Fatalf("\t%s\tBody of agreement offer response not a reject : %s", tests.Failed,
			responseMsg.Code())
	}
	reject, ok := responseMsg.(*actions.Rejection)
	if !ok {
		t.Fatalf("\t%s\tFailed to convert response to rejection", tests.Failed)
	}
	if reject.RejectionCode != actions.RejectionsContractNotPermitted {
		t.Fatalf("\t%s\tWrong reject code for body of agreement offer reject : %d", tests.Failed,
			reject.RejectionCode)
	}

	t.Logf("\t%s\tInvalid body of agreement offer rejection : (%d) %s", tests.Success,
		reject.RejectionCode, reject.Message)
}

func agreementWithDefinitions(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}

	mockUpContractForAgreement(t, ctx, "Test Contract", "I", 1, "John Bitcoin", true, true, false,
		false, false)

	// New Body of Agreement Offer
	agreementData := actions.BodyOfAgreementOffer{
		Chapters: []*actions.ChapterField{
			&actions.ChapterField{
				Title:    "First Chapter",
				Preamble: "This is a chapter that has a [term]() defined.",
			},
		},
		Definitions: []*actions.DefinedTermField{
			&actions.DefinedTermField{
				Term:       "term",
				Definition: "Term is a term for a definition.",
			},
		},
	}

	// Create funding tx
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100004, issuerKey.Address)

	// Build offer transaction
	offerTx := wire.NewMsgTx(1)

	offerInputHash := fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	offerTx.TxIn = append(offerTx.TxIn, wire.NewTxIn(wire.NewOutPoint(offerInputHash, 0),
		make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(2000, script))

	// Data output
	var err error
	script, err = protocol.Serialize(&agreementData, test.NodeConfig.IsTest)
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

	t.Logf("Body of agreement offer tx : %s", offerItx.Hash)

	// Resubmit to handler
	if err := a.Trigger(ctx, "SEE", offerItx); err != nil {
		t.Fatalf("\t%s\tFailed to handle agreement offer : %v", tests.Failed, err)
	}

	t.Logf("Body of agreement offer accepted")

	// Check the response
	checkResponse(t, "C7")

	// Verify data
	agree, err := agreement.Retrieve(ctx, test.MasterDB, test.ContractKey.Address)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve agreement : %v", tests.Failed, err)
	}

	if len(agree.Chapters) != 1 {
		t.Fatalf("Wrong chapter count : got %d, want %d", len(agree.Chapters), 1)
	}

	if agree.Chapters[0].Title != "First Chapter" {
		t.Fatalf("Wrong chapter title : got %s, want %s", agree.Chapters[0].Title, "First Chapter")
	}

	if len(agree.Definitions) != 1 {
		t.Fatalf("Wrong definition count : got %d, want %d", len(agree.Definitions), 1)
	}

	if agree.Definitions[0].Term != "term" {
		t.Fatalf("Wrong definition term : got %s, want %s", agree.Definitions[0].Term, "term")
	}

	if agree.Definitions[0].Definition != "Term is a term for a definition." {
		t.Fatalf("Wrong definition : got %s, want %s", agree.Definitions[0].Definition,
			"Term is a term for a definition.")
	}
}

func agreementMissingDefinition(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}

	mockUpContractForAgreement(t, ctx, "Test Contract", "I", 1, "John Bitcoin", true, true, false,
		false, false)

	// New Body of Agreement Offer
	agreementData := actions.BodyOfAgreementOffer{
		Chapters: []*actions.ChapterField{
			&actions.ChapterField{
				Title:    "First Chapter",
				Preamble: "This is a chapter that has a [term]() defined.",
			},
		},
	}

	// Create funding tx
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100004, issuerKey.Address)

	// Build offer transaction
	offerTx := wire.NewMsgTx(1)

	offerInputHash := fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	offerTx.TxIn = append(offerTx.TxIn, wire.NewTxIn(wire.NewOutPoint(offerInputHash, 0),
		make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(2000, script))

	// Data output
	var err error
	script, err = protocol.Serialize(&agreementData, test.NodeConfig.IsTest)
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

	t.Logf("Body of agreement offer tx : %s", offerItx.Hash)

	err = a.Trigger(ctx, "SEE", offerItx)
	if err == nil {
		t.Fatalf("\t%s\tAccepted invalid body of agreement offer", tests.Failed)
	}
	t.Logf("\t%s\tRejected invalid body of agreement offer : %s", tests.Success, err)

	// ********************************************************************************************
	// Check reject response
	if len(responses) != 1 {
		t.Fatalf("\t%s\tHandle body of agreement offer created no reject response", tests.Failed)
	}

	var responseMsg actions.Action
	response := responses[0].Copy()
	responses = nil
	for _, output := range response.TxOut {
		responseMsg, err = protocol.Deserialize(output.PkScript, test.NodeConfig.IsTest)
		if err == nil {
			break
		}
	}
	if responseMsg == nil {
		t.Fatalf("\t%s\tBody of agreement offer response doesn't contain tokenized op return",
			tests.Failed)
	}
	if responseMsg.Code() != "M2" {
		t.Fatalf("\t%s\tBody of agreement offer response not a reject : %s", tests.Failed,
			responseMsg.Code())
	}
	reject, ok := responseMsg.(*actions.Rejection)
	if !ok {
		t.Fatalf("\t%s\tFailed to convert response to rejection", tests.Failed)
	}
	if reject.RejectionCode != actions.RejectionsMsgMalformed {
		t.Fatalf("\t%s\tWrong reject code for body of agreement offer reject : %d", tests.Failed,
			reject.RejectionCode)
	}

	t.Logf("\t%s\tInvalid body of agreement offer rejection : (%d) %s", tests.Success,
		reject.RejectionCode, reject.Message)
}

func agreementUnreferencedDefinition(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}

	mockUpContractForAgreement(t, ctx, "Test Contract", "I", 1, "John Bitcoin", true, true, false,
		false, false)

	// New Body of Agreement Offer
	agreementData := actions.BodyOfAgreementOffer{
		Chapters: []*actions.ChapterField{
			&actions.ChapterField{
				Title:    "First Chapter",
				Preamble: "This is a chapter that has a term defined but not linked.",
			},
		},
		Definitions: []*actions.DefinedTermField{
			&actions.DefinedTermField{
				Term:       "term",
				Definition: "Term is a term for a definition.",
			},
		},
	}

	// Create funding tx
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100004, issuerKey.Address)

	// Build offer transaction
	offerTx := wire.NewMsgTx(1)

	offerInputHash := fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	offerTx.TxIn = append(offerTx.TxIn, wire.NewTxIn(wire.NewOutPoint(offerInputHash, 0),
		make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(2000, script))

	// Data output
	var err error
	script, err = protocol.Serialize(&agreementData, test.NodeConfig.IsTest)
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

	t.Logf("Body of agreement offer tx : %s", offerItx.Hash)

	err = a.Trigger(ctx, "SEE", offerItx)
	if err == nil {
		t.Fatalf("\t%s\tAccepted invalid body of agreement offer", tests.Failed)
	}
	t.Logf("\t%s\tRejected invalid body of agreement offer : %s", tests.Success, err)

	// ********************************************************************************************
	// Check reject response
	if len(responses) != 1 {
		t.Fatalf("\t%s\tHandle body of agreement offer created no reject response", tests.Failed)
	}

	var responseMsg actions.Action
	response := responses[0].Copy()
	responses = nil
	for _, output := range response.TxOut {
		responseMsg, err = protocol.Deserialize(output.PkScript, test.NodeConfig.IsTest)
		if err == nil {
			break
		}
	}
	if responseMsg == nil {
		t.Fatalf("\t%s\tBody of agreement offer response doesn't contain tokenized op return",
			tests.Failed)
	}
	if responseMsg.Code() != "M2" {
		t.Fatalf("\t%s\tBody of agreement offer response not a reject : %s", tests.Failed,
			responseMsg.Code())
	}
	reject, ok := responseMsg.(*actions.Rejection)
	if !ok {
		t.Fatalf("\t%s\tFailed to convert response to rejection", tests.Failed)
	}
	if reject.RejectionCode != actions.RejectionsMsgMalformed {
		t.Fatalf("\t%s\tWrong reject code for body of agreement offer reject : %d", tests.Failed,
			reject.RejectionCode)
	}

	t.Logf("\t%s\tInvalid body of agreement offer rejection : (%d) %s", tests.Success,
		reject.RejectionCode, reject.Message)
}

func agreementAmendment(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}

	mockUpContractForAgreement(t, ctx, "Test Contract", "I", 1, "John Bitcoin", true, true, true,
		false, false)
	mockUpAgreement(t, ctx,
		[]*actions.ChapterField{
			&actions.ChapterField{
				Title:    "First Chapter",
				Preamble: "This is a chapter that has a [term]() defined.",
			},
		},
		[]*actions.DefinedTermField{
			&actions.DefinedTermField{
				Term:       "term",
				Definition: "Term is a term for a definition.",
			},
		})

	chaptersFIP := permissions.FieldIndexPath{
		actions.BodyOfAgreementFieldChapters,
		1, // add as second element
	}
	chaptersFIPBytes, err := chaptersFIP.Bytes()
	if err != nil {
		t.Fatalf("Failed to get bytes for chapter field index path : %s", err)
	}

	newChapter := &actions.ChapterField{
		Title:    "Second Chapter",
		Preamble: "This is the second chapter",
	}

	newChapterBytes, err := proto.Marshal(newChapter)
	if err != nil {
		t.Fatalf("Failed to marshal new chapter : %s", err)
	}

	// New Body of Agreement Offer
	amendmentData := actions.BodyOfAgreementAmendment{
		Revision: 0,
		Amendments: []*actions.AmendmentField{
			&actions.AmendmentField{
				FieldIndexPath: chaptersFIPBytes,
				Operation:      1, // Add item
				Data:           newChapterBytes,
			},
		},
	}

	// Create funding tx
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100004, issuerKey.Address)

	// Build amendment transaction
	amendmentTx := wire.NewMsgTx(1)

	offerInputHash := fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	amendmentTx.TxIn = append(amendmentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(offerInputHash, 0),
		make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(3000, script))

	// Data output
	script, err = protocol.Serialize(&amendmentData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize amendment : %v", tests.Failed, err)
	}
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(0, script))

	amendmentItx, err := inspector.NewTransactionFromWire(ctx, amendmentTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create itx : %v", tests.Failed, err)
	}

	if err := amendmentItx.Promote(ctx, test.RPCNode); err != nil {
		t.Fatalf("\t%s\tFailed to promote itx : %v", tests.Failed, err)
	}

	if err := amendmentItx.Validate(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to validate itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, amendmentTx)

	t.Logf("Body of agreement amendment tx : %s", amendmentItx.Hash)

	// Resubmit to handler
	if err := a.Trigger(ctx, "SEE", amendmentItx); err != nil {
		t.Fatalf("\t%s\tFailed to handle agreement amendment : %v", tests.Failed, err)
	}

	t.Logf("Body of agreement amendment accepted")

	// Check the response
	checkResponse(t, "C7")

	// Verify data
	agree, err := agreement.Retrieve(ctx, test.MasterDB, test.ContractKey.Address)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve agreement : %v", tests.Failed, err)
	}

	if len(agree.Chapters) != 2 {
		t.Fatalf("Wrong chapter count : got %d, want %d", len(agree.Chapters), 2)
	}

	if agree.Chapters[0].Title != "First Chapter" {
		t.Fatalf("Wrong chapter title : got %s, want %s", agree.Chapters[0].Title, "First Chapter")
	}

	if len(agree.Definitions) != 1 {
		t.Fatalf("Wrong definition count : got %d, want %d", len(agree.Definitions), 1)
	}

	if agree.Definitions[0].Term != "term" {
		t.Fatalf("Wrong definition term : got %s, want %s", agree.Definitions[0].Term, "term")
	}

	if agree.Definitions[0].Definition != "Term is a term for a definition." {
		t.Fatalf("Wrong definition : got %s, want %s", agree.Definitions[0].Definition,
			"Term is a term for a definition.")
	}
}

func mockUpAgreement(t testing.TB, ctx context.Context, chapters []*actions.ChapterField,
	definitions []*actions.DefinedTermField) (*state.Agreement, *actions.BodyOfAgreementFormation) {

	agreementData := &state.Agreement{
		CreatedAt: protocol.CurrentTimestamp(),
		UpdatedAt: protocol.CurrentTimestamp(),
	}

	agreementFormation := &actions.BodyOfAgreementFormation{
		Chapters:    chapters,
		Definitions: definitions,
	}

	if err := node.Convert(ctx, agreementFormation, agreementData); err != nil {
		t.Fatalf("Failed to convert agreement : %s", err)
	}

	if err := agreement.Save(ctx, test.MasterDB, test.ContractKey.Address,
		agreementData); err != nil {
		t.Fatalf("Failed to save agreement : %s", err)
	}

	return agreementData, agreementFormation
}

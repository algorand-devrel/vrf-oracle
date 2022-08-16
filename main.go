package main

import (
	"context"
	"crypto/sha512"
	"encoding/binary"
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/barnjamin/vrf-oracle/vrfproducers/algorand"

	"github.com/algorand/go-algorand-sdk/abi"
	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/future"
	"github.com/algorand/go-algorand-sdk/types"
	"github.com/algorand/go-algorand/rpcs"
	"github.com/algorand/indexer/fetcher"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.New()

	algodAddress = "http://localhost:4001"
	algodToken   = strings.Repeat("a", 64) // contents of algod.token

	algodClient *algod.Client
	vrfp        *algorand.AlgorandVRFProducer

	appId uint64

	accts    []crypto.Account
	contract *abi.Contract
)

func main() {
	var err error

	// Create an algod client
	algodClient, err = algod.MakeClient(algodAddress, algodToken)
	if err != nil {
		return
	}

	// Get signing accounts
	accts, err = GetAccounts()
	if err != nil {
		log.Fatalf("Failed to get accounts: %+v", err)
	}

	// Get the app id from local
	appId = getAppId()

	// Set it global
	contract = setContract()

	// Setup vrf using our main acct
	vrfp = algorand.New(accts[0].PublicKey[:], accts[0].PrivateKey[:])
	if err != nil {
		log.Fatalf("Failed to create producer: %+v", err)
	}

	// Set up fetcher to get new blocks
	f, err := fetcher.ForNetAndToken(algodAddress, algodToken, log)
	if err != nil {
		log.Fatalf("Failed to create fetcher: %+v", err)
	}
	f.SetNextRound(1500)
	f.SetBlockHandler(handler)

	log.Fatalf("%v", f.Run(context.Background()))
}

func handler(ctx context.Context, cert *rpcs.EncodedBlockCert) error {
	//  Get block seed
	var block_seed = cert.Block.Seed()
	var round = cert.Block.Round()

	// Build input to vrf
	buff := make([]byte, 32+8)
	binary.BigEndian.PutUint64(buff, uint64(round))
	copy(buff[8:], block_seed[:])
	vrfInput := sha512.Sum512_256(buff[:])

	_, proof := vrfp.Prove(vrfInput[:])

	// TODO: Can i get this from the block?
	sp, err := algodClient.SuggestedParams().Do(context.Background())
	if err != nil {
		return err
	}

	sp.LastRoundValid = sp.FirstRoundValid + 10
	if round < 100 {
		return nil
	}

	// Skipping error checks below during AddMethodCall and txn create
	var atc = future.AtomicTransactionComposer{}

	signer := future.BasicAccountTransactionSigner{Account: accts[0]}

	ingest_meth, err := contract.GetMethodByName("ingest")
	if err != nil {
		return err
	}

	mcp := future.AddMethodCallParams{
		AppID:           appId,
		Sender:          accts[0].Address,
		Method:          ingest_meth,
		SuggestedParams: sp,
		OnComplete:      types.NoOpOC,
		Signer:          signer,
		MethodArgs:      []interface{}{int(round), proof[:]},
	}

	// add call to update method
	if err = atc.AddMethodCall(mcp); err != nil {
		log.Printf("Cant add method: %+v", err)
		return err
	}

	noop_meth, err := contract.GetMethodByName("noop")
	if err != nil {
		return err
	}
	for x := 0; x < 10; x++ {
		noop_mcp := future.AddMethodCallParams{
			AppID:           appId,
			Sender:          accts[0].Address,
			Method:          noop_meth,
			SuggestedParams: sp,
			OnComplete:      types.NoOpOC,
			Signer:          signer,
			Note:            []byte(strconv.Itoa(x)),
		}
		if err = atc.AddMethodCall(noop_mcp); err != nil {
			log.Printf("Cant add dummy method: %+v", err)
			return err
		}
	}

	log.Printf("Executing and waiting...")

	// execute
	ret, err := atc.Execute(algodClient, context.Background(), 2)
	if err != nil {
		return err
	}

	// Print returned stuff
	for _, r := range ret.MethodResults {
		if r.DecodeError != nil {
			log.Errorf("Failed to decode: %+v", r.RawReturnValue)
		} else if r.ReturnValue != nil {
			log.Printf("%s:  %+v", r.TxID, r.ReturnValue)
		}
	}

	return nil
}

func setContract() *abi.Contract {
	f, err := os.Open("contract.json")
	if err != nil {
		log.Fatalf("Failed to open contract file: %+v", err)
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("Failed to read file: %+v", err)
	}

	contract := &abi.Contract{}
	if err = json.Unmarshal(b, contract); err != nil {
		log.Fatalf("Failed to marshal contract: %+v", err)
	}

	return contract
}

func getAppId() uint64 {
	b, err := ioutil.ReadFile(".app_id")
	if err != nil {
		log.Fatalf("Couldnt read app id file: %+v", err)
	}
	parsedAppId, err := strconv.Atoi(string(b))
	if err != nil {
		log.Fatalf("Failed to read app id: %+v", err)
	}

	return uint64(parsedAppId)
}

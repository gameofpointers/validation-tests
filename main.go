package main

import (
	"context"
	"log"
	"math/big"
	"strconv"
	"sync"

	"github.com/dominant-strategies/go-quai/cmd/utils"
	"github.com/dominant-strategies/go-quai/common"
	"github.com/dominant-strategies/go-quai/consensus/misc"
	"github.com/dominant-strategies/go-quai/core/rawdb"
	"github.com/dominant-strategies/go-quai/core/types"
	"github.com/dominant-strategies/go-quai/params"
	"github.com/dominant-strategies/go-quai/quaiclient/ethclient"
)

func main() {
	wsClientCyprus1, err := ethclient.Dial("ws://127.0.0.1:" + strconv.Itoa(utils.GetWSPort(common.Location{0, 0})))
	if err != nil {
		log.Printf("Failed to connect to the Ethereum WebSocket client: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go CheckIfMinDenominationsExists(wsClientCyprus1, "Cyprus1", &wg)
	wg.Wait()
}

func CheckIfMinDenominationsExists(wsClient *ethclient.Client, node string, wg *sync.WaitGroup) {
	latestNumber, err := wsClient.BlockNumber(context.Background())
	if err != nil {
		log.Fatalf("Failed to get latest block number: %v", err)
	}
	log.Printf("%s: Latest block number: %d\n", node, latestNumber)
	if latestNumber == 0 {
		wg.Done()
		return
	}

	// denominations for which we will check for
	checkDenominations := []int{0, 1, 2, 3, 4, 5}

	coinbaseConversionHashes := make(map[int][][]byte)
	denominationHashes := make(map[int][][]byte)

	for _, denomination := range checkDenominations {
		coinbaseConversionHashes[denomination] = make([][]byte, 0)
		denominationHashes[denomination] = make([][]byte, 0)
	}

	// every denomination that was created below latestNumber-20 should not exist
	// in the database
	for k := 1; k <= int(latestNumber-types.TrimDepths[0]); k++ {

		block, err := wsClient.BlockByNumber(context.Background(), new(big.Int).SetInt64(int64(k)))
		if err != nil {
			log.Fatalf("Failed to get block: %v", err)
		}

		// go through all the external transactions
		for _, tx := range block.Body().Transactions() {

			// If the tx to is in Qi ledger scope, we can call the find min denominations
			// generate the outpoints
			if tx.Type() == types.ExternalTxType && tx.To().IsInQiLedgerScope() {

				var value *big.Int
				if types.IsCoinBaseTx(tx) {
					value = params.CalculateCoinbaseValueWithLockup(tx.Value(), 0)
				} else if types.IsConversionTx(tx) {
					value = tx.Value()
				}

				denominations := misc.FindMinDenominations(value)

				outputIndex := 0
				for denomination := types.MaxDenomination; denomination >= 0; denomination-- {
					if denominations[uint8(denomination)] == 0 {
						continue
					}
					for j := uint64(0); j < denominations[uint8(denomination)]; j++ {
						if outputIndex >= types.MaxOutputIndex {
							break
						}

						if k <= int(latestNumber-types.TrimDepths[uint8(denomination)]) {
							_, exists := coinbaseConversionHashes[denomination]
							if exists {
								coinbaseConversionHashes[denomination] = append(coinbaseConversionHashes[denomination], rawdb.UtxoKey(tx.Hash(), uint16(outputIndex)))
							}
						}
						outputIndex++
					}
				}
			}

			// Normal Qi Tx transaction
			if tx.Type() == types.QiTxType {
				for i, out := range tx.TxOut() {
					if common.BytesToAddress(out.Address, common.Location{0, 0}).IsInQiLedgerScope() {
						if k <= int(latestNumber-types.TrimDepths[uint8(out.Denomination)]) {
							_, exists := denominationHashes[int(out.Denomination)]
							if exists {
								denominationHashes[int(out.Denomination)] = append(denominationHashes[int(out.Denomination)], rawdb.UtxoKey(tx.Hash(), uint16(i)))
							}
						}
					}
				}
			}

		}
	}

	for denomination := range checkDenominations {
		log.Println("Number of min denominations till denomination", denomination, "block number", latestNumber-20, "coinbase/conversion tx", len(coinbaseConversionHashes[int(denomination)]), "qi tx", len(denominationHashes[int(denomination)]))
	}

	existingUtxos := make(map[int]int)
	coinbaseConversionUtxos := make(map[int]int)

	for i := range checkDenominations {
		for _, outpoint := range denominationHashes[i] {
			txHash, index, _ := rawdb.ReverseUtxoKey(outpoint)
			utxo, _ := wsClient.GetUTXO(context.Background(), txHash, index)
			if utxo != nil {
				existingUtxos[i]++
			}
		}

		for _, outpoint := range coinbaseConversionHashes[i] {
			txHash, index, _ := rawdb.ReverseUtxoKey(outpoint)
			utxo, _ := wsClient.GetUTXO(context.Background(), txHash, index)
			if utxo != nil {
				coinbaseConversionUtxos[i]++
			}
		}
	}

	for i := range checkDenominations {

		log.Println("checking denomination ", i)

		if existingUtxos[i] == 0 {
			log.Println("Test passed, no min denomination utxos exist")
		} else {
			log.Println("Test failed, Number of existing utxos denomination", i, "count", existingUtxos[i])
		}

		if coinbaseConversionUtxos[i] != len(coinbaseConversionHashes[i]) {
			log.Println("Test failed, some of the coinbase or conversion outpoints got trimmed expected", len(coinbaseConversionHashes[i]), "got", coinbaseConversionUtxos[i])
		} else {
			log.Println("Test passed, none of the coinbase or conversion outpoints got trimmed")
		}
	}

	wg.Done()
}

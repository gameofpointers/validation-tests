package main

import (
	"context"
	"log"
	"math/big"
	"sync"

	"github.com/dominant-strategies/go-quai/common"
	"github.com/dominant-strategies/go-quai/consensus/misc"
	"github.com/dominant-strategies/go-quai/core/rawdb"
	"github.com/dominant-strategies/go-quai/core/types"
	"github.com/dominant-strategies/go-quai/params"
	"github.com/dominant-strategies/go-quai/quaiclient/ethclient"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

func main() {
	wsClientCyprus1, err := ethclient.Dial("ws://127.0.0.1:8200")
	if err != nil {
		log.Printf("Failed to connect to the Ethereum WebSocket client: %v", err)
	}

	// Test 1: checks if there are any min denominations exists when they were
	// supposed to be trimmed
	// var wg sync.WaitGroup
	// wg.Add(1)
	// go CheckIfMinDenominationsExists(wsClientCyprus1, "Cyprus1", &wg)
	// wg.Wait()

	// Test 2: Goes through the chain and adds up all the quai that was emitted
	// for coinbase upto a given block
	// CheckEmissionNumbers(wsClientCyprus1, "Cyprus1")

	// Test 3: checks the gas limit and plots it at the end
	// CheckGasLimit(wsClientCyprus1)

	// Test 4: Verifies the locked quai indexing
	ValidateQuaiLockedBalance(wsClientCyprus1)

}

func ValidateQuaiLockedBalance(wsClient *ethclient.Client) {
	// since none of the coinbases get unlocked in the first two weeks
	// accumulate all the addresses and their corresponsing balance and check
	// against the api
	blockNumber, err := wsClient.BlockNumber(context.Background())
	if err != nil {
		log.Println("Error getting the block number", err)
		return
	}
	log.Println("Current block number is ", blockNumber)

	addressToBalance := make(map[string]*big.Int)
	for i := 1; i <= int(blockNumber); i++ {
		block, err := wsClient.BlockByNumber(context.Background(), big.NewInt(int64(i)))
		if err != nil {
			log.Println("Error getting block ", i, err)
			return
		}

		for _, tx := range block.Transactions() {
			if tx.Type() == types.ExternalTxType {
				balance, exists := addressToBalance[tx.From(common.Location{0, 0}).Hex()]
				if exists {
					addressToBalance[tx.From(common.Location{0, 0}).Hex()] = new(big.Int).Add(tx.Value(), balance)
				} else {
					addressToBalance[tx.From(common.Location{0, 0}).Hex()] = tx.Value()
				}
			}
		}
	}

	for addr, balance := range addressToBalance {
		// get the locked balance and compare
		lockedBalance, err := wsClient.GetLockedBalance(context.Background(), common.HexToAddress(addr, common.Location{0, 0}))
		if err != nil {
			log.Println("Error getting the locked balance", err)
		}
		unlockedBalance, err := wsClient.BalanceAt(context.Background(), common.NewMixedcaseAddress(common.HexToAddress(addr, common.Location{0, 0})), new(big.Int).SetInt64(int64(blockNumber)))
		if err != nil {
			log.Println("Error getting the unlocked balance", err)
		}
		log.Println("Address", addr, "Balance", balance, "locked balance", lockedBalance, "unlocked balance", unlockedBalance)
		if balance.Cmp(new(big.Int).Add(lockedBalance, unlockedBalance)) != 0 {
			log.Println("Indexed locked balance is not equal to the true locked balance", balance, new(big.Int).Add(lockedBalance, unlockedBalance))
		}
	}
}

func CheckGasLimit(wsClient *ethclient.Client) {

	pointXY := make([]plotter.XY, 0)

	for i := 1; i < 2000; i++ {
		block, err := wsClient.BlockByNumber(context.Background(), big.NewInt(int64(i)))
		if err != nil {
			log.Println("Error getting block ", i, err)
			return
		}

		pointXY = append(pointXY, plotter.XY{X: float64(i), Y: float64(block.GasLimit())})
	}
	// Create a new plot
	p := plot.New()

	p.Title.Text = "Gas Limit graph"
	p.X.Label.Text = "X-axis"
	p.Y.Label.Text = "Y-axis"

	// Convert to plotter.XYs format
	points := make(plotter.XYs, len(pointXY))
	copy(points, pointXY)

	// Add a line plot
	err := plotutil.AddLinePoints(p, "Data", points)
	if err != nil {
		log.Fatal(err)
	}

	// Save to a PNG file
	err = p.Save(6*vg.Inch, 4*vg.Inch, "gasLimit.png")
	if err != nil {
		log.Fatal(err)
	}
}

func CheckEmissionNumbers(wsClient *ethclient.Client, node string) {
	totalBalance := big.NewInt(0)
	// go through the first 1000 blocks and report the total amount of quai given out
	for i := 1; i < 211; i++ {
		block, err := wsClient.BlockByNumber(context.Background(), big.NewInt(int64(i)))
		if err != nil {
			log.Println("Error getting block ", i)
			return
		}
		// go through the transactions list and add all the value from the transactions that are etx
		for _, tx := range block.Transactions() {
			if tx.Type() == types.ExternalTxType {
				totalBalance.Add(totalBalance, tx.Value())
			}
		}
	}

	log.Println("total emission after 4000 blocks", totalBalance)

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
					value = params.CalculateCoinbaseValueWithLockup(tx.Value(), 0, block.NumberU64(common.ZONE_CTX))
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

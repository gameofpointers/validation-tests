package main

import (
	"context"
	"log"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/dominant-strategies/go-quai/cmd/utils"
	"github.com/dominant-strategies/go-quai/common"
	"github.com/dominant-strategies/go-quai/quaiclient/ethclient"
)

var (
	numNodes = 9
)

func main() {
	startTime := time.Now()
	wsClientCyprus1, err := ethclient.Dial("ws://127.0.0.1:" + strconv.Itoa(utils.GetWSPort(common.Location{0, 0})))
	if err != nil {
		log.Printf("Failed to connect to the Ethereum WebSocket client: %v", err)
	}
	wsClientCyprus2, err := ethclient.Dial("ws://127.0.0.1:" + strconv.Itoa(utils.GetWSPort(common.Location{0, 1})))
	if err != nil {
		log.Printf("Failed to connect to the Ethereum WebSocket client: %v", err)
	}
	wsClientCyprus3, err := ethclient.Dial("ws://127.0.0.1:" + strconv.Itoa(utils.GetWSPort(common.Location{0, 2})))
	if err != nil {
		log.Printf("Failed to connect to the Ethereum WebSocket client: %v", err)
	}
	wsClientPaxos1, err := ethclient.Dial("ws://127.0.0.1:" + strconv.Itoa(utils.GetWSPort(common.Location{1, 0})))
	if err != nil {
		log.Printf("Failed to connect to the Ethereum WebSocket client: %v", err)
	}
	wsClientPaxos2, err := ethclient.Dial("ws://127.0.0.1:" + strconv.Itoa(utils.GetWSPort(common.Location{1, 1})))
	if err != nil {
		log.Printf("Failed to connect to the Ethereum WebSocket client: %v", err)
	}
	wsClientPaxos3, err := ethclient.Dial("ws://127.0.0.1:" + strconv.Itoa(utils.GetWSPort(common.Location{1, 2})))
	if err != nil {
		log.Printf("Failed to connect to the Ethereum WebSocket client: %v", err)
	}
	wsClientHydra1, err := ethclient.Dial("ws://127.0.0.1:" + strconv.Itoa(utils.GetWSPort(common.Location{2, 0})))
	if err != nil {
		log.Printf("Failed to connect to the Ethereum WebSocket client: %v", err)
	}
	wsClientHydra2, err := ethclient.Dial("ws://127.0.0.1:" + strconv.Itoa(utils.GetWSPort(common.Location{2, 1})))
	if err != nil {
		log.Printf("Failed to connect to the Ethereum WebSocket client: %v", err)
	}
	wsClientHydra3, err := ethclient.Dial("ws://127.0.0.1:" + strconv.Itoa(utils.GetWSPort(common.Location{2, 2})))
	if err != nil {
		log.Printf("Failed to connect to the Ethereum WebSocket client: %v", err)
	}

	outboundsCyprus1 := 0
	inboundsCyprus1 := 0
	outboundsCyprus2 := 0
	inboundsCyprus2 := 0
	outboundsCyprus3 := 0
	inboundsCyprus3 := 0
	outboundsPaxos1 := 0
	inboundsPaxos1 := 0
	outboundsPaxos2 := 0
	inboundsPaxos2 := 0
	outboundsPaxos3 := 0
	inboundsPaxos3 := 0
	outboundsHydra1 := 0
	inboundsHydra1 := 0
	outboundsHydra2 := 0
	inboundsHydra2 := 0
	outboundsHydra3 := 0
	inboundsHydra3 := 0

	var wg sync.WaitGroup
	wg.Add(9)
	go GetInboundsOutbounds(wsClientCyprus1, "Cyprus1", &inboundsCyprus1, &outboundsCyprus1, &wg)
	go GetInboundsOutbounds(wsClientCyprus2, "Cyprus2", &inboundsCyprus2, &outboundsCyprus2, &wg)
	go GetInboundsOutbounds(wsClientCyprus3, "Cyprus3", &inboundsCyprus3, &outboundsCyprus3, &wg)
	go GetInboundsOutbounds(wsClientPaxos1, "Paxos1", &inboundsPaxos1, &outboundsPaxos1, &wg)
	go GetInboundsOutbounds(wsClientPaxos2, "Paxos2", &inboundsPaxos2, &outboundsPaxos2, &wg)
	go GetInboundsOutbounds(wsClientPaxos3, "Paxos3", &inboundsPaxos3, &outboundsPaxos3, &wg)
	go GetInboundsOutbounds(wsClientHydra1, "Hydra1", &inboundsHydra1, &outboundsHydra1, &wg)
	go GetInboundsOutbounds(wsClientHydra2, "Hydra2", &inboundsHydra2, &outboundsHydra2, &wg)
	go GetInboundsOutbounds(wsClientHydra3, "Hydra3", &inboundsHydra3, &outboundsHydra3, &wg)
	wg.Wait()
	log.Printf("Took %f s\n", time.Since(startTime).Seconds())
	log.Printf("Cyprus1: Inbounds: %d, Outbounds: %d\n", inboundsCyprus1, outboundsCyprus1)
	log.Printf("Cyprus2: Inbounds: %d, Outbounds: %d\n", inboundsCyprus2, outboundsCyprus2)
	log.Printf("Cyprus3: Inbounds: %d, Outbounds: %d\n", inboundsCyprus3, outboundsCyprus3)
	log.Printf("Paxos1: Inbounds: %d, Outbounds: %d\n", inboundsPaxos1, outboundsPaxos1)
	log.Printf("Paxos2: Inbounds: %d, Outbounds: %d\n", inboundsPaxos2, outboundsPaxos2)
	log.Printf("Paxos3: Inbounds: %d, Outbounds: %d\n", inboundsPaxos3, outboundsPaxos3)
	log.Printf("Hydra1: Inbounds: %d, Outbounds: %d\n", inboundsHydra1, outboundsHydra1)
	log.Printf("Hydra2: Inbounds: %d, Outbounds: %d\n", inboundsHydra2, outboundsHydra2)
	log.Printf("Hydra3: Inbounds: %d, Outbounds: %d\n", inboundsHydra3, outboundsHydra3)
	totalOutbounds := outboundsCyprus1 + outboundsCyprus2 + outboundsCyprus3 + outboundsPaxos1 + outboundsPaxos2 + outboundsPaxos3 + outboundsHydra1 + outboundsHydra2 + outboundsHydra3
	totalInbounds := inboundsCyprus1 + inboundsCyprus2 + inboundsCyprus3 + inboundsPaxos1 + inboundsPaxos2 + inboundsPaxos3 + inboundsHydra1 + inboundsHydra2 + inboundsHydra3
	log.Printf("Total Inbounds: %d, Total Outbounds: %d\n", totalInbounds, totalOutbounds)
	log.Printf("Missing: %d\n", totalOutbounds-totalInbounds)
}

func GetInboundsOutbounds(wsClient *ethclient.Client, node string, inbounds *int, outbounds *int, wg *sync.WaitGroup) {
	if numNodes == 2 && !(node == "Cyprus1" || node == "Cyprus2") {
		wg.Done()
		return
	}
	latestNumber, err := wsClient.BlockNumber(context.Background())
	if err != nil {
		log.Fatalf("Failed to get latest block number: %v", err)
	}
	log.Printf("%s: Latest block number: %d\n", node, latestNumber)
	if latestNumber == 0 {
		wg.Done()
		return
	}
	for {

		block, err := wsClient.BlockByNumber(context.Background(), new(big.Int).SetUint64(latestNumber))
		if err != nil {
			log.Fatalf("Failed to get block: %v", err)
		}
		*outbounds += len(block.ExtTransactions())

		*inbounds += len(block.Body().ExternalTransactions())
		latestNumber-- // we could use the parentHash, but BlockByNumber returns the canonical block at the given number
		if latestNumber == 0 {
			break
		}
	}
	wg.Done()
}

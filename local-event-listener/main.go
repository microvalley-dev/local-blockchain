package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// contractABI is the minimal ABI of MyContract that includes:
//   - the function: increment()
//   - the event: CounterIncremented(uint256)
var contractABI = `
[
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": false,
        "internalType": "uint256",
        "name": "newValue",
        "type": "uint256"
      }
    ],
    "name": "CounterIncremented",
    "type": "event"
  },
  {
    "inputs": [],
    "name": "increment",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  }
]
`

func main() {
    // ------------------------------------------------------------------------
    // 1. Connect to local anvil chain (HTTP or WS). We'll use HTTP here.
    //    If you'd prefer WebSocket, anvil also logs a ws endpoint (e.g. ws://127.0.0.1:8545).
    // ------------------------------------------------------------------------
    client, err := ethclient.Dial("ws://127.0.0.1:8545")
    if err != nil {
        log.Fatalf("Failed to connect to local anvil node: %v", err)
    }
    defer client.Close()
    fmt.Println("Successfully connected to local anvil node.")

    // ------------------------------------------------------------------------
    // 2. Parse the contract ABI so we can decode logs
    // ------------------------------------------------------------------------
    parsedABI, err := abi.JSON(strings.NewReader(contractABI))
    if err != nil {
        log.Fatalf("Failed to parse contract ABI: %v", err)
    }

    // ------------------------------------------------------------------------
    // 3. The contract address from 'forge create' step
    //    Replace with the actual address returned from your deployment.
    // ------------------------------------------------------------------------
    contractAddress := common.HexToAddress("0x5fbdb2315678afecb367f032d93f642f64180aa3")

    // ------------------------------------------------------------------------
    // 4. Create a filter query to only listen for the "CounterIncremented" event
    // ------------------------------------------------------------------------
    eventSig := parsedABI.Events["CounterIncremented"].ID
    query := ethereum.FilterQuery{
        Addresses: []common.Address{contractAddress},
        Topics:    [][]common.Hash{{eventSig}},
    }

    // ------------------------------------------------------------------------
    // 5. Subscribe to these logs using the FilterLogs approach.
    //    Since it's a local chain, we can also loop over blocks, but
    //    subscription is simpler in real-time scenarios.
    // ------------------------------------------------------------------------
    logsChan := make(chan types.Log)
    sub, err := client.SubscribeFilterLogs(context.Background(), query, logsChan)
    if err != nil {
        log.Fatalf("Failed to subscribe to filter logs: %v", err)
    }
    fmt.Println("Subscribed to CounterIncremented events. Listening...")

    // ------------------------------------------------------------------------
    // 6. Listen in a loop for new events. Unpack log data to get the newValue.
    // ------------------------------------------------------------------------
    for {
        select {
        case err := <-sub.Err():
            log.Fatalf("Subscription error: %v", err)

        case vLog := <-logsChan:
            fmt.Println("--------------------------------------------")
            fmt.Printf("New event in Tx: %s\n", vLog.TxHash.Hex())

            // Unpack the log data. The event has 1 parameter: newValue (uint256).
            // go-ethereum decodes these as *big.Int.
            unpacked, err := parsedABI.Unpack("CounterIncremented", vLog.Data)
            if err != nil {
                log.Printf("Failed to unpack event data: %v\n", err)
                continue
            }

            newValue := unpacked[0].(*big.Int)
            fmt.Printf("CounterIncremented => newValue = %s\n", newValue.String())

            // If you want to see transaction details, you can do so:
            _, isPending, err := client.TransactionByHash(context.Background(), vLog.TxHash)
            if err != nil {
                log.Printf("Failed to fetch transaction: %v\n", err)
                continue
            }
            // Or check receipt info, etc.

            fmt.Printf("Is transaction pending?: %t\n", isPending)
            fmt.Println("--------------------------------------------")
        }
    }
}

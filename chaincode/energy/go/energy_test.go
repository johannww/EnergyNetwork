/*
Created by Johann Westphall
*/

package main

import (
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shimtest"
)

func fakeMain(energyChaincode *EnergyChaincode) {
	averageFunctionTimes = make(map[string]*FunctionStats)

	initFuncMap(energyChaincode)

	for functionName := range functionMap {
		averageFunctionTimes[functionName] = &FunctionStats{0, 0}
	}

	channelAverageCalculator = make(chan *FunctionAndDuration)
	go recalculateFunctionAverageTime()
}

func TestAuctionCalling(test *testing.T) {
	energyChaincode := &EnergyChaincode{}
	stub := shimtest.NewMockStub("energy", energyChaincode)

	fakeMain(energyChaincode)

	stub.MockInit("tx1", [][]byte{})

	stub.MockInvoke("tx2", [][]byte{[]byte("auction")})
}

/*
Created by Johann Westphall
*/

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"

	"github.com/hyperledger/fabric-chaincode-go/pkg/attrmgr"
	"github.com/hyperledger/fabric-chaincode-go/shim"

	"github.com/hyperledger/fabric-protos-go/msp"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
)

const (
	SENSORS_MSP        = "sensor_org"
	PAYMENTCOMPANY_MSP = "paymentcompany"
	UTILITY_MSP        = "utility"
	CANDELA_UNIT       = "3834792229"
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

func generateX509(attrs attrmgr.Attributes, msp, cn string) []byte {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)
	keyUsage := x509.KeyUsageDigitalSignature
	marshaledAttr, _ := json.Marshal(attrs)
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:       []string{strings.ToUpper(msp)},
			OrganizationalUnit: []string{msp},
			CommonName:         cn,
		},
		ExtraExtensions: []pkix.Extension{{
			Id:       attrmgr.AttrOID,
			Critical: false,
			Value:    marshaledAttr}},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Duration(365 * 24 * time.Hour)),

		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		fmt.Println(err.Error())
	}
	certBytesPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	//ioutil.WriteFile("cert.pem", certBytesPem, 0)
	return certBytesPem
}

func createAdminCertBytes(admName, admMsp string) []byte {
	return generateX509(attrmgr.Attributes{Attrs: map[string]string{
		"admin":        "true",
		"energy.admin": "true",
		"energy.init":  "true",
	}}, admMsp, admName)
}

func createBuyerCertBytes() []byte {
	return generateX509(attrmgr.Attributes{Attrs: map[string]string{
		"ou": "idemixorg"}}, "", "")
}

func createSellerCertBytes(sellerName string) []byte {
	return generateX509(attrmgr.Attributes{Attrs: map[string]string{
		"energy.seller": "true"}}, UTILITY_MSP, sellerName)
}

func createSensorCertBytes(sensorName string) []byte {
	return generateX509(attrmgr.Attributes{Attrs: map[string]string{
		"energy.sensor": "true",
		"energy.x":      "0",
		"energy.y":      "0",
		"energy.z":      "0",
		"energy.radius": "1000",
	}}, SENSORS_MSP, sensorName)
}

func createPaymentCompanyCertBytes() []byte {
	return generateX509(attrmgr.Attributes{Attrs: map[string]string{
		"energy.paymentcompany": "true"}}, PAYMENTCOMPANY_MSP, "")
}

func createUtilityCertBytes() []byte {
	return generateX509(attrmgr.Attributes{Attrs: map[string]string{
		"energy.utility": "true"}}, UTILITY_MSP, "")
}

func mockTxId() string {
	return time.Now().String()
}

func TestAuctionCalling(test *testing.T) {
	energyChaincode := &EnergyChaincode{}
	stub := shimtest.NewMockStub("energy", energyChaincode)

	fakeMain(energyChaincode)

	stub.MockInit("tx1", [][]byte{})

	stub.MockInvoke("tx2", [][]byte{[]byte("auction")})

	insertBuyBids(stub, 1)
}

func insertBuyBids(stub *shimtest.MockStub, quantity int) {
	stub.MockInvoke(stub.TxTimestamp.AsTime().String(), [][]byte{
		[]byte("registerBuyBid"),
		[]byte("payCompany"), []byte("PAYMENT-TOKEN"),
		[]byte("utilityCompany"),
		[]byte("10"), []byte("5"),
		[]byte("solar")})
}

func TestSensorsManipulation(test *testing.T) {
	energyChaincode := &EnergyChaincode{}
	fakeMain(energyChaincode)

	stub := shimtest.NewMockStub("energy", energyChaincode)

	sensorCreator, _ := proto.Marshal(&msp.SerializedIdentity{
		Mspid:   SENSORS_MSP,
		IdBytes: createSensorCertBytes("sensor1"),
	})

	stub.Creator = sensorCreator
	stub.MockInvoke(mockTxId(), [][]byte{[]byte("sensorDeclareActive")})

	adminSensorsCreator, _ := proto.Marshal(&msp.SerializedIdentity{
		Mspid:   SENSORS_MSP,
		IdBytes: createAdminCertBytes("admin1", SENSORS_MSP),
	})

	stub.Creator = adminSensorsCreator
	res := stub.MockInvoke(mockTxId(), [][]byte{[]byte("getActiveSensors")})
	var sensorIds []string
	json.Unmarshal(res.GetPayload(), &sensorIds)

	stub.MockInvoke(mockTxId(), [][]byte{
		[]byte("disableSensors"),
		[]byte(sensorIds[0])})

	res = stub.MockInvoke(mockTxId(), [][]byte{[]byte("getActiveSensors")})
	var emptySensorIds []string
	json.Unmarshal(res.GetPayload(), &emptySensorIds)
	if len(emptySensorIds) > 0 {
		test.Errorf("The active sensors list should be empty")
	}

	res = stub.MockInvoke(mockTxId(), [][]byte{
		[]byte("enableSensors"),
		[]byte(sensorIds[0])})
	json.Unmarshal(res.GetPayload(), &sensorIds)
	if len(sensorIds) < 1 {
		test.Errorf("The active sensors list should NOT be empty")
	}

	adminUtilityCreator, _ := proto.Marshal(&msp.SerializedIdentity{
		Mspid:   UTILITY_MSP,
		IdBytes: createAdminCertBytes("admin1", UTILITY_MSP),
	})

	stub.Creator = adminUtilityCreator
	stub.MockInvoke(mockTxId(), [][]byte{
		[]byte("setTrustedSensors"),
		[]byte(SENSORS_MSP),
		[]byte(sensorIds[0])})

	res = stub.MockInvoke(mockTxId(), [][]byte{
		[]byte("getTrustedSensors")})

	var trustedSensors [][]string
	json.Unmarshal(res.GetPayload(), &trustedSensors)
	if len(trustedSensors) < 1 {
		test.Errorf("The trusted sensors map should NOT be empty")
	}

	stub.MockInvoke(mockTxId(), [][]byte{
		[]byte("setDistrustedSensors"),
		[]byte(SENSORS_MSP),
		[]byte(sensorIds[0])})

	res = stub.MockInvoke(mockTxId(), [][]byte{
		[]byte("getTrustedSensors")})

	var emptyTrustedSensors [][]string
	json.Unmarshal(res.GetPayload(), &emptyTrustedSensors)
	if len(emptyTrustedSensors) > 0 {
		test.Errorf("The trusted sensors map should be empty")
	}
}

func TestEnergyValidationAndSelling(test *testing.T) {
	energyChaincode := &EnergyChaincode{}
	fakeMain(energyChaincode)

	stub := shimtest.NewMockStub("energy", energyChaincode)

	sellerCreator, _ := proto.Marshal(&msp.SerializedIdentity{
		Mspid:   SENSORS_MSP,
		IdBytes: createSellerCertBytes("seller1"),
	})

	test.Log("Declaring a sensor active")
	sensorCreator, _ := proto.Marshal(&msp.SerializedIdentity{
		Mspid:   SENSORS_MSP,
		IdBytes: createSensorCertBytes("sensor1"),
	})

	stub.Creator = sensorCreator
	stub.MockInvoke(mockTxId(), [][]byte{[]byte("sensorDeclareActive")})

	test.Log("Getting sellerID for later registering")
	stub.Creator = sellerCreator
	sellerID, _ := cid.GetID(stub)

	test.Log("Getting sensor active ID to set as trusted")
	adminSensorsCreator, _ := proto.Marshal(&msp.SerializedIdentity{
		Mspid:   SENSORS_MSP,
		IdBytes: createAdminCertBytes("admin1", SENSORS_MSP),
	})

	stub.Creator = adminSensorsCreator
	res := stub.MockInvoke(mockTxId(), [][]byte{[]byte("getActiveSensors")})
	var sensorIds []string
	json.Unmarshal(res.GetPayload(), &sensorIds)

	test.Log("Setting sensor as trusted by the utility")
	adminUtilityCreator, _ := proto.Marshal(&msp.SerializedIdentity{
		Mspid:   UTILITY_MSP,
		IdBytes: createAdminCertBytes("sensor1", UTILITY_MSP),
	})

	stub.Creator = adminUtilityCreator
	stub.MockInvoke(mockTxId(), [][]byte{
		[]byte("setTrustedSensors"),
		[]byte(SENSORS_MSP),
		[]byte(sensorIds[0])})

	test.Log("Registering seller and setting the sensor as its SmartMeter")
	res = stub.MockInvoke(mockTxId(), [][]byte{
		[]byte("registerSeller"),
		[]byte(sellerID),
		[]byte(SENSORS_MSP),
		[]byte(sensorIds[0]),
		[]byte("3"),
		[]byte("3")})

	if res.GetStatus() != shim.OK {
		test.Errorf("Seller registering failed")
	}

	test.Log("Delaying 10 seconds to avoid timestamp errors")
	time.Sleep(time.Duration(10 * time.Second))

	stub.Creator = sensorCreator
	sensorStartTimstamp := uint64(time.Now().Unix())
	test.Logf("publishing %d SmartData since timestamp %d", 10, sensorStartTimstamp)
	for i := 0; i < 10; i++ {
		stub.MockInvoke(mockTxId(), [][]byte{
			[]byte("publishSensorData"),
			[]byte("1"),
			[]byte(CANDELA_UNIT),
			[]byte(strconv.FormatUint(sensorStartTimstamp+uint64(i), 10)),
			[]byte("50"),
			[]byte("0"),
			[]byte("1"),
			[]byte("0")})
	}

	sensorMspID, _ := cid.GetMSPID(stub)
	sensorID, _ := cid.GetID(stub)
	startKey := "SmartData" + sensorMspID + sensorID + getMaxUint64CharsStrTimestamp(sensorStartTimstamp)
	endKey := "SmartData" + sensorMspID + sensorID + getMaxUint64CharsStrTimestamp(sensorStartTimstamp+10)
	stateIterator, _ := stub.GetStateByRange(startKey, endKey)

	i := 0
	for stateIterator.HasNext() {
		i++
		stateIterator.Next()
	}

	if i != 10 {
		test.Errorf("Smart data publication failed. Expected: %d. Got: %d.", 10, i)
	}

	test.Logf("Setting the env CORE_PEER_LOCALMSPID to %s to simulate the peer MSP", UTILITY_MSP)
	os.Setenv("CORE_PEER_LOCALMSPID", UTILITY_MSP)
	test.Log("Publishing a energyGenerationClaim outside the sensors published data interval")
	res = stub.MockInvoke(mockTxId(), [][]byte{
		[]byte("publishEnergyGeneration"),
		[]byte(strconv.FormatUint(sensorStartTimstamp-10, 10)),
		[]byte(strconv.FormatUint(sensorStartTimstamp-5, 10)),
		[]byte("solar"),
		[]byte("15")})

	if res.GetStatus() != shim.ERROR {
		test.Error("This energy generation should have been REJECTED! The generation interval did not matched the SmartData published interval")
	}

	test.Log("Publishing a energyGenerationClaim of wind type wihtout wind SmartData in World State")
	res = stub.MockInvoke(mockTxId(), [][]byte{
		[]byte("publishEnergyGeneration"),
		[]byte(strconv.FormatUint(sensorStartTimstamp+1, 10)),
		[]byte(strconv.FormatUint(sensorStartTimstamp+9, 10)),
		[]byte("wind"),
		[]byte("15")})

	if res.GetStatus() != shim.ERROR {
		test.Error("This energy generation should have been REJECTED! There is no SmartData to back up this energy generation")
	}

	test.Log("Publishing a energyGenerationClaim inside the sensors published data interval")
	res = stub.MockInvoke(mockTxId(), [][]byte{
		[]byte("publishEnergyGeneration"),
		[]byte(strconv.FormatUint(sensorStartTimstamp+1, 10)),
		[]byte(strconv.FormatUint(sensorStartTimstamp+9, 10)),
		[]byte("solar"),
		[]byte("15")})

	if res.GetStatus() != shim.OK {
		test.Error("This energy generation should have been ACCEPTED!")
	}

}

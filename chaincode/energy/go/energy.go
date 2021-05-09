/*
Created by Johann Westphall
*/

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"time"

	st "energy/proto_structs"

	"google.golang.org/protobuf/types/known/anypb"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

// EnergyChaincode example simple Chaincode implementation
type EnergyChaincode struct {
}

// FunctionStats keeps track of functions number of calls and the average
type FunctionStats struct {
	NCalls            uint64  `json:"calls"`
	AvarageExecTimeMs float64 `json:"average_ms"`
}

//averageFunctionTime tracks the average time to call a chaincode function
var averageFunctionTimes map[string]*FunctionStats

// FunctionAndDuration encapsulates the function name and the a single exec duration to be sent from
// the execution threads (or routines) to the average calculator routine
type FunctionAndDuration struct {
	FunctionName string
	Duration     time.Duration
}

// channelAverageCalculator transfers the function duration to a specific goroutine to avoid blocking the chaincode return
var channelAverageCalculator chan *FunctionAndDuration

//functionMap to make function calls faster
var functionMap map[string]func(shim.ChaincodeStubInterface, []string) pb.Response

//accept timestamp delay of XX seconds
var acceptedDelay uint64 = 30

//accepted clock drift between this chaincode and the applications  (XX seconds)
//only applied on publishEnergyGeneration function
var acceptedClockDrift uint64 = 10

//max uint64 str chars
var maxUint64Chars int = len(strconv.FormatUint(^uint64(0), 10))

//mask to identify only the unit exponent fields of a SmartData unit
var smartDataUnitMask uint32 = 0x07FFFFFF

//Getting the last 27 bits identifying a m/s SmartData unit
var smartDataMeterPerSecondUnitPart uint32 = 0xe4963924 & smartDataUnitMask

//Getting the last 27 bits identifying a candela SmartData unit
var smartDataCandelaUnitPart uint32 = 0xe4924925 & smartDataUnitMask

//ActiveSensor represents the registering of a sensor
//ActiveSensor aprox. memory size = 10 + 177 + 1 + 4*3 + 8 = 208 bytes
/*type ActiveSensor struct {
	MspID    string  `json:"mspid"`
	SensorID string  `json:"sensorid"`
	IsActive bool    `json:"active"`
	X        int32   `json:"x"`
	Y        int32   `json:"y"`
	Z        int32   `json:"z"`
	Radius   float64 `json:"radius"`
}

//acceleration smartdata unit
var ACCELERATION = 2224433444

//temperature smartdata unit
var TEMPERATURE = 2224179556

//SmartData struct with data from sensor
//SmartData aprox. memory size = 10 + 177 + 1 + 4 + 8 + 8 + 1 + 1 + 4 + 4 + 4 + 4 = 226 bytes
type SmartData struct {
	AssetID    string  `json:"assetid"`
	Version    int8    `json:"version"`
	Unit       uint32  `json:"unit"`
	Timestamp  uint64  `json:"timestamp"`
	Value      float64 `json:"value"`
	Error      uint8   `json:"e"`
	Confidence uint8   `json:"confidence"`
	//X          int     `json:"x"`
	//Y          int     `json:"y"`
	//Z          int     `json:"z"`
	Dev uint32 `json:"dev"`
}

//SellerInfo stores information regarding the seller in terms of
//energy generated, generation gear and coin balance
//SellerInfo aprox. memory size = 10 + 177 + 10 + 177 + 4 + 4 + (len(EnergyTypes)*(8 + 10)) + 8 + 8 + 4 = 492 bytes
type SellerInfo struct {
	MspIDSeller             string             `json:"mspseller"`
	SellerID                string             `json:"sellerid"`
	MspIDSmartMeter         string             `json:"mspsmartmeter"`
	SmartMeterID            string             `json:"smartmeterid"`
	WindTurbinesNumber      uint64             `json:"windturbinesnumber"`
	SolarPanelsNumber       uint64             `json:"solarpanelsnumber"`
	EnergyToSellByType      map[string]float64 `json:"energytosell"`
	LastGenerationTimestamp uint64             `json:"lastgenerationtimestamp"`
	//CoinBalance             float64            `json:"coinbalance"`
	LastBidID uint64 `json:"lastbid"`
}

// MeterSeller was created so we could abdon CouchDB by adding a link from
// the MspIDSmartMeter+SmartMeterID pointing to the SellerInfo of the seller the
// meter belongs to.
// We found out that LevelDB with StateKey or KeyRange queries are 1000x faster than CouchDB JSON
// queries and 10x faster than CouchDB StateKey or KeyRange queries.
// Our intent was to substitute the query:  queryString := fmt.Sprintf(`{"selector":{"mspsmartmeter":"%s","smartmeterid":"%s"}}`, meterMspID, meterID)
// in function: getSellerInfoRelatedToSmartMeter()
// So we had to create the MeterSeller to gain efficiency on database access
type MeterSeller struct {
	MspIDSeller string `json:"mspseller"`
	SellerID    string `json:"sellerid"`
}

//SellBid stores information regarding the seller wish
//to sell a certain energy type.
//SellBid is used in the auction
//SellBid aprox. memory size = 1 + 10 + 177 + 4 + 8 + 8 + 10 = 218 bytes
type SellBid struct {
	//IsSellBid       bool    `json:"issellbid"`
	MspIDSeller       string  `json:"mspseller"`
	SellerID          string  `json:"sellerid"`
	SellerBidNumber   uint64  `json:"sellerbidnumber"`
	EnergyQuantityKWH float64 `json:"energyquantity"`
	PricePerKWH       float64 `json:"priceperkwh"`
	EnergyType        string  `json:"energytype"`
}
functionStats.nCall./ FunctionStats keeps track of functions number of calls and the avarage varageExecTime//BuyBid is used in the auction
//ByBid aprox. memory size = 10 + len(token) + 8 + 8 + 10 + 1 = 37 + len(token) bytes
type BuyBid struct {
	MspIDPaymentCompany string `json:"msppaymentcompany"`
	Token               string `json:"token"`
	//TxID                string  `json:"txid"`
	UtilityMspID      string  `json:"utilityid"`
	EnergyQuantityKWH float64 `json:"energyquantity"`
	PricePerKWH       float64 `json:"priceperkwh"`
	EnergyType        string  `json:"energytype"`
	Validated         bool    `json:"validated"`
}

//EnergyTransaction is the result of a SellBid matched to a BuyBid after the auction
//EnergyTransaction aprox. memory size = 10 + 177 + 4 + 10 + len(token) + 8 + 8 + 10 = 227 + len(token) bytes
type EnergyTransaction struct {
	MspIDSeller         string `json:"mspseller"`
	SellerID            string `json:"sellerid"`
	SellerBidNumber     uint64 `json:"sellerbidnumber"`
	MspIDPaymentCompany string `json:"msppaymentcompany"`
	Token               string `json:"token"`
	//BuyBidTxID          string  `json:"buybidtxid"`
	BuyerUtilityMspID string  `json:"utilityid"`
	EnergyQuantityKWH float64 `json:"energyquantity"`
	PricePerKWH       float64 `json:"priceperkwh"`
	EnergyType        string  `json:"energytype"`
}

type FullToken struct {
	MspIDPaymentCompany string `json:"msppaymentcompany"`
	Token               string `json:"token"`
}

type SellBidEnergyTransactions struct {
	FullTokens []FullToken `json:"fulltokens"`
}*/

var EnergyTypes []string = []string{"solar", "wind", "tidal", "hydro", "geothermal"}

var EnergyTypeSmartDataUnits = map[string][]uint64{
	"solar":      []uint64{3834792229, 3834792292},             //candela and kelvin
	"wind":       []uint64{3835050276},                         //meters per second
	"tidal":      []uint64{3835050276, 3835574564, 3835054372}, //meters per second, cubic meters per second or water level (meter)
	"hydro":      []uint64{3835050276, 3835574564, 3835054372}, //meters per second, cubic meters per second or water level (meter)
	"geothermal": []uint64{3834792292},                         //kelvin
}

var printEnabled = false

func printf(mainStr string, a ...interface{}) {
	if printEnabled {
		fmt.Printf(mainStr, a...)
	}
}

func println(a ...interface{}) {
	if printEnabled {
		fmt.Println(a...)
	}
}

func (chaincode *EnergyChaincode) setPrint(print bool) pb.Response {
	printEnabled = print
	return shim.Success(nil)
}

func (chaincode *EnergyChaincode) getAverageFunctionTimes() pb.Response {
	averageFunctionTimesMap := make(map[string]FunctionStats)
	for functionName := range functionMap {
		averageFunctionTimesMap[functionName] = *(averageFunctionTimes[functionName])
	}
	jsonFunctionTimes, _ := json.Marshal(averageFunctionTimesMap)
	return shim.Success([]byte(jsonFunctionTimes))
}

func (chaincode *EnergyChaincode) getState(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	println("---- getState function beggining ----")
	key := args[0]
	state, err := stub.GetState(key)

	if err != nil {
		return shim.Error(err.Error())
	}

	println("State: " + string(state))
	return shim.Success(nil)
}

/*
 @Param ChaincodeStubInterface
 - Function called by SENSORS to declare that they are active. It also gathers the sensor coordinates and
 the sensor radius of relevance.
 - Function stores a struct 'ActiveState' to the ledger.
 - The key is formed with the MSP ID and SENSOR ID.
 - MSP ID is unique among all organizations and SENSOR ID is unique within the same MSP.
*/
func (chaincode *EnergyChaincode) sensorDeclareActive(stub shim.ChaincodeStubInterface) pb.Response {
	println("---- sensorDeclareActive function beggining ----")

	//check if caller is a sensor
	err := cid.AssertAttributeValue(stub, "energy.sensor", "true")
	if err != nil {
		return shim.Error(err.Error())
	}

	//get the SensorID and the MSP ID of the sensor to create the composite state key
	sensorID, err := cid.GetID(stub)
	mspID, err := cid.GetMSPID(stub)

	//check if sensor is already in the database
	key, err := stub.CreateCompositeKey("ActiveSensor", []string{mspID, sensorID})
	isActive, err := stub.GetState(key)

	if err != nil {
		return shim.Error(err.Error())
	}

	println("Key: " + key)
	println(fmt.Sprintf("Key (hex): %x", key))
	println("State: " + string(isActive))

	if isActive == nil && err == nil {
		//setting sensor to ACTIVE
		//getting sensor coordinates and the influence radius from the certificate
		xCert, _, _ := cid.GetAttributeValue(stub, "energy.x")
		yCert, _, _ := cid.GetAttributeValue(stub, "energy.y")
		zCert, _, _ := cid.GetAttributeValue(stub, "energy.z")
		radiusCert, _, _ := cid.GetAttributeValue(stub, "energy.radius")

		x, err := strconv.Atoi(xCert)
		y, err := strconv.Atoi(yCert)
		z, err := strconv.Atoi(zCert)
		radius, err := strconv.ParseFloat(radiusCert, 64)

		//putting the info in the struct ActiveSensor
		activityData := st.ActiveSensor{
			MspID:    mspID,
			SensorID: sensorID,
			IsActive: true,
			X:        int32(x),
			Y:        int32(y),
			Z:        int32(z),
			Radius:   radius,
		}

		//the struct will be saved as a Marshalled json
		activityDataBytes, err := proto.Marshal(&activityData)
		err = stub.PutState(key, activityDataBytes)
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success(nil)
	}

	return shim.Error("SENSOR IS DISABLED or ALREADY ACTIVE!")

}

/*
 @Param stub - to interact with the World State
 - Returns the State Key of each active sensor in a JSON formatted string
*/
func (chaincode *EnergyChaincode) getActiveSensors(stub shim.ChaincodeStubInterface) pb.Response {
	println("---- getActiveSensors function beggining ----")

	err := cid.AssertAttributeValue(stub, "energy.admin", "true")
	if err != nil {
		return shim.Error(err.Error())
	}

	mspID, err := cid.GetMSPID(stub)
	activeSensorsIDs, _, err := getActiveSensorsList(stub, mspID)
	if err != nil {
		return shim.Error(err.Error())
	}

	sensorsIDsBytes, err := json.Marshal(activeSensorsIDs)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(sensorsIDsBytes)
}

/*
 @Param stub - to interact with the World State
 @Param mspID to get the active sensors from a specific MSP.
 - Returns the State Key of each active sensor,
 the 'ActiveSensor' struct related to each active sensor
 and an error
 - If the 'mspID' is empty, it returns ALL Active Sensors
*/
func getActiveSensorsList(stub shim.ChaincodeStubInterface, mspID string) ([]string, []st.ActiveSensor, error) {
	println("---- getActiveSensorsList function beggining ----")

	var stateIterator shim.StateQueryIteratorInterface
	var err error

	//getting ActiveSensor iterator from the database
	if len(mspID) > 0 {
		stateIterator, err = stub.GetStateByPartialCompositeKey("ActiveSensor", []string{mspID})
	} else {
		stateIterator, err = stub.GetStateByPartialCompositeKey("ActiveSensor", []string{})
	}
	if err != nil {
		return nil, nil, err
	}

	var activeSensorIDs []string
	var activeSensorsDataList []st.ActiveSensor
	var activityData st.ActiveSensor

	//filter out the inactive sensors
	for stateIterator.HasNext() {
		queryResult, err := stateIterator.Next()
		if err != nil {
			return nil, nil, err
		}
		err = proto.Unmarshal(queryResult.Value, &activityData)
		//println("NameSpace: " + queryResult.Namespace)
		//println("Key: " + queryResult.Key)
		//printf("%+v\n", activityData)
		if activityData.IsActive == true {
			activeSensorIDs = append(activeSensorIDs, activityData.SensorID)
			activeSensorsDataList = append(activeSensorsDataList, activityData)
			printf("%+v\n", activityData)
		} else {
			println("SENSOR NOT ACTIVE!")
		}
	}

	stateIterator.Close()

	//returning the state keys for each sensor, followed a list of ActiveSensor
	return activeSensorIDs, activeSensorsDataList, nil

}

/*
 @Param stub - to interact with the World State
 @Param sensorIDs - list of sensorsIDs to be disabled
 - Only admins from the organization that OWNS the sensor might
 disable a sensor
*/
func (chaincode *EnergyChaincode) disableSensors(stub shim.ChaincodeStubInterface, sensorsIDs []string) pb.Response {
	println("---- disableSensor function beggining ----")

	var activityData st.ActiveSensor

	//check if caller an admin
	err := cid.AssertAttributeValue(stub, "energy.admin", "true")
	if err != nil {
		return shim.Error(err.Error())
	}

	if len(sensorsIDs) < 1 {
		return shim.Error("NO SENSOR ID SPECIFIED!")
	}

	mspID, err := cid.GetMSPID(stub)

	for _, sensorID := range sensorsIDs {

		key, err := stub.CreateCompositeKey("ActiveSensor", []string{mspID, sensorID})

		activityDataBytes, err := stub.GetState(key)
		//abort if sensor is not registered
		if activityDataBytes == nil {
			return shim.Error("SENSOR " + sensorID + " IS NOT REGISTERED!")
		}

		err = proto.Unmarshal(activityDataBytes, &activityData)
		if err != nil {
			return shim.Error("error Unmarshalling ActiveSensor data. Sensor ID: " + sensorID)
		}

		//disable sensor
		activityData.IsActive = false
		activityDataBytes, _ = proto.Marshal(&activityData)
		err = stub.PutState(key, activityDataBytes)
		if err != nil {
			return shim.Error(err.Error())
		}
	}
	return shim.Success(nil)
}

/*
 @Param stub - to interact with the World State
 @Param sensorIDs - list of sensorsIDs to be enabled
 - Only admins from the organization that OWNS the sensor might
 enable a sensor
*/
func (chaincode *EnergyChaincode) enableSensors(stub shim.ChaincodeStubInterface, sensorsIDs []string) pb.Response {
	println("---- enableSensor function beggining ----")

	var activityData st.ActiveSensor

	//check if caller is an admin
	err := cid.AssertAttributeValue(stub, "energy.admin", "true")
	if err != nil {
		return shim.Error(err.Error())
	}

	if len(sensorsIDs) < 1 {
		return shim.Error("NO SENSOR ID SPECIFIED!")
	}

	mspID, err := cid.GetMSPID(stub)

	for _, sensorID := range sensorsIDs {
		//abort if sensor has not declared itself ACTIVE
		key, err := stub.CreateCompositeKey("ActiveSensor", []string{mspID, sensorID})
		activityDataBytes, err := stub.GetState(key)
		if activityDataBytes == nil && err == nil {
			return shim.Error("SENSOR HAS NOT DECLARED ITSELF ACTIVE!")
		}

		err = proto.Unmarshal(activityDataBytes, &activityData)
		if err != nil {
			return shim.Error("error Unmarshalling ActiveSensor data. Sensor ID: " + sensorID)
		}

		//enable sensor
		activityData.IsActive = true
		activityDataBytes, _ = proto.Marshal(&activityData)
		stub.PutState(key, activityDataBytes)
	}
	return shim.Success(nil)
}

/*
 @Param stub - to interact with the World State
 @Param sensorIDs - list of sensorsIDs to be trusted by an organization
 - Only admins can set the trusted sensors for THEIR
 organizations
*/
func (chaincode *EnergyChaincode) setTrustedSensors(stub shim.ChaincodeStubInterface, sensorMspOwnerIDs []string, sensorsIDs []string) pb.Response {
	println("---- setTrustedSensors function beggining ----")

	//check if caller in an admin
	err := cid.AssertAttributeValue(stub, "energy.admin", "true")
	if err != nil {
		return shim.Error(err.Error())
	}

	//check if MSPIDs and sensorsIDs are the same size and greater than 0
	if len(sensorsIDs) != len(sensorMspOwnerIDs) || len(sensorsIDs) < 1 {
		return shim.Error("Please, specify first ALL msp sensor owners IDs before ALL sensors IDs")
	}

	var mspTrustedSensors map[string]bool

	mspID, err := cid.GetMSPID(stub)
	key, _ := stub.CreateCompositeKey("MspTrustedSensors", []string{mspID})
	//get map with the trusted sensors by the caller organization
	mspTrustedSensorsBytes, err := stub.GetState(key)

	//if map exists, unmarhsall it. If not, create the map.
	if mspTrustedSensorsBytes != nil && err != nil {
		err = json.Unmarshal(mspTrustedSensorsBytes, &mspTrustedSensors)
	} else {
		mspTrustedSensors = make(map[string]bool)
	}

	//set the sensors as trusted, using as key the concatenation of sensorMspOwnerIDs[index] and sensorID[index]
	for index, sensorID := range sensorsIDs {
		trusted, alreadyInMap := mspTrustedSensors[sensorID]
		if !alreadyInMap || !trusted {
			mspTrustedSensors[sensorMspOwnerIDs[index]+sensorID] = true
		}
	}

	mspTrustedSensorsBytes, _ = json.Marshal(mspTrustedSensors)
	err = stub.PutState(key, mspTrustedSensorsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)

}

/*
 @Param stub - to interact with the World State
 @Param sensorIDs - list of sensorsIDs to be distrusted by an organization
 - Only admins can set the distrusted sensors for THEIR
 organizations
 - "Distrust" MEANS only that the sensor might not be used for the energy validation
 process
 - It DOES NOT MEAN that the organization consider the sensor a malicious attacker
*/
func (chaincode *EnergyChaincode) setDistrustedSensors(stub shim.ChaincodeStubInterface, sensorMspOwnerIDs []string, sensorsIDs []string) pb.Response {
	println("---- setDistrustedSensors function beggining ----")

	//check if caller in an admin
	err := cid.AssertAttributeValue(stub, "energy.admin", "true")
	if err != nil {
		return shim.Error(err.Error())
	}

	//check if MSPIDs and sensorsIDs are the same size and greater than 0
	if len(sensorsIDs) != len(sensorMspOwnerIDs) || len(sensorsIDs) < 1 {
		return shim.Error("Please, specify first ALL msp sensor owners IDs before ALL sensors IDs")
	}

	var mspTrustedSensors map[string]bool

	mspID, err := cid.GetMSPID(stub)
	key, _ := stub.CreateCompositeKey("MspTrustedSensors", []string{mspID})
	//get map with the trusted sensors by the caller organization

	mspTrustedSensorsBytes, err := stub.GetState(key)
	//if map exists, unmarhsall it. If not, create the map.
	if mspTrustedSensorsBytes != nil && err != nil {
		err = json.Unmarshal(mspTrustedSensorsBytes, &mspTrustedSensors)
	} else {
		mspTrustedSensors = make(map[string]bool)
	}

	//set the sensors as trusted, using as key the concatenation of sensorMspOwnerIDs[index] and sensorID[index]
	for index, sensorID := range sensorsIDs {
		trusted, alreadyInMap := mspTrustedSensors[sensorID]
		if !alreadyInMap || trusted {
			mspTrustedSensors[sensorMspOwnerIDs[index]+sensorID] = false
		}
	}

	mspTrustedSensorsBytes, _ = json.Marshal(mspTrustedSensors)
	stub.PutState(key, mspTrustedSensorsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}

/*
 @Param stub - to interact with the World State
 @Return a JSON formatted string with the sensorsIDs trusted
 by the organization related to the admin calling this function
 - Only called by ADMINS
*/
func (chaincode *EnergyChaincode) getTrustedSensors(stub shim.ChaincodeStubInterface) pb.Response {

	println("---- getTrustedSensors function beggining ----")

	//only organization admins can call this function
	err := cid.AssertAttributeValue(stub, "energy.admin", "true")
	if err != nil {
		return shim.Error(err.Error())
	}

	mspID, err := cid.GetMSPID(stub)

	_, mspTrustedSensorsBytes, err := getMspTrustedSensorsMap(stub, mspID)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(mspTrustedSensorsBytes)
}

func getMspTrustedSensorsMap(stub shim.ChaincodeStubInterface, mspID string) (map[string]bool, []byte, error) {
	println("---- getMspTrustedSensorsMap function beggining ----")
	var mspTrustedSensors map[string]bool

	key, _ := stub.CreateCompositeKey("MspTrustedSensors", []string{mspID})

	mspTrustedSensorsBytes, err := stub.GetState(key)
	if err != nil {
		return nil, nil, err
	} else if mspTrustedSensorsBytes == nil {
		return nil, nil, nil
	}

	_ = json.Unmarshal(mspTrustedSensorsBytes, &mspTrustedSensors)

	for sensorID, trusted := range mspTrustedSensors {
		if trusted {
			println("Sensor: " + sensorID + "\nTrusted: true")
		} else {
			delete(mspTrustedSensors, sensorID)
		}
	}

	return mspTrustedSensors, mspTrustedSensorsBytes, nil
}

func getAllMspsTrustedSensorsMaps(stub shim.ChaincodeStubInterface) (map[string]map[string]bool, error) {
	println("---- getAllMspsTrustedSensorsMaps function beggining ----")

	mspTrustedSensorsMapIterator, _ := stub.GetStateByPartialCompositeKey("MspTrustedSensors", []string{})

	var mspTrustedSensors map[string]bool
	allMspTrustedSensorsMaps := make(map[string]map[string]bool)

	for mspTrustedSensorsMapIterator.HasNext() {
		queryResult, err := mspTrustedSensorsMapIterator.Next()
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(queryResult.Value, &mspTrustedSensors)
		if err != nil {
			return nil, err
		}
		mspID := queryResult.Key[len("MspTrustedSensors")+2 : len(queryResult.Key)-1]
		printf("mspID: %s\n", mspID)
		printf("mspTrustedSensors: %+v\n", mspTrustedSensors)
		allMspTrustedSensorsMaps[mspID] = mspTrustedSensors
	}

	return allMspTrustedSensorsMaps, nil

}

/*
 @Param stub - to interact with the World State
 @Param version - the version of the SmartData in the series
 @Param unit - the type of the SmartData in the series (see the SmartData documentation and typical units)
 @Param x, y, z - the absolute coordinates of the center of the sphere containing the data points in the series (from a SmartData Interest)
 @Param timestamp - the time instant at which the data originated (in UNIX epoch microseconds)
 @Param e - a measure of uncertainty, usually transducer-dependent, expressing Accuracy, Precision, Resolution, or a combination thereof
 @Param confidence - the value of the SmartData matching the query is replaced by each SmartData confidence
 @Param dev - a disambiguation identifier for multiple transducers of the same Unit at the same space-time coordinates
 Source: https://epos.lisha.ufsc.br/IoT+Platform#SmartData
 and: https://epos.lisha.ufsc.br/EPOS+2+User+Guide#Persistent_Storage
 - Gathers the Params in a 'SmartData' struct and stores it in the world state
 - The State key is formed with the mspID, the sensorID and the data timestamp
 - The data is only stored IF there is less than 'acceptedDelay' difference between
 the current time and the data timestamp.
*/
func (chaincode *EnergyChaincode) publishSensorData(stub shim.ChaincodeStubInterface, version int8, unit uint32, timestamp uint64, value float64, e uint8, confidence uint8, dev uint32) pb.Response {
	println("---- publishSensorData function beggining ----")

	//verify if data is still valid based on timestamp
	currentTime := uint64(time.Now().Unix())
	if currentTime > timestamp+acceptedDelay {
		return shim.Error("The data timestamp is too old!")
	}

	//check if caller is a sensor
	err := cid.AssertAttributeValue(stub, "energy.sensor", "true")
	if err != nil {
		return shim.Error(err.Error())
	}

	var activityData st.ActiveSensor

	//get sensor id from the certificate
	sensorID, err := cid.GetID(stub)
	mspID, err := cid.GetMSPID(stub)
	assetID := mspID + sensorID

	//verify if sensor is active
	key, err := stub.CreateCompositeKey("ActiveSensor", []string{mspID, sensorID})
	activityDataBytes, err := stub.GetState(key)
	err = proto.Unmarshal(activityDataBytes, &activityData)
	if err != nil || activityData.IsActive != true {
		return shim.Error("SENSOR IS NOT ACTIVE!")
	}

	//xCert, _, _ := cid.GetAttributeValue(stub, "energy.x")
	//yCert, _, _ := cid.GetAttributeValue(stub, "energy.y")
	//zCert, _, _ := cid.GetAttributeValue(stub, "energy.z")

	//x, err := strconv.Atoi(xCert)
	//y, err := strconv.Atoi(yCert)
	//z, err := strconv.Atoi(zCert)

	asset := st.SmartData{
		AssetID:    assetID,
		Version:    int32(version), //because of proto
		Unit:       unit,
		Timestamp:  timestamp,
		Value:      value,
		Error:      uint32(e),          //because of proto
		Confidence: uint32(confidence), //because of proto
		//X:          x,
		//Y:          y,
		//Z:          z,
		Dev: dev,
	}

	assetJSON, err := proto.Marshal(&asset)
	//key, err = stub.CreateCompositeKey(objectType, []string{mspID, sensorID, strconv.FormatUint(timestamp, 10)})
	//we do not use CompositeKey, because CompositeKeys are not supported for the method shim.ChaincodeStubInterface.GetStateByRange()
	key = "SmartData" + assetID + getMaxUint64CharsStrTimestamp(timestamp)
	stub.PutState(key, assetJSON)
	return shim.Success(nil)
}

func getMaxUint64CharsStrTimestamp(timestamp uint64) string {
	timestampStr := strconv.FormatUint(timestamp, 10)
	for i := len(timestampStr); i < maxUint64Chars; i++ {
		timestampStr = "0" + timestampStr
	}
	return timestampStr
}

func (chaincode *EnergyChaincode) getSensorsPublishedData(stub shim.ChaincodeStubInterface, sensorsIDs []string) pb.Response {
	//CreateAsset(ctx contractapi.TransactionContextInterface,
	//	id string, version int8, unit int, timestamp int, value float64, e int8, confidence int8, x int, y int, z int, dev int8) error

	println("---- getSensorPublishedData function beggining ----")

	err := cid.AssertAttributeValue(stub, "energy.admin", "true")
	if err != nil {
		return shim.Error(err.Error())
	}

	for _, sensorID := range sensorsIDs {
		mspID, err := cid.GetMSPID(stub)
		//stateIterator, err := stub.GetStateByPartialCompositeKey("SmartData", []string{mspID, sensorID})
		//SmartData key must be SIMPLE and not COMPOSITE to enable calling the method shim.ChaincodeStubInterface.GetStateByRange()
		stateIterator, err := stub.GetStateByRange("SmartData"+mspID+sensorID, "")
		if err != nil {
			return shim.Error(err.Error())
		}

		println(stateIterator.HasNext())

		for stateIterator.HasNext() {
			queryResult, err := stateIterator.Next()
			if err != nil {
				return shim.Error(err.Error())
			}
			println("NameSpace: " + queryResult.Namespace)
			println("Key: " + queryResult.Key)
			var asset st.SmartData
			err = proto.Unmarshal(queryResult.Value, &asset)
			if err != nil {
				return shim.Error(err.Error())
			}
			printf("%+v\n", asset)
		}

		stateIterator.Close()
	}
	return shim.Success(nil)
}

/*
 @Param stub - to interact with the World State
 @Param sellerID - the same as returned by the function cid.GetID() of the seller certificate
 @Param mspIDSmartMeter - the MSP ID of the Smart Meter related to the registered seller
 @Param smartMeterID - the same as returned by the function cid.GetID() of the Smart Meter certificate
 @Param windTurbinesNumber - the number of wind turbines owned by the registered seller
 @Param solarPanelsNumber - the number of solar panels owned by the registered seller

 - Function gathers the parameters and create a SellerInfo for the registered seller in the World State
 - Admins can only register seller for their MSP
*/
func (chaincode *EnergyChaincode) registerSeller(stub shim.ChaincodeStubInterface, sellerID string, mspIDSmartMeter string, smartMeterID string, windTurbinesNumber uint64, solarPanelsNumber uint64) pb.Response {
	println("---- registerSeller function beggining ----")

	//only admins can register sellers
	err := cid.AssertAttributeValue(stub, "energy.admin", "true")
	if err != nil {
		return shim.Error(err.Error())
	}

	//test if timestamp is recent enough comparing with the transaction creation timestamp
	timestampStruct, err := stub.GetTxTimestamp()
	if err != nil {
		return shim.Error(err.Error())
	}
	if uint64(time.Now().Unix()-timestampStruct.Seconds) > acceptedDelay {
		return shim.Error("current timestamp provided is too old!")
	}

	//the seller msp is the same of the admin registering them
	mspIDSeller, err := cid.GetMSPID(stub)

	//generate compositekey
	key, err := stub.CreateCompositeKey("SellerInfo", []string{mspIDSeller, sellerID})
	sellerInfoBytes, err := stub.GetState(key)

	//check if seller is not already registered
	if sellerInfoBytes != nil {
		return shim.Error("Seller is ALREADY registered!")
	}

	//check if smartmeter is not already registered
	_, err = getSellerInfoRelatedToSmartMeter(stub, mspIDSmartMeter, smartMeterID) //REACTIVATE THIS LINE
	if err == nil {
		return shim.Error("Smart Meter is ALREADY registered!")
	}

	energyToSellByType := make(map[string]float64)
	energyToSellByType["solar"] = 0.0

	sellerInfo := st.SellerInfo{
		MspIDSeller:             mspIDSeller,
		SellerID:                sellerID,
		MspIDSmartMeter:         mspIDSmartMeter,
		SmartMeterID:            smartMeterID,
		WindTurbinesNumber:      windTurbinesNumber,
		SolarPanelsNumber:       solarPanelsNumber,
		EnergyToSellByType:      energyToSellByType,
		LastGenerationTimestamp: uint64(timestampStruct.Seconds), //REACTIVATE THIS LINE!!!
		//LastGenerationTimestamp: 0, //DELETE THIS LINE!!!
		//CoinBalance:             0,
		LastBidID: 0,
	}

	//save seller info to the World State
	sellerInfoBytes, err = proto.Marshal(&sellerInfo)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutState(key, sellerInfoBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	printf("SellerInfo: %+v\n", sellerInfo)

	// storing the relation SmartMeter --> Seller Info
	meterToSellerPointer := st.MeterSeller{
		MspIDSeller: mspIDSeller,
		SellerID:    sellerID,
	}

	key, err = stub.CreateCompositeKey("MeterSeller", []string{mspIDSmartMeter, smartMeterID})

	meterToSellerPointerBytes, err := proto.Marshal(&meterToSellerPointer)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutState(key, meterToSellerPointerBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

/*
 @Param stub - to interact with the World State
 @Param sellerInfo - the SellerInfo to be updated in the world state
 @Return error
*/
func updateSellerInfo(stub shim.ChaincodeStubInterface, sellerInfo st.SellerInfo) error {
	println("---- updateSellerInfo function beggining ----")

	key, err := stub.CreateCompositeKey("SellerInfo", []string{sellerInfo.MspIDSeller, sellerInfo.SellerID})

	sellerInfoBytes, err := proto.Marshal(&sellerInfo)
	if err != nil {
		return err
	}
	err = stub.PutState(key, sellerInfoBytes)
	if err != nil {
		return err
	}
	return nil
}

/*
 @Param stub - to interact with the World State
 @Param meterMspID - MSP ID of the smart meter
 @Param meterID - the same as returned by the function cid.GetID() of the Smart Meter certificate
 @Return SellerInfor related to the Smart Meter of MSP 'meterMspID' and ID of 'meterID'
*/
func getSellerInfoRelatedToSmartMeter(stub shim.ChaincodeStubInterface, meterMspID string, meterID string) (st.SellerInfo, error) {
	println("---- getSellerInfoRelatedToSmartMeter function beggining ----")

	var sellerInfo st.SellerInfo
	var meterSeller st.MeterSeller
	var sellerInfoBytes []byte

	/*queryString := fmt.Sprintf(`{"selector":{"mspsmartmeter":"%s","smartmeterid":"%s"}}`, meterMspID, meterID)
	//queryIterator, err := stub.GetQueryResult(queryString)

	if err != nil {
		return sellerInfo, err
	}

	if queryIterator.HasNext() {
		queryResult, _ := queryIterator.Next()
		sellerInfoBytes = queryResult.Value
	} else {
		queryIterator.Close()
		return sellerInfo, fmt.Errorf("No seller related to the meter of MSP %s and of Smart Meter ID %s", meterMspID, meterID)
	}

	queryIterator.Close()*/

	key, err := stub.CreateCompositeKey("MeterSeller", []string{meterMspID, meterID})
	meterSellerBytes, err := stub.GetState(key)
	if err != nil {
		return sellerInfo, err
	}
	err = proto.Unmarshal(meterSellerBytes, &meterSeller)

	key, err = stub.CreateCompositeKey("SellerInfo", []string{meterSeller.MspIDSeller, meterSeller.SellerID})
	sellerInfoBytes, err = stub.GetState(key)
	if err != nil {
		return sellerInfo, err
	}

	err = proto.Unmarshal(sellerInfoBytes, &sellerInfo)

	if err != nil {
		return sellerInfo, err
	}

	return sellerInfo, nil
}

//document later
func (chaincode *EnergyChaincode) publishEnergyGeneration(stub shim.ChaincodeStubInterface, t0 uint64, t1 uint64, energyByTypeGeneratedKWH map[string]float64) pb.Response {
	println("---- publishEnergyGeneration function beggining ----")

	//check if information comes form a meter
	//err := cid.AssertAttributeValue(stub, "energy.meter", "true")
	err := cid.AssertAttributeValue(stub, "energy.sensor", "true")
	if err != nil {
		return shim.Error(err.Error())
	}

	//test if t0 is greater than t1
	if t0 >= t1 {
		return shim.Error("t1 MUST be greater than t0")
	}

	//test if t1 is greater or equal NOW
	currentTimestamp := uint64(time.Now().Unix())
	if t1 > currentTimestamp+acceptedClockDrift {
		return shim.Error(fmt.Sprintf("t1: %d MUST be less or equal than the CURRENT TIME %d + the accepted clock drift of %ds", t1, currentTimestamp, acceptedClockDrift))
	}

	//get Meter MSP and MeterID
	meterID, err := cid.GetID(stub)
	meterMspID, err := cid.GetMSPID(stub)

	println("Meter MSP ID: " + meterMspID)
	println("Meter ID: " + meterID)

	//get SellerInfo related to the smart meter
	sellerInfo, err := getSellerInfoRelatedToSmartMeter(stub, meterMspID, meterID) //REACTIVATE THIS LINE

	if err != nil {
		return shim.Error(err.Error())
	}
	printf("sellerInfo: %+v\n", sellerInfo)

	//test seller is not trying to generate energy twice for the same time interval
	if t0 < sellerInfo.LastGenerationTimestamp || t1 < sellerInfo.LastGenerationTimestamp {
		return shim.Error("Seller already registered energy generation for this time interval")
	}

	//test if energy generated is positive
	for _, energyGeneratedKWH := range energyByTypeGeneratedKWH {
		if energyGeneratedKWH <= 0 {
			return shim.Error("The energy generated MUST be greater than 0")
		}
	}

	//get Meter Location
	xCert, _, _ := cid.GetAttributeValue(stub, "energy.x")
	yCert, _, _ := cid.GetAttributeValue(stub, "energy.y")
	zCert, _, _ := cid.GetAttributeValue(stub, "energy.z")

	x, err := strconv.Atoi(xCert)
	y, err := strconv.Atoi(yCert)
	z, err := strconv.Atoi(zCert)
	println(x)
	println(y)
	println(z)

	//get Active Sensors
	_, activeSensorsDataList, err := getActiveSensorsList(stub, "")

	printf("Active Sensors Data List %+v\n", activeSensorsDataList)
	//definir criterios de aceitacao. EX: 3 organizacoes precisam ter sensores a distancia X
	//usar X, Y e Z para calcular a distancia
	var nearActiveSensorsList []st.ActiveSensor

	for _, activeSensorData := range activeSensorsDataList {
		distanceBetweenSensorAndGenerator := math.Sqrt(math.Pow(float64(activeSensorData.X-int32(x)), 2) + math.Pow(float64(activeSensorData.Y-int32(y)), 2))
		printf("distanceBetweenSensorAndGenerator: %f\n", distanceBetweenSensorAndGenerator)
		printf("activeSensorData.Radius: %f\n", activeSensorData.Radius)
		if distanceBetweenSensorAndGenerator <= activeSensorData.Radius {
			nearActiveSensorsList = append(nearActiveSensorsList, activeSensorData)
		}
	}

	printf("nearActiveSensorsList: %+v\n", nearActiveSensorsList)
	//chamar alguma funcao que puxe do banco de dados os criterios de cada validador (EXECUTAR DE ACORDO COM A ORGANIZACAO A QUAL O PEER EXECUTANDO PERTENCE)
	peerOrg, err := GetMSPID()
	println("Peer executing MSP: " + peerOrg)

	//mspTrustedSensors, _, err := getMspTrustedSensorsMap(stub, peerOrg)
	allMspsTrustedSensorsMaps, _ := getAllMspsTrustedSensorsMaps(stub)
	mspTrustedSensors := allMspsTrustedSensorsMaps[peerOrg]

	printf("mspTrustedSensors: %+v\n", mspTrustedSensors)

	var nearTrustedActiveSensors []st.ActiveSensor

	for _, nearActiveSensor := range nearActiveSensorsList {
		printf("nearActiveSensor: %+v\n Trusted: %t\n", nearActiveSensor, mspTrustedSensors[nearActiveSensor.MspID+nearActiveSensor.SensorID])
		if mspTrustedSensors[nearActiveSensor.MspID+nearActiveSensor.SensorID] {
			nearTrustedActiveSensors = append(nearTrustedActiveSensors, nearActiveSensor)
		}
	}

	printf("nearTrustedActiveSensors: %+v\n", nearTrustedActiveSensors)

	//timeTestFunction
	//t.measureSpeedSmartDataQuery(stub, 100, &nearTrustedActiveSensors, t0, t1)

	//get nearTrustedActiveSensors smart data in time interval [t0,t1]
	var nearTrustedSensorsSmartData []st.SmartData
	nearTrustedSensorsSmartData, err = getSmartDataBySensorsInInterval(stub, &nearTrustedActiveSensors, t0, t1)
	if err != nil {
		return shim.Error(err.Error())
	}

	//check if energy could have been generated considering the nearTrustedActiveSensors published data
	// JUST TEMPORARY! CHECK POLICIES!
	var maxPossibleGeneratedEnergy float64
	for energyType, energyGeneratedKWH := range energyByTypeGeneratedKWH {

		switch energyType {
		case "solar":
			maxPossibleGeneratedEnergy = getMaxPossibleGeneratedSolarEnergyInInterval(stub, &nearTrustedSensorsSmartData, sellerInfo.SolarPanelsNumber, t0, t1)
		case "wind":
			maxPossibleGeneratedEnergy = getMaxPossibleGeneratedWindEnergyInInterval(stub, &nearTrustedSensorsSmartData, sellerInfo.WindTurbinesNumber, t0, t1)
		case "tidal":
			return shim.Error("Not implemented YET!")
		case "hydro":
			return shim.Error("Not implemented YET!")
		case "geothermal":
			return shim.Error("Not implemented YET!")
		default:
			return shim.Error(energyType + " is an INVALID energy type!")

		}

		if energyGeneratedKWH > maxPossibleGeneratedEnergy {
			return shim.Error("Network ruled the alleged generation of " + energyType + " energy INVALID!")
		}
		sellerInfo.EnergyToSellByType[energyType] += energyGeneratedKWH

	}

	sellerInfo.LastGenerationTimestamp = uint64(t1)
	err = updateSellerInfo(stub, sellerInfo)
	if err != nil {
		return shim.Error(err.Error())
	}

	printf("Energy available for selling by type: %+v\n", energyByTypeGeneratedKWH)

	successMessage := fmt.Sprintf("%+v\n successfully available for selling by the seller %s%s with the smart meter of ID: %s%s", energyByTypeGeneratedKWH, sellerInfo.MspIDSeller, sellerInfo.SellerID, sellerInfo.MspIDSmartMeter, sellerInfo.SmartMeterID)

	return shim.SuccessWithPriorityBypassPhantomReadCheck([]byte(successMessage), pb.Priority_MEDIUM)
	//return shim.Error("Testing time")
}

func getMaxPossibleGeneratedWindEnergyInInterval(stub shim.ChaincodeStubInterface, nearTrustedSensorsSmartData *[]st.SmartData, windTurbinesNumber uint64, t0 uint64, t1 uint64) float64 {
	println("---- getMaxPossibleGeneratedWindEnergyInInterval function beggining ----")

	sensorSmartDataQuantity := make(map[string]int)
	sensorSmartDataSum := make(map[string]float64)
	sensorSmartDataMean := make(map[string]float64)

	//possibleUnits := EnergyTypeSmartDataUnits["wind"]
	//calculating each sensor smartData.Value mean in the time interval [t0, t1]
	for _, smartData := range *nearTrustedSensorsSmartData {
		//to understand the variables below read https://epos.lisha.ufsc.br/EPOS+2+User+Guide#Unit
		si := smartData.Unit >> 31
		num := smartData.Unit >> 29 & 3
		mod := smartData.Unit >> 27 & 3
		//we assume that each sensor send data with a uniform periodicity
		if si == 1 {
			//printf("MOD field: %d\n", mod)
			if mod == 0 {
				isMetersPerSecondUnit := (smartData.Unit & smartDataUnitMask) == smartDataMeterPerSecondUnitPart
				if isMetersPerSecondUnit {
					//treat the NUM FIELD
					sensorSmartDataQuantity[smartData.AssetID]++
					if num < 2 {
						//consider float64 bytes as int
						sensorSmartDataSum[smartData.AssetID] += float64(math.Float64bits(smartData.Value))
					} else {
						sensorSmartDataSum[smartData.AssetID] += smartData.Value
					}
				} else {
					printf("SmartData of AssetID: %s%d is not a meter/second unit\n", smartData.AssetID, smartData.Timestamp)
				}
			} else {
				printf("SmartData of AssetID: %s%d is not DIRECTLY DESCRIBED\n", smartData.AssetID, smartData.Timestamp)
			}
		} else {
			printf("SmartData of AssetID: %s%d is not a SI unit\n", smartData.AssetID, smartData.Timestamp)
		}
	}

	for sensorID, sum := range sensorSmartDataSum {
		sensorSmartDataMean[sensorID] = sum / float64(sensorSmartDataQuantity[sensorID])
	}

	windSpeedMean := 0.0
	nSensors := 0.0

	for _, sensorWindSpeedMean := range sensorSmartDataMean {
		windSpeedMean = (windSpeedMean*nSensors + sensorWindSpeedMean) / (nSensors + 1)
		nSensors++
	}

	//hypothetic energy generation function of energy generation based on luminosity around it
	return windSpeedMean * float64(windTurbinesNumber) * 10000000
}

func getMaxPossibleGeneratedSolarEnergyInInterval(stub shim.ChaincodeStubInterface, nearTrustedSensorsSmartData *[]st.SmartData, solarPanelsNumber uint64, t0 uint64, t1 uint64) float64 {
	println("---- getMaxPossibleGeneratedSolarEnergyInInterval function beggining ----")

	sensorSmartDataQuantity := make(map[string]int)
	sensorSmartDataSum := make(map[string]float64)
	sensorSmartDataMean := make(map[string]float64)

	//calculating each sensor smartData.Value mean in the time interval [t0, t1]
	for _, smartData := range *nearTrustedSensorsSmartData {
		//to understand the variables below read https://epos.lisha.ufsc.br/EPOS+2+User+Guide#Unit
		si := smartData.Unit >> 31
		num := smartData.Unit >> 29 & 3
		mod := smartData.Unit >> 27 & 3
		//we assume that each sensor send data with a uniform periodicity
		if si == 1 {
			//printf("MOD field: %d\n", mod)
			if mod == 0 {
				isCandelaUnit := (smartData.Unit & smartDataUnitMask) == smartDataCandelaUnitPart
				if isCandelaUnit {
					//treat the MOD FIELD
					//treat the NUM FIELD
					sensorSmartDataQuantity[smartData.AssetID]++
					if num < 2 {
						//consider float64 bytes as int
						sensorSmartDataSum[smartData.AssetID] += float64(math.Float64bits(smartData.Value))
					} else {
						sensorSmartDataSum[smartData.AssetID] += smartData.Value
					}
				} else {
					printf("SmartData of AssetID: %s%d is not a CANDELA unit", smartData.AssetID, smartData.Timestamp)
				}
			} else {
				printf("SmartData of AssetID: %s%d is not DIRECTLY DESCRIBED", smartData.AssetID, smartData.Timestamp)
			}
		} else {
			printf("SmartData of AssetID: %s%d is not a SI unit", smartData.AssetID, smartData.Timestamp)
		}
	}

	for sensorID, sum := range sensorSmartDataSum {
		sensorSmartDataMean[sensorID] = sum / float64(sensorSmartDataQuantity[sensorID])
	}

	luminosityMean := 0.0
	nSensors := 0.0

	for _, sensorLuminosityMean := range sensorSmartDataMean {
		luminosityMean = (luminosityMean*nSensors + sensorLuminosityMean) / (nSensors + 1)
		nSensors++
	}

	//hypothetic energy generation function of energy generation based on luminosity around it
	return luminosityMean * float64(solarPanelsNumber) * 10000000
}

func getSmartDataBySensorsInInterval(stub shim.ChaincodeStubInterface, nearTrustedActiveSensors *[]st.ActiveSensor, t0 uint64, t1 uint64) ([]st.SmartData, error) {
	println("---- getSmartDataBySensorsInInterval function beggining ----")
	//var assetsIDs string
	var nearTrustedSensorsSmartData []st.SmartData
	var smartDataAux st.SmartData

	/*tStart := time.Now()
	assetsIDs = "["
	for _, nearTrustedActiveSensor := range *nearTrustedActiveSensors {
		assetsIDs += `"` + nearTrustedActiveSensor.MspID + nearTrustedActiveSensor.SensorID + `",`
	}
	assetsIDs = assetsIDs[:len(assetsIDs)-1] + "]"
	queryString := fmt.Sprintf(`{"selector":{"timestamp":{"$gt": %d},"timestamp":{"$lt": %d},"assetid":{ "$in": %s }}}`, t0, t1, assetsIDs)
	println("Query string: " + queryString)
	queryIterator, err := stub.GetQueryResult(queryString)

	if err != nil {
		return nil, err
	}
	duration := time.Since(tStart)

	println("JSON QUERY TIME: " + duration.String())*/

	//TESTING A MORE EFFICIENT QUERY
	//tStart := time.Now()

	var queryIterators []shim.StateQueryIteratorInterface

	for _, nearTrustedActiveSensor := range *nearTrustedActiveSensors {
		startKey := "SmartData" + nearTrustedActiveSensor.MspID + nearTrustedActiveSensor.SensorID + getMaxUint64CharsStrTimestamp(t0)
		println("startKey: " + startKey)
		endKey := "SmartData" + nearTrustedActiveSensor.MspID + nearTrustedActiveSensor.SensorID + getMaxUint64CharsStrTimestamp(t1)
		println("endKey: " + endKey)
		queryIterator, err := stub.GetStateByRange(startKey, endKey)
		queryIterators = append(queryIterators, queryIterator)
		if err != nil {
			return nil, err
		}
	}
	// TEST END
	//duration := time.Since(tStart)
	//println("StateByRange QUERY TIME: " + duration.String())

	numberOfSmartDataFetched := 0
	for _, queryIterator := range queryIterators {
		for queryIterator.HasNext() {
			queryResult, _ := queryIterator.Next()
			err := proto.Unmarshal(queryResult.Value, &smartDataAux)
			if err != nil {
				return nil, err
			}
			nearTrustedSensorsSmartData = append(nearTrustedSensorsSmartData, smartDataAux)
			//printf("%+v\n", smartDataAux)
			numberOfSmartDataFetched++
		}
		queryIterator.Close()
	}
	printf("numberOfSmartDataFetched: %d\n", numberOfSmartDataFetched)
	return nearTrustedSensorsSmartData, nil

}

//func getSmartDataMean()

//GetMSPID returns the MSPID of the peer running this contract
func GetMSPID() (string, error) {
	mspid := os.Getenv("CORE_PEER_LOCALMSPID")

	if mspid == "" {
		return "", errors.New("'CORE_PEER_LOCALMSPID' is not set")
	}

	return mspid, nil
}

/*
 @Param stub - to interact with the World State
 @Param amountKWH - amount of energy to be put in the SellBid (in Kilowatt-hour)
 @Param pricePerKWH - desired price per Kilowatt-hour
 @Param energyType - the source of the energy (e.g. solar, wind)
*/
func (chaincode *EnergyChaincode) registerSellBid(stub shim.ChaincodeStubInterface, quantityKWH float64, pricePerKWH float64, energyType string) pb.Response {
	println("---- registerSellBid function beggining ----")

	var sellerInfo st.SellerInfo

	//only sellers can execute this function
	err := cid.AssertAttributeValue(stub, "energy.seller", "true")
	if err != nil {
		return shim.Error(err.Error())
	}

	//get seller MSPID and their ID
	mspIDSeller, err := cid.GetMSPID(stub)
	sellerID, err := cid.GetID(stub)

	//get SellerInfo from the seller
	keySellerInfo, err := stub.CreateCompositeKey("SellerInfo", []string{mspIDSeller, sellerID})
	sellerInfoBytes, err := stub.GetState(keySellerInfo)

	//check if seller is registered
	if sellerInfoBytes == nil || err != nil {
		return shim.Error("Seller COULD NOT be fetched from the ledger!")
	}

	err = proto.Unmarshal(sellerInfoBytes, &sellerInfo)
	if err != nil {
		return shim.Error(err.Error())
	}

	//check if seller has already produced the amount of energy type
	printf("sellerInfo.EnergyToSellByType['%s']: %f\n", energyType, sellerInfo.EnergyToSellByType[energyType])
	printf("amountKWH: %f\n", quantityKWH)
	if sellerInfo.EnergyToSellByType[energyType] < quantityKWH {
		return shim.Error("Seller does not have the indicated amount of " + energyType + " energy to sell!")
	}

	//subtract energy to be sold from SellerInfo
	sellerInfo.EnergyToSellByType[energyType] -= quantityKWH
	//update lastbid
	sellerInfo.LastBidID++
	//store updated SellerInfo
	err = updateSellerInfo(stub, sellerInfo)
	if err != nil {
		return shim.Error(err.Error())
	}

	sellBid := st.SellBid{
		//IsSellBid:       true, //delete later
		MspIDSeller:       mspIDSeller,
		SellerID:          sellerID,
		SellerBidNumber:   sellerInfo.LastBidID,
		EnergyQuantityKWH: quantityKWH,
		PricePerKWH:       pricePerKWH,
		EnergyType:        energyType,
	}

	//generate compositekey for the sellbid
	lastBidIDStr := strconv.FormatUint(uint64(sellerInfo.LastBidID), 10)
	keySellBid, err := stub.CreateCompositeKey("SellBid", []string{mspIDSeller, sellerID, lastBidIDStr})

	sellBidBytes, err := proto.Marshal(&sellBid)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutState(keySellBid, sellBidBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

/*
 @Param stub - to interact with the World State
 @Param mspIDPaymentCompany - MSP ID responsible to handle the off-chain payment between the seller and the IDEMIX buyer
 @Param token - token given to the IDEMIX buyer by the Payment Company to validate the funds
 @Param amountKWH - amount of energy to be put in the SellBid (in Kilowatt-hour)
 @Param pricePerKWH - desired price per Kilowatt-hour
 @Param energyType - the source of the energy (e.g. solar, wind)

 - This function registers a BuyBid and it is called by a buyer with IDEMIX CREDENTIALS!
 - The BuyBid is put in the World State for later validation by the Payment Company (see validateBuyBid())
*/
func (chaincode *EnergyChaincode) registerBuyBid(stub shim.ChaincodeStubInterface, mspIDPaymentCompany string, token string, utilityMspID string, quantityKWH float64, pricePerKWH float64, energyType string) pb.Response {
	println("---- registerBuyBid function beggining ----")

	//check if caller is a buyer
	err := cid.AssertAttributeValue(stub, "ou", "idemixorg")
	if err != nil {
		return shim.Error(err.Error())
	}

	if quantityKWH <= 0 || pricePerKWH <= 0 {
		return shim.Error("Energy Amount and Price per KWH must be greater than ZERO!")
	}

	energyTransactionsIterator, err := stub.GetStateByPartialCompositeKey("EnergyTransaction", []string{mspIDPaymentCompany, token})
	if err != nil {
		return shim.Error(err.Error())
	}

	key, err := stub.CreateCompositeKey("BuyBid", []string{"false", mspIDPaymentCompany, token})
	buyBidBytes, err := stub.GetState(key)

	//check if BuyBid is already registered based on the token
	if buyBidBytes != nil || energyTransactionsIterator.HasNext() {
		return shim.Error("BuyBid is ALREADY registered or token was already used in past EnergyTransaction!")
	}

	buyBid := st.BuyBid{
		MspIDPaymentCompany: mspIDPaymentCompany,
		Token:               token,
		UtilityMspID:        utilityMspID,
		EnergyQuantityKWH:   quantityKWH,
		PricePerKWH:         pricePerKWH,
		EnergyType:          energyType,
		Validated:           false,
	}

	buyBidBytes, err = proto.Marshal(&buyBid)
	if err != nil {
		return shim.Error(err.Error())
	}
	//save BuyBid
	err = stub.PutState(key, buyBidBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	printf("Registered BuyBid: %+v\n", buyBid)

	return shim.Success(nil)

}

/*
 @Param stub - to interact with the World State
 @Param token - token given to the IDEMIX buyer by the Payment Company to validate the funds
 @Param maxBidPayment - the max amount of money to be covered by the Payment Company for the BuyBid

 - This functions validates the BuyBid identified by the 'token'
 - After the validated, the BuyBid can be finally used in the Double Auction
*/
func (chaincode *EnergyChaincode) validateBuyBid(stub shim.ChaincodeStubInterface, token string, maxBuyBidPaymentCover float64) pb.Response {
	println("---- validateBuyBid function beggining ----")

	var buyBid st.BuyBid

	err := cid.AssertAttributeValue(stub, "energy.paymentcompany", "true")
	if err != nil {
		return shim.Error(err.Error())
	}

	paymentCompanyMspID, err := cid.GetMSPID(stub)
	key, err := stub.CreateCompositeKey("BuyBid", []string{"false", paymentCompanyMspID, token})
	buyBidBytes, err := stub.GetState(key)
	if buyBidBytes == nil {
		return shim.Error("Error retriving BuyBid of token " + token)
	}

	err = proto.Unmarshal(buyBidBytes, &buyBid)
	if err != nil {
		return shim.Error("Error unmarshaling BuyBid of token " + token)
	}

	if buyBid.Token != token {
		return shim.Error("Argument 'token' does not match BuyBid Token")
	}

	if buyBid.PricePerKWH*buyBid.EnergyQuantityKWH > maxBuyBidPaymentCover {
		return shim.Error("Payment Company cannot cover the cost of the BuyBid of token " + token)
	}

	buyBid.Validated = true
	buyBidBytes, err = proto.Marshal(&buyBid)
	if err != nil {
		return shim.Error(err.Error())
	}

	//delete BuyBid with the composite key "BuyBidfalse..."
	err = stub.DelState(key)
	if err != nil {
		return shim.Error(err.Error())
	}

	//... and replace with the composite key "BuyBidtrue..."
	key, err = stub.CreateCompositeKey("BuyBid", []string{"true", paymentCompanyMspID, token})

	err = stub.PutState(key, buyBidBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	//send it to auction
	printf("Validated BuyBid: %+v\n", buyBid)

	return shim.Success([]byte("BuyBid of token " + token + " validated!"))
}

func (chaincode *EnergyChaincode) auction(stub shim.ChaincodeStubInterface) pb.Response {
	//println("---- auction function beggining ----")

	/*now := time.Now()
	defer func() {
		elapsed := time.Since(now)
		printf("auction took: ")
		println(elapsed)
	}()*/

	//get SellBids
	sellBidsIterator, err := stub.GetStateByPartialCompositeKey("SellBid", []string{})
	if err != nil {
		return shim.Error(err.Error())
	}

	//get VALIDATED BuyBids
	//queryString := fmt.Sprintf(`{"selector":{"validated":true}}`)
	//println("Query string: " + queryString)
	//buyBidsIterator, err := stub.GetQueryResult(queryString)
	buyBidsIterator, err := stub.GetStateByPartialCompositeKey("BuyBid", []string{"true"})
	if err != nil {
		return shim.Error(err.Error())
	}

	//create a map that points to lists of SellBids with same ENERGY TYPE
	sellBidsByType := make(map[string][]st.SellBid)
	var sellBid st.SellBid
	for sellBidsIterator.HasNext() {
		queryResult, err := sellBidsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		sellBidBytes := queryResult.Value
		err = proto.Unmarshal(sellBidBytes, &sellBid)
		if err != nil {
			return shim.Error(err.Error())
		}
		sellBidsByType[sellBid.EnergyType] = append(sellBidsByType[sellBid.EnergyType], sellBid)
	}

	//create a map that points to lists of BuyBids with same ENERGY TYPE
	buyBidsByType := make(map[string][]st.BuyBid)
	var buyBid st.BuyBid
	for buyBidsIterator.HasNext() {
		queryResult, err := buyBidsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		buyBidBytes := queryResult.Value
		err = proto.Unmarshal(buyBidBytes, &buyBid)
		if err != nil {
			return shim.Error(err.Error())
		}
		buyBidsByType[buyBid.EnergyType] = append(buyBidsByType[buyBid.EnergyType], buyBid)
	}

	//match sell and buybids for each energy type
	for energyType, sellBidsEnergyType := range sellBidsByType {
		buyBidsEnergyType := buyBidsByType[energyType]
		//printf("%s energy auction sellBids: %+v\n", energyType, sellBidsEnergyType)
		//printf("%s energy auction buyBids: %+v\n", energyType, buyBidsEnergyType)
		if len(sellBidsEnergyType) > 0 && len(buyBidsEnergyType) > 0 {
			err := matchBuyAndSellBidsWithSameEnergyType(stub, sellBidsEnergyType, buyBidsEnergyType)
			if err != nil {
				return shim.Error(err.Error())
			}
		}
	}

	sellBidsIterator.Close()
	buyBidsIterator.Close()
	//return shim.Error("JUST TESTING auction TIME")
	//return shim.Success(nil)
	//ensuring that applications are warned when auction is performed
	stub.SetEvent("auctionPerformed", nil)
	return shim.SuccessWithPriorityBypassPhantomReadCheck([]byte("Auction success. THIS TRANSACTION HAS HIGH PRIORITY WITH THE ORDERER and bypasses PHANTOM_READ_CONFLICT"),
		pb.Priority_HIGH)
}

func matchBuyAndSellBidsWithSameEnergyType(stub shim.ChaincodeStubInterface, sellBids []st.SellBid, buyBids []st.BuyBid) error {
	//println("---- matchBuyAndSellBidsWithSameEnergyType function beggining ----")

	//println("")
	//println("---- MATCHING " + sellBids[0].EnergyType + " ENERGY BIDS ----")
	//println("")
	//sort SellBids in ASCENDING order
	sort.SliceStable(sellBids[:], func(i, j int) bool {
		return sellBids[i].PricePerKWH < sellBids[j].PricePerKWH
	})
	//printf("Sorted ASCENDING sellBids: %+v\n", sellBids)

	//sort BuyBids in DESCENDING order
	sort.SliceStable(buyBids[:], func(i, j int) bool {
		return buyBids[i].PricePerKWH > buyBids[j].PricePerKWH
	})
	//printf("Sorted DESCENDING buyBids: %+v\n", buyBids)

	//match the bids
	i, j := 0, 0
	excessEnergyToSell := sellBids[0].EnergyQuantityKWH
	neededEnergyToBuy := buyBids[0].EnergyQuantityKWH
	lastWelfarePrice := 0.0
	energyTransactions := []st.EnergyTransaction{}
	var err, err2 error = nil, nil

	//while the sell price is lower than or equal the buy price
	for sellBids[i].PricePerKWH <= buyBids[j].PricePerKWH {
		lastWelfarePrice = (sellBids[i].PricePerKWH + buyBids[j].PricePerKWH) / 2

		//if sellBid[i] has more energy than buyBid[j] wants to buy
		if excessEnergyToSell > neededEnergyToBuy {
			excessEnergyToSell -= neededEnergyToBuy
			//printf("BuyBid %+v\n is SATISFIED with %f KWH sold by \n SellBid %+v\n", buyBids[j], neededEnergyToBuy, sellBids[i])
			energyTransactions = append(energyTransactions, instantiateEnergyTransaction(&sellBids[i], &buyBids[j], neededEnergyToBuy))
			err = deleteBuyBidFromWorldState(stub, &buyBids[j])
			j++
			//test if there are more buyBids to fetch
			if j < len(buyBids) {
				neededEnergyToBuy = buyBids[j].EnergyQuantityKWH
			} else {
				neededEnergyToBuy = 0
				break
			}

			//if sellBid[i] has less energy than buyBid[j] wants to buy
		} else if excessEnergyToSell < neededEnergyToBuy {
			neededEnergyToBuy -= excessEnergyToSell
			//printf("SellBid: %+v\n is SATISFIED with %f KWH bought from \n BuyBid %+v\n", sellBids[i], excessEnergyToSell, buyBids[j])
			energyTransactions = append(energyTransactions, instantiateEnergyTransaction(&sellBids[i], &buyBids[j], excessEnergyToSell))
			err = deleteSellBidFromWorldState(stub, &sellBids[i])
			i++
			//test if there are more sellBids to fetch
			if i < len(sellBids) {
				excessEnergyToSell = sellBids[i].EnergyQuantityKWH
			} else {
				excessEnergyToSell = 0
				break
			}

			//if sellBid[i] has the exact amount of energy that buyBid[j] wants to buy
		} else {
			//printf("SellBid: %+v\n AND BuyBid %+v\n  SATISFIED each other exchanging %f KWH\n", sellBids[i], buyBids[j], neededEnergyToBuy)
			energyTransactions = append(energyTransactions, instantiateEnergyTransaction(&sellBids[i], &buyBids[j], neededEnergyToBuy))
			err = deleteBuyBidFromWorldState(stub, &buyBids[j])
			err2 = deleteSellBidFromWorldState(stub, &sellBids[i])
			i++
			j++
			//test if there are more sellBids and buyBids to fetch
			if j < len(buyBids) && i < len(sellBids) {
				excessEnergyToSell = sellBids[i].EnergyQuantityKWH
				neededEnergyToBuy = buyBids[j].EnergyQuantityKWH
			} else {
				excessEnergyToSell = 0
				neededEnergyToBuy = 0
				break
			}
		}
		if err != nil || err2 != nil {
			return fmt.Errorf("Problem while deleting some bid from world state")
		}
	}

	//update buy and sell bids that were not FULLY satisfied
	if excessEnergyToSell > 0 {
		//if sellBid[i] was only partially satisfied, update it in World State
		if excessEnergyToSell < sellBids[i].EnergyQuantityKWH {
			toBeUpdatedSellBid := sellBids[i]
			toBeUpdatedSellBid.EnergyQuantityKWH = excessEnergyToSell
			err := updateSellBid(stub, toBeUpdatedSellBid)
			if err != nil {
				return err
			}
		}
	}

	if neededEnergyToBuy > 0 {
		//if buyBid[j] was only partially satisfied, update it in World State
		if neededEnergyToBuy < buyBids[j].EnergyQuantityKWH {
			toBeUpdatedBuyBid := buyBids[j]
			toBeUpdatedBuyBid.EnergyQuantityKWH = neededEnergyToBuy
			err := updateBuyBid(stub, toBeUpdatedBuyBid)
			if err != nil {
				return err
			}
		}
	}

	//update EnergyTransactions with the Welfare Energy Price
	for index := range energyTransactions {
		energyTransactions[index].PricePerKWH = lastWelfarePrice
	}

	//save EnergyTransactions
	err = saveEnergyTransactions(stub, &energyTransactions)

	//printf("%s EnergyTransactions: %+v\n", sellBids[0].EnergyType, energyTransactions)

	return err
}

func instantiateEnergyTransaction(sellBid *st.SellBid, buyBid *st.BuyBid, energySettled float64) st.EnergyTransaction {
	//println("---- instatiateEnergyTransaction function beggining ----")

	energyTransaction := st.EnergyTransaction{
		MspIDSeller:         sellBid.MspIDSeller,
		SellerID:            sellBid.SellerID,
		SellerBidNumber:     sellBid.SellerBidNumber,
		MspIDPaymentCompany: buyBid.MspIDPaymentCompany,
		Token:               buyBid.Token,
		BuyerUtilityMspID:   buyBid.UtilityMspID,
		EnergyQuantityKWH:   energySettled,
		PricePerKWH:         -1,
		EnergyType:          sellBid.EnergyType,
	}

	return energyTransaction
}

func deleteSellBidFromWorldState(stub shim.ChaincodeStubInterface, sellBid *st.SellBid) error {
	var err error = nil
	sellerBidNumberStr := strconv.FormatUint(uint64(sellBid.SellerBidNumber), 10)
	key, _ := stub.CreateCompositeKey("SellBid", []string{sellBid.MspIDSeller, sellBid.SellerID, sellBid.EnergyType, sellerBidNumberStr})
	err = stub.DelState(key)
	return err
}

func deleteBuyBidFromWorldState(stub shim.ChaincodeStubInterface, buyBid *st.BuyBid) error {
	var err error = nil

	var validated string
	if buyBid.Validated {
		validated = "true"
	} else {
		validated = "false"
	}
	key, _ := stub.CreateCompositeKey("BuyBid", []string{validated, buyBid.MspIDPaymentCompany, buyBid.Token})
	err = stub.DelState(key)
	return err
}

func updateSellBid(stub shim.ChaincodeStubInterface, sellBid st.SellBid) error {
	//println("---- updateSellBid function beggining ----")

	key, err := stub.CreateCompositeKey("SellBid", []string{sellBid.MspIDSeller, sellBid.SellerID})
	if err != nil {
		return err
	}

	sellBidBytes, err := proto.Marshal(&sellBid)
	if err != nil {
		return err
	}

	err = stub.PutState(key, sellBidBytes)
	if err != nil {
		return err
	}
	return nil
}

func updateBuyBid(stub shim.ChaincodeStubInterface, buyBid st.BuyBid) error {
	//println("---- updateBuyBid function beggining ----")

	var validated string
	if buyBid.Validated {
		validated = "true"
	} else {
		validated = "false"
	}
	key, err := stub.CreateCompositeKey("BuyBid", []string{validated, buyBid.MspIDPaymentCompany, buyBid.Token})
	if err != nil {
		return err
	}

	buyBidBytes, err := proto.Marshal(&buyBid)
	if err != nil {
		return err
	}

	err = stub.PutState(key, buyBidBytes)
	if err != nil {
		return err
	}
	return nil
}

func saveEnergyTransactions(stub shim.ChaincodeStubInterface, energyTransactions *[]st.EnergyTransaction) error {
	//println("---- saveEnergyTransactions function beggining ----")

	sellBidEnergyTrans := make(map[string]st.SellBidEnergyTransactions)

	for _, energyTransaction := range *energyTransactions {
		sellerBidNumberStr := strconv.FormatUint(uint64(energyTransaction.SellerBidNumber), 10)
		key, _ := stub.CreateCompositeKey("EnergyTransaction", []string{energyTransaction.MspIDPaymentCompany, energyTransaction.Token, energyTransaction.MspIDSeller, energyTransaction.SellerID, sellerBidNumberStr})

		energyTransactionBytes, err := proto.Marshal(&energyTransaction)
		if err != nil {
			return err
		}
		stub.PutState(key, energyTransactionBytes)

		key, _ = stub.CreateCompositeKey("SellBidEnergyTransactions", []string{energyTransaction.MspIDSeller, energyTransaction.SellerID, sellerBidNumberStr})
		sellBidTransactions := sellBidEnergyTrans[key]
		sellBidTransactions.FullTokens = append(sellBidTransactions.FullTokens,
			&st.FullToken{
				MspIDPaymentCompany: energyTransaction.MspIDPaymentCompany,
				Token:               energyTransaction.Token})
		sellBidEnergyTrans[key] = sellBidTransactions
	}

	for key, sellBidTransactions := range sellBidEnergyTrans {
		sellBidTransactionBytes, err := proto.Marshal(&sellBidTransactions)
		if err != nil {
			return err
		}
		stub.PutState(key, sellBidTransactionBytes)
	}

	return nil
}

func (chaincode *EnergyChaincode) transactionsEnergyQuantityFromPaymentToken(stub shim.ChaincodeStubInterface, mspIDPaymentCompany string, token string) pb.Response {
	println("---- transactionsEnergyQuantityFromPaymentToken function beggining ----")

	energyTransactionsIterator, err := stub.GetStateByPartialCompositeKey("EnergyTransaction", []string{mspIDPaymentCompany, token})
	if err != nil {
		return shim.Error(err.Error())
	}

	energyQuantityOnTransactions := 0.0
	var energyTransactionAux st.EnergyTransaction
	for energyTransactionsIterator.HasNext() {
		queryResult, err := energyTransactionsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		proto.Unmarshal(queryResult.Value, &energyTransactionAux)
		energyQuantityOnTransactions += energyTransactionAux.EnergyQuantityKWH
	}

	if energyQuantityOnTransactions > 0 {
		return shim.Success([]byte(fmt.Sprintf("%f", energyQuantityOnTransactions)))
	}
	return shim.Success([]byte("0"))
}

func (chaincode *EnergyChaincode) getEnergyTransactionsFromPaymentToken(stub shim.ChaincodeStubInterface, mspIDPaymentCompany string, token string) pb.Response {
	println("---- getEnergyTransactionsFromPaymentToken function beggining ----")

	err := cid.AssertAttributeValue(stub, "energy.utility", "true")
	if err != nil {
		return shim.Error(err.Error())
	}

	energyTransactionsIterator, err := stub.GetStateByPartialCompositeKey("EnergyTransaction", []string{mspIDPaymentCompany, token})
	if err != nil {
		return shim.Error(err.Error())
	}

	if energyTransactionsIterator.HasNext() != true {
		return shim.Error("No Energy Transaction formed from the payment token: " + mspIDPaymentCompany + token)
	}

	energyTransactionsJSON := ""
	var energyTransactionAux st.EnergyTransaction
	for energyTransactionsIterator.HasNext() {
		queryResult, err := energyTransactionsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		//energyTransactionsJSON = energyTransactionsJSON[:len(energyTransactionsJSON)] + string(queryResult.Value) + "," // WHEN WE USED JSON
		proto.Unmarshal(queryResult.Value, &energyTransactionAux)
		transactionStr := protojson.Format(proto.MessageReflect(&energyTransactionAux).Interface())
		println("transactionStr: " + transactionStr)
		energyTransactionsJSON = energyTransactionsJSON[:len(energyTransactionsJSON)] + transactionStr + ","
	}

	energyTransactionsJSON = "[" + energyTransactionsJSON[:len(energyTransactionsJSON)-1] + "]"

	return shim.Success([]byte(energyTransactionsJSON))
}

func (chaincode *EnergyChaincode) getEnergyTransactionsFromSellBidNumbers(stub shim.ChaincodeStubInterface, sellBidNumbers []string) pb.Response {
	println("---- getEnergyTransactionsFromSellBidNumbers function beggining ----")

	err := cid.AssertAttributeValue(stub, "energy.seller", "true")
	if err != nil {
		return shim.Error(err.Error())
	}

	mspIDSeller, err := cid.GetMSPID(stub)
	sellerID, err := cid.GetID(stub)

	energyTransactionsJSON := ""
	for _, sellBidNumber := range sellBidNumbers {
		pbResponse := chaincode.getEnergyTransactionsFromFullSellBidKey(stub, mspIDSeller, sellerID, sellBidNumber)
		if pbResponse.Status == shim.OK {
			energyTransactionsJSONSellBid := string(pbResponse.GetPayload())
			energyTransactionsJSON += energyTransactionsJSONSellBid + ","
		}
	}
	if len(energyTransactionsJSON) > 0 {
		energyTransactionsJSON = "[" + energyTransactionsJSON[:len(energyTransactionsJSON)-1] + "]"
	} else {
		energyTransactionsJSON = "[" + "]"
	}

	return shim.Success([]byte(energyTransactionsJSON))
}

func (chaincode *EnergyChaincode) getEnergyTransactionsFromFullSellBidKey(stub shim.ChaincodeStubInterface, mspIDSeller string, sellerID string, sellBidNumberStr string) pb.Response {
	println("---- getEnergyTransactionsFromFullSellBidKey function beggining ----")

	key, _ := stub.CreateCompositeKey("SellBidEnergyTransactions", []string{mspIDSeller, sellerID, sellBidNumberStr})
	sellBidEnergyTransactionsBytes, err := stub.GetState(key)
	if err != nil {
		return shim.Error(err.Error())
	}

	if sellBidEnergyTransactionsBytes == nil {
		return shim.Error("No Energy Transaction formed from the sell bid: " + mspIDSeller + sellerID + sellBidNumberStr)
	}

	var sellBidEnergyTrans st.SellBidEnergyTransactions
	proto.Unmarshal(sellBidEnergyTransactionsBytes, &sellBidEnergyTrans)

	energyTransactionsJSON := ""
	var energyTransactionAux st.EnergyTransaction
	for _, fullToken := range sellBidEnergyTrans.FullTokens {
		key, _ := stub.CreateCompositeKey("EnergyTransaction", []string{fullToken.MspIDPaymentCompany, fullToken.Token, mspIDSeller, sellerID, sellBidNumberStr})
		energyTransactionsBytes, err := stub.GetState(key)
		if err != nil {
			return shim.Error(err.Error())
		}
		//energyTransactionsJSON = energyTransactionsJSON[:len(energyTransactionsJSON)] + string(energyTransactionsBytes) + ","  // WHEN WE USED JSON
		proto.Unmarshal(energyTransactionsBytes, &energyTransactionAux)
		transactionStr := protojson.Format(proto.MessageReflect(&energyTransactionAux).Interface())
		println("transactionStr: " + transactionStr)
		energyTransactionsJSON = energyTransactionsJSON[:len(energyTransactionsJSON)] + transactionStr + ","
	}
	energyTransactionsJSON = "[" + energyTransactionsJSON[:len(energyTransactionsJSON)-1] + "]"

	return shim.Success([]byte(energyTransactionsJSON))
}

//talvez deletar
func (chaincode *EnergyChaincode) getEnergyTransactionsFromSellerAfterSellBidNumber(stub shim.ChaincodeStubInterface, mspIDSeller string, sellerID string, initialSellBidNumber uint64) pb.Response {
	println("---- getEnergyTransactionsFromSellerAfterSellBidNumber function beggining ----")

	var sellerInfo st.SellerInfo
	key, _ := stub.CreateCompositeKey("SellerInfo", []string{mspIDSeller, sellerID})
	sellerInfoBytes, err := stub.GetState(key)
	if err != nil {
		return shim.Error(err.Error())
	}
	proto.Unmarshal(sellerInfoBytes, &sellerInfo)

	finalSellBid := sellerInfo.LastBidID
	energyTransactionsJSON := ""
	for sellBidNum := initialSellBidNumber; sellBidNum <= finalSellBid; sellBidNum++ {
		sellBidNumStr := strconv.FormatUint(uint64(sellBidNum), 10)
		pbResponse := chaincode.getEnergyTransactionsFromFullSellBidKey(stub, mspIDSeller, sellerID, sellBidNumStr)
		if pbResponse.Status == shim.OK {
			energyTransactionsJSONSellBid := string(pbResponse.GetPayload())
			energyTransactionsJSON += energyTransactionsJSONSellBid[1:len(energyTransactionsJSONSellBid)-1] + ","
		}
	}
	if len(energyTransactionsJSON) > 0 {
		energyTransactionsJSON = "[" + energyTransactionsJSON[:len(energyTransactionsJSON)-1] + "]"
	} else {
		energyTransactionsJSON = "[" + "]"
	}

	return shim.Success([]byte(energyTransactionsJSON))
}

func (chaincode *EnergyChaincode) getCallerIDAndCallerMspID(stub shim.ChaincodeStubInterface) pb.Response {

	callerID, err := cid.GetID(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	mspID, err := cid.GetMSPID(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	println("Caller ID: " + callerID)
	println("Caller MSP ID: " + mspID)

	return shim.Success([]byte("Caller ID: " + callerID + "\nCaller MSP ID: " + mspID))
}

func (chaincode *EnergyChaincode) auctionSortedQueries(stub shim.ChaincodeStubInterface) pb.Response {
	//println("---- auctionSortedQueries function beggining ----")

	/*now := time.Now()
	defer func() {
		elapsed := time.Since(now)
		printf("auctionSortedQueries took: ")
		println(elapsed)
	}()*/

	//get SellBids
	queryString := fmt.Sprintf(`{"selector":{"issellbid":true}, "sort": [{"priceperkwh": "asc"}]}`)
	//println("Query string: " + queryString)
	sellBidsIterator, err := stub.GetQueryResult(queryString)
	if err != nil {
		return shim.Error(err.Error())
	}

	//get VALIDATED BuyBids
	queryString = fmt.Sprintf(`{"selector":{"validated":true}, "sort": [{"priceperkwh": "desc"}]}`)
	//println("Query string: " + queryString)
	buyBidsIterator, err := stub.GetQueryResult(queryString)
	if err != nil {
		return shim.Error(err.Error())
	}

	//create a map that points to lists of SellBids with same ENERGY TYPE
	sellBidsByType := make(map[string][]st.SellBid)
	var sellBid st.SellBid
	for sellBidsIterator.HasNext() {
		queryResult, err := sellBidsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		sellBidBytes := queryResult.Value
		err = proto.Unmarshal(sellBidBytes, &sellBid)
		if err != nil {
			return shim.Error(err.Error())
		}
		sellBidsByType[sellBid.EnergyType] = append(sellBidsByType[sellBid.EnergyType], sellBid)
	}

	//create a map that points to lists of BuyBids with same ENERGY TYPE
	buyBidsByType := make(map[string][]st.BuyBid)
	var buyBid st.BuyBid
	for buyBidsIterator.HasNext() {
		queryResult, err := buyBidsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		buyBidBytes := queryResult.Value
		err = proto.Unmarshal(buyBidBytes, &buyBid)
		if err != nil {
			return shim.Error(err.Error())
		}
		buyBidsByType[buyBid.EnergyType] = append(buyBidsByType[buyBid.EnergyType], buyBid)
	}

	//match sell and buybids for each energy type
	for energyType, sellBidsEnergyType := range sellBidsByType {
		buyBidsEnergyType := buyBidsByType[energyType]
		//printf("%s energy auction sellBids: %+v\n", energyType, sellBidsEnergyType)
		//printf("%s energy auction buyBids: %+v\n", energyType, buyBidsEnergyType)
		if len(sellBidsEnergyType) > 0 && len(buyBidsEnergyType) > 0 {
			err := matchBuyAndSellBidsWithSameEnergyTypeSortedQueries(stub, sellBidsEnergyType, buyBidsEnergyType)
			if err != nil {
				return shim.Error(err.Error())
			}
		}
	}

	sellBidsIterator.Close()
	buyBidsIterator.Close()
	//return shim.Error("JUST TESTING auctionSortedQueries TIME")
	return shim.Success(nil)
}

func matchBuyAndSellBidsWithSameEnergyTypeSortedQueries(stub shim.ChaincodeStubInterface, sellBids []st.SellBid, buyBids []st.BuyBid) error {
	//println("---- matchBuyAndSellBidsWithSameEnergyType function beggining ----")

	//println("")
	//println("---- MATCHING " + sellBids[0].EnergyType + " ENERGY BIDS ----")
	//println("")

	//match the bids
	i, j := 0, 0
	excessEnergyToSell := sellBids[0].EnergyQuantityKWH
	neededEnergyToBuy := buyBids[0].EnergyQuantityKWH
	lastWelfarePrice := 0.0
	energyTransactions := []st.EnergyTransaction{}
	var err, err2 error = nil, nil

	//while the sell price is lower than or equal the buy price
	for sellBids[i].PricePerKWH <= buyBids[j].PricePerKWH {
		lastWelfarePrice = (sellBids[i].PricePerKWH + buyBids[j].PricePerKWH) / 2

		//if sellBid[i] has more energy than buyBid[j] wants to buy
		if excessEnergyToSell > neededEnergyToBuy {
			excessEnergyToSell -= neededEnergyToBuy
			//printf("BuyBid %+v\n is SATISFIED with %f KWH sold by \n SellBid %+v\n", buyBids[j], neededEnergyToBuy, sellBids[i])
			energyTransactions = append(energyTransactions, instantiateEnergyTransaction(&sellBids[i], &buyBids[j], neededEnergyToBuy))
			err = deleteBuyBidFromWorldState(stub, &buyBids[j])
			j++
			//test if there are more buyBids to fetch
			if j < len(buyBids) {
				neededEnergyToBuy = buyBids[j].EnergyQuantityKWH
			} else {
				neededEnergyToBuy = 0
				break
			}

			//if sellBid[i] has less energy than buyBid[j] wants to buy
		} else if excessEnergyToSell < neededEnergyToBuy {
			neededEnergyToBuy -= excessEnergyToSell
			//printf("SellBid: %+v\n is SATISFIED with %f KWH bought from \n BuyBid %+v\n", sellBids[i], excessEnergyToSell, buyBids[j])
			energyTransactions = append(energyTransactions, instantiateEnergyTransaction(&sellBids[i], &buyBids[j], excessEnergyToSell))
			err = deleteSellBidFromWorldState(stub, &sellBids[i])
			i++
			//test if there are more sellBids to fetch
			if i < len(sellBids) {
				excessEnergyToSell = sellBids[i].EnergyQuantityKWH
			} else {
				excessEnergyToSell = 0
				break
			}

			//if sellBid[i] has the exact amount of energy that buyBid[j] wants to buy
		} else {
			//printf("SellBid: %+v\n AND BuyBid %+v\n  SATISFIED each other exchanging %f KWH\n", sellBids[i], buyBids[j], neededEnergyToBuy)
			energyTransactions = append(energyTransactions, instantiateEnergyTransaction(&sellBids[i], &buyBids[j], neededEnergyToBuy))
			err = deleteBuyBidFromWorldState(stub, &buyBids[j])
			err2 = deleteSellBidFromWorldState(stub, &sellBids[i])
			i++
			j++
			//test if there are more sellBids and buyBids to fetch
			if j < len(buyBids) && i < len(sellBids) {
				excessEnergyToSell = sellBids[i].EnergyQuantityKWH
				neededEnergyToBuy = buyBids[j].EnergyQuantityKWH
			} else {
				excessEnergyToSell = 0
				neededEnergyToBuy = 0
				break
			}
		}
		if err != nil || err2 != nil {
			return fmt.Errorf("Problem while deleting some bid from world state")
		}
	}

	//update buy and sell bids that were not FULLY satisfied
	if excessEnergyToSell > 0 {
		//if sellBid[i] was only partially satisfied, update it in World State
		if excessEnergyToSell < sellBids[i].EnergyQuantityKWH {
			toBeUpdatedSellBid := sellBids[i]
			toBeUpdatedSellBid.EnergyQuantityKWH = excessEnergyToSell
			err := updateSellBid(stub, toBeUpdatedSellBid)
			if err != nil {
				return err
			}
		}
	}

	if neededEnergyToBuy > 0 {
		//if buyBid[j] was only partially satisfied, update it in World State
		if neededEnergyToBuy < buyBids[j].EnergyQuantityKWH {
			toBeUpdatedBuyBid := buyBids[j]
			toBeUpdatedBuyBid.EnergyQuantityKWH = neededEnergyToBuy
			err := updateBuyBid(stub, toBeUpdatedBuyBid)
			if err != nil {
				return err
			}
		}
	}

	//update EnergyTransactions with the Welfare Energy Price
	for index := range energyTransactions {
		energyTransactions[index].PricePerKWH = lastWelfarePrice
	}

	//save EnergyTransactions
	err = saveEnergyTransactions(stub, &energyTransactions)

	//printf("%s EnergyTransactions: %+v\n", sellBids[0].EnergyType, energyTransactions)

	return err
}

// Init initializes the chaincode
func (chaincode *EnergyChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {

	println("energy Init")

	//
	// Demonstrate the use of Attribute-Based Access Control (ABAC) by checking
	// to see if the caller has the "abac.init" attribute with a value of true;
	// if not, return an error.
	//
	err := cid.AssertAttributeValue(stub, "energy.init", "true")
	if err != nil {
		return shim.Error(err.Error())
	}

	//_, args := stub.GetFunctionAndParameters()

	return shim.Success(nil)

}

func recalculateFunctionAverageTime() {
	for {
		functionNameAndDuration := <-channelAverageCalculator
		printf("%+v\n", functionNameAndDuration)
		function := functionNameAndDuration.FunctionName
		elapsed := functionNameAndDuration.Duration

		functionStats := averageFunctionTimes[function]
		n := float64(functionStats.NCalls)
		functionStats.NCalls++
		functionStats.AvarageExecTimeMs = (n/(n+1))*functionStats.AvarageExecTimeMs +
			float64(elapsed.Milliseconds())/(n+1)
	}

}

//Invoke calls a function using the "peer chaincode invoke" command
func (chaincode *EnergyChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	println("energy Invoke")
	function, args := stub.GetFunctionAndParameters()

	if functionPointer, ok := functionMap[function]; ok {
		now := time.Now()
		defer func() {
			elapsed := time.Since(now)
			channelAverageCalculator <- &FunctionAndDuration{function, elapsed}
			printf("Function: %s took \n", function)
			println(elapsed)
		}()
		return functionPointer(stub, args)
	}

	/*lenArgs := len(args)
	if function == "sensorDeclareActive" {
		return chaincode.sensorDeclareActive(stub)
	} else if function == "disableSensors" {
		return chaincode.disableSensors(stub, args)
	} else if function == "enableSensors" {
		return chaincode.enableSensors(stub, args)
	} else if function == "getActiveSensors" {
		return chaincode.getActiveSensors(stub)
	} else if function == "setTrustedSensors" {
		return chaincode.setTrustedSensors(stub, args[:lenArgs/2], args[lenArgs/2:])
	} else if function == "setDistrustedSensors" {
		return chaincode.setDistrustedSensors(stub, args[:lenArgs/2], args[lenArgs/2:])
	} else if function == "getTrustedSensors" {
		return chaincode.getTrustedSensors(stub)
	} else if function == "publishSensorData" {
		versionInt, _ := strconv.Atoi(args[0])
		version := int8(versionInt)
		unit64, _ := strconv.ParseUint(args[1], 10, 32)
		unit := uint32(unit64)
		timestamp, _ := strconv.ParseUint(args[2], 10, 64)
		value, _ := strconv.ParseFloat(args[3], 64)
		eUint, _ := strconv.ParseUint(args[4], 10, 8)
		e := uint8(eUint)
		confidenceUint, _ := strconv.ParseUint(args[5], 10, 8)
		confidence := uint8(confidenceUint)
		dev, _ := strconv.ParseUint(args[6], 10, 32)
		return chaincode.publishSensorData(stub, version, unit, timestamp, value, e, confidence, uint32(dev))
	} else if function == "getSensorsPublishedData" {
		return chaincode.getSensorsPublishedData(stub, args)
	} else if function == "registerSeller" {
		windTurbinesNumber, _ := strconv.ParseUint(args[3], 10, 64)
		solarPanelsNumber, _ := strconv.ParseUint(args[4], 10, 64)
		return chaincode.registerSeller(stub, args[0], args[1], args[2], windTurbinesNumber, solarPanelsNumber)
	} else if function == "publishEnergyGeneration" {
		t0, _ := strconv.ParseUint(args[0], 10, 64)
		t1, _ := strconv.ParseUint(args[1], 10, 64)
		if len(args)%2 != 0 || len(args) > 2+len(EnergyTypes) {
			return shim.Error("Wrong number of arguments for function publishEnergyGeneration")
		}
		energyByTypeGeneratedKWH := make(map[string]float64)
		for i := 2; i < len(args); i += 2 {
			energyByTypeGeneratedKWH[args[i]], _ = strconv.ParseFloat(args[i+1], 64)
		}
		return chaincode.publishEnergyGeneration(stub, t0, t1, energyByTypeGeneratedKWH)
	} else if function == "registerSellBid" {
		amountKWH, _ := strconv.ParseFloat(args[0], 64)
		pricePerKWH, _ := strconv.ParseFloat(args[1], 64)
		return chaincode.registerSellBid(stub, amountKWH, pricePerKWH, args[2])
	} else if function == "registerBuyBid" {
		amountKWH, _ := strconv.ParseFloat(args[3], 64)
		pricePerKWH, _ := strconv.ParseFloat(args[4], 64)
		return chaincode.registerBuyBid(stub, args[0], args[1], args[2], amountKWH, pricePerKWH, args[5])
	} else if function == "validateBuyBid" {
		maxBuyBidPaymentCover, _ := strconv.ParseFloat(args[1], 64)
		return chaincode.validateBuyBid(stub, args[0], maxBuyBidPaymentCover)
	} else if function == "getCallerIDAndCallerMspID" {
		return chaincode.getCallerIDAndCallerMspID(stub)
	} else if function == "auction" {
		return chaincode.auction(stub)
	} else if function == "transactionsEnergyQuantityFromPaymentToken" {
		return chaincode.transactionsEnergyQuantityFromPaymentToken(stub, args[0], args[1])
	} else if function == "getEnergyTransactionsFromPaymentToken" {
		return chaincode.getEnergyTransactionsFromPaymentToken(stub, args[0], args[1])
	} else if function == "getEnergyTransactionsFromSellBidNumbers" {
		return chaincode.getEnergyTransactionsFromSellBidNumbers(stub, args)
	} else if function == "getEnergyTransactionsFromFullSellBidKey" {
		return chaincode.getEnergyTransactionsFromFullSellBidKey(stub, args[0], args[1], args[2])
	} else if function == "getEnergyTransactionsFromSellerAfterSellBidNumber" {
		initialSellBidNum, _ := strconv.ParseUint(args[2], 10, 64)
		return chaincode.getEnergyTransactionsFromSellerAfterSellBidNumber(stub, args[0], args[1], initialSellBidNum)
		////////// TESTING FUNCTIONS ////////////////
	} else if function == "auctionSortedQueries" {
		return chaincode.auctionSortedQueries(stub)

	} else {
		if function == "registerMultipleSellBids" {
			nSellBids, _ := strconv.ParseUint(args[0], 10, 64)
			minAmountKWH, _ := strconv.ParseFloat(args[1], 64)
			maxAmountKWH, _ := strconv.ParseFloat(args[2], 64)
			minPricePerKWH, _ := strconv.ParseFloat(args[3], 64)
			maxPricePerKWH, _ := strconv.ParseFloat(args[4], 64)
			return chaincode.registerMultipleSellBids(stub, nSellBids, minAmountKWH, maxAmountKWH, minPricePerKWH, maxPricePerKWH, args[5])
		} else if function == "registerMultipleBuyBids" {
			nBuyBids, _ := strconv.Atoi(args[0])
			minAmountKWH, _ := strconv.ParseFloat(args[2], 64)
			maxAmountKWH, _ := strconv.ParseFloat(args[3], 64)
			minPricePerKWH, _ := strconv.ParseFloat(args[4], 64)
			maxPricePerKWH, _ := strconv.ParseFloat(args[5], 64)
			return chaincode.registerMultipleBuyBids(stub, nBuyBids, args[1], minAmountKWH, maxAmountKWH, minPricePerKWH, maxPricePerKWH, args[6])
		} else if function == "validateMultipleBuyBids" {
			nBuyBids, _ := strconv.Atoi(args[0])
			return chaincode.validateMultipleBuyBids(stub, nBuyBids)
		} else if function == "clearSellBids" {
			return chaincode.clearSellBids(stub)
		} else if function == "clearBuyBids" {
			return chaincode.clearBuyBids(stub)
		} else if function == "printDataQuantityByPartialCompositeKey" {
			if len(args) < 2 {
				return chaincode.printDataQuantityByPartialCompositeKey(stub, args[0], []string{})
			}
			return chaincode.printDataQuantityByPartialCompositeKey(stub, args[0], args[1:len(args)])
		} else if function == "deleteDataByPartialCompositeKey" {
			if len(args) < 2 {
				return chaincode.deleteDataByPartialCompositeKey(stub, args[0], []string{})
			}
			return chaincode.deleteDataByPartialCompositeKey(stub, args[0], args[1:len(args)])
		} else if function == "printDataQuantityByPartialSimpleKey" {
			return chaincode.printDataQuantityByPartialSimpleKey(stub, args[0])
		} else if function == "deleteDataByPartialSimpleKey" {
			return chaincode.deleteDataByPartialSimpleKey(stub, args[0])
		} else if function == "registerMultipleSellers" {
			nSellers, _ := strconv.Atoi(args[0])
			return chaincode.registerMultipleSellers(stub, nSellers)
		} else if function == "measureTimeDifferentAuctions" {
			repeatAuction, _ := strconv.Atoi(args[0])
			return chaincode.measureTimeDifferentAuctions(stub, repeatAuction)
		} else if function == "testWorldStateLogic" {
			return chaincode.testWorldStateLogic(stub)
		} else if function == "sensorDeclareActiveTestContext" {
			return chaincode.sensorDeclareActiveTestContext(stub, args[0])
		} else if function == "publishSensorDataTestContext" {
			versionInt, _ := strconv.Atoi(args[1])
			version := int8(versionInt)
			unit64, _ := strconv.ParseUint(args[2], 10, 32)
			unit := uint32(unit64)
			timestamp, _ := strconv.ParseUint(args[3], 10, 64)
			value, _ := strconv.ParseFloat(args[4], 64)
			eUint, _ := strconv.ParseUint(args[5], 10, 8)
			e := uint8(eUint)
			confidenceUint, _ := strconv.ParseUint(args[6], 10, 8)
			confidence := uint8(confidenceUint)
			dev, _ := strconv.ParseUint(args[7], 10, 32)
			return chaincode.publishSensorDataTestContext(stub, args[0], version, unit, timestamp, value, e, confidence, uint32(dev))
		} else if function == "registerSellerTestContext" {
			windTurbinesNumber, _ := strconv.ParseUint(args[1], 10, 64)
			solarPanelsNumber, _ := strconv.ParseUint(args[2], 10, 64)
			return chaincode.registerSellerTestContext(stub, args[0], windTurbinesNumber, solarPanelsNumber)
		} else if function == "publishEnergyGenerationTestContext" {
			t0, _ := strconv.ParseUint(args[1], 10, 64)
			t1, _ := strconv.ParseUint(args[2], 10, 64)
			if len(args)%2 == 0 || len(args) > 3+len(EnergyTypes) {
				return shim.Error("Wrong number of arguments for function publishEnergyGenerationTestContext")
			}
			energyByTypeGeneratedKWH := make(map[string]float64)
			for i := 3; i < len(args); i += 2 {
				energyByTypeGeneratedKWH[args[i]], _ = strconv.ParseFloat(args[i+1], 64)
			}
			return chaincode.publishEnergyGenerationTestContext(stub, args[0], t0, t1, energyByTypeGeneratedKWH)
		} else if function == "registerSellBidTestContext" {
			amountKWH, _ := strconv.ParseFloat(args[1], 64)
			pricePerKWH, _ := strconv.ParseFloat(args[2], 64)
			return chaincode.registerSellBidTestContext(stub, args[0], amountKWH, pricePerKWH, args[3])
		} else if function == "getEnergyTransactionsFromSellBidNumbersTestContext" {
			return chaincode.getEnergyTransactionsFromSellBidNumbersTestContext(stub, args[0], args[1:len(args)])
		} else if function == "validateBuyBidTestContext" {
			return chaincode.validateBuyBidTestContext(stub, args[0], args[1])
		}
	}*/
	return shim.Error("Invalid invoke function name")
}

func initFuncMap(chaincode *EnergyChaincode) {
	functionMap = map[string]func(shim.ChaincodeStubInterface, []string) pb.Response{
		"getAverageFunctionTimes": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			return chaincode.getAverageFunctionTimes()
		},
		"setPrint": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			parsedBool, _ := strconv.ParseBool(args[0])
			return chaincode.setPrint(parsedBool)
		},
		"sensorDeclareActive": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			return chaincode.sensorDeclareActive(stub)
		},
		"disableSensors": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			return chaincode.disableSensors(stub, args)
		},
		"enableSensors": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			return chaincode.enableSensors(stub, args)
		},
		"getActiveSensors": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			return chaincode.getActiveSensors(stub)
		},
		"setTrustedSensors": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			lenArgs := len(args)
			return chaincode.setTrustedSensors(stub, args[:lenArgs/2], args[lenArgs/2:])
		},
		"setDistrustedSensors": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			lenArgs := len(args)
			return chaincode.setDistrustedSensors(stub, args[:lenArgs/2], args[lenArgs/2:])
		},
		"getTrustedSensors": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			return chaincode.getTrustedSensors(stub)
		},
		"publishSensorData": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			versionInt, _ := strconv.Atoi(args[0])
			version := int8(versionInt)
			unit64, _ := strconv.ParseUint(args[1], 10, 32)
			unit := uint32(unit64)
			timestamp, _ := strconv.ParseUint(args[2], 10, 64)
			value, _ := strconv.ParseFloat(args[3], 64)
			eUint, _ := strconv.ParseUint(args[4], 10, 8)
			e := uint8(eUint)
			confidenceUint, _ := strconv.ParseUint(args[5], 10, 8)
			confidence := uint8(confidenceUint)
			dev, _ := strconv.ParseUint(args[6], 10, 32)
			return chaincode.publishSensorData(stub, version, unit, timestamp, value, e, confidence, uint32(dev))
		},
		"getSensorsPublishedData": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			return chaincode.getSensorsPublishedData(stub, args)
		},
		"registerSeller": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			windTurbinesNumber, _ := strconv.ParseUint(args[3], 10, 64)
			solarPanelsNumber, _ := strconv.ParseUint(args[4], 10, 64)
			return chaincode.registerSeller(stub, args[0], args[1], args[2], windTurbinesNumber, solarPanelsNumber)
		},
		"publishEnergyGeneration": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			t0, _ := strconv.ParseUint(args[0], 10, 64)
			t1, _ := strconv.ParseUint(args[1], 10, 64)
			if len(args)%2 != 0 || len(args) > 2+len(EnergyTypes) {
				return shim.Error("Wrong number of arguments for function publishEnergyGeneration")
			}
			energyByTypeGeneratedKWH := make(map[string]float64)
			for i := 2; i < len(args); i += 2 {
				energyByTypeGeneratedKWH[args[i]], _ = strconv.ParseFloat(args[i+1], 64)
			}
			return chaincode.publishEnergyGeneration(stub, t0, t1, energyByTypeGeneratedKWH)
		},
		"registerSellBid": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			amountKWH, _ := strconv.ParseFloat(args[0], 64)
			pricePerKWH, _ := strconv.ParseFloat(args[1], 64)
			return chaincode.registerSellBid(stub, amountKWH, pricePerKWH, args[2])
		},
		"registerBuyBid": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			amountKWH, _ := strconv.ParseFloat(args[3], 64)
			pricePerKWH, _ := strconv.ParseFloat(args[4], 64)
			return chaincode.registerBuyBid(stub, args[0], args[1], args[2], amountKWH, pricePerKWH, args[5])
		},
		"validateBuyBid": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			maxBuyBidPaymentCover, _ := strconv.ParseFloat(args[1], 64)
			return chaincode.validateBuyBid(stub, args[0], maxBuyBidPaymentCover)
		},
		"getCallerIDAndCallerMspID": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			return chaincode.getCallerIDAndCallerMspID(stub)
		},
		"auction": func(stub shim.ChaincodeStubInterface, args []string) pb.Response { return chaincode.auction(stub) },
		"transactionsEnergyQuantityFromPaymentToken": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			return chaincode.transactionsEnergyQuantityFromPaymentToken(stub, args[0], args[1])
		},
		"getEnergyTransactionsFromPaymentToken": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			return chaincode.getEnergyTransactionsFromPaymentToken(stub, args[0], args[1])
		},
		"getEnergyTransactionsFromSellBidNumbers": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			return chaincode.getEnergyTransactionsFromSellBidNumbers(stub, args)
		},
		"getEnergyTransactionsFromFullSellBidKey": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			return chaincode.getEnergyTransactionsFromFullSellBidKey(stub, args[0], args[1], args[2])
		},
		"getEnergyTransactionsFromSellerAfterSellBidNumber": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			initialSellBidNum, _ := strconv.ParseUint(args[2], 10, 64)
			return chaincode.getEnergyTransactionsFromSellerAfterSellBidNumber(stub, args[0], args[1], initialSellBidNum)
		},

		////////// TESTING FUNCTIONS ////////////////
		"auctionSortedQueries": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			return chaincode.auctionSortedQueries(stub)
		},
		"registerMultipleSellBids": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			nSellBids, _ := strconv.ParseUint(args[0], 10, 64)
			minAmountKWH, _ := strconv.ParseFloat(args[1], 64)
			maxAmountKWH, _ := strconv.ParseFloat(args[2], 64)
			minPricePerKWH, _ := strconv.ParseFloat(args[3], 64)
			maxPricePerKWH, _ := strconv.ParseFloat(args[4], 64)
			return chaincode.registerMultipleSellBids(stub, nSellBids, minAmountKWH, maxAmountKWH, minPricePerKWH, maxPricePerKWH, args[5])
		},
		"registerMultipleBuyBids": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			nBuyBids, _ := strconv.Atoi(args[0])
			minAmountKWH, _ := strconv.ParseFloat(args[2], 64)
			maxAmountKWH, _ := strconv.ParseFloat(args[3], 64)
			minPricePerKWH, _ := strconv.ParseFloat(args[4], 64)
			maxPricePerKWH, _ := strconv.ParseFloat(args[5], 64)
			return chaincode.registerMultipleBuyBids(stub, nBuyBids, args[1], minAmountKWH, maxAmountKWH, minPricePerKWH, maxPricePerKWH, args[6])
		},
		"validateMultipleBuyBids": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			nBuyBids, _ := strconv.Atoi(args[0])
			return chaincode.validateMultipleBuyBids(stub, nBuyBids)
		},
		"clearSellBids": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			return chaincode.clearSellBids(stub)
		},
		"clearBuyBids": func(stub shim.ChaincodeStubInterface, args []string) pb.Response { return chaincode.clearBuyBids(stub) },
		"printDataQuantityByPartialCompositeKey": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			if len(args) < 2 {
				return chaincode.printDataQuantityByPartialCompositeKey(stub, args[0], []string{})
			}
			return chaincode.printDataQuantityByPartialCompositeKey(stub, args[0], args[1:len(args)])
		},
		"deleteDataByPartialCompositeKey": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			if len(args) < 2 {
				return chaincode.deleteDataByPartialCompositeKey(stub, args[0], []string{})
			}
			return chaincode.deleteDataByPartialCompositeKey(stub, args[0], args[1:len(args)])
		},
		"printDataQuantityByPartialSimpleKey": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			return chaincode.printDataQuantityByPartialSimpleKey(stub, args[0])
		},
		"deleteDataByPartialSimpleKey": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			return chaincode.deleteDataByPartialSimpleKey(stub, args[0])
		},
		"registerMultipleSellers": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			nSellers, _ := strconv.Atoi(args[0])
			return chaincode.registerMultipleSellers(stub, nSellers)
		},
		"measureTimeDifferentAuctions": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			repeatAuction, _ := strconv.Atoi(args[0])
			return chaincode.measureTimeDifferentAuctions(stub, repeatAuction)
		},
		"testWorldStateLogic": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			return chaincode.testWorldStateLogic(stub)
		},
		"sensorDeclareActiveTestContext": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			return chaincode.sensorDeclareActiveTestContext(stub, args[0])
		},
		"publishSensorDataTestContext": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			versionInt, _ := strconv.Atoi(args[1])
			version := int8(versionInt)
			unit64, _ := strconv.ParseUint(args[2], 10, 32)
			unit := uint32(unit64)
			timestamp, _ := strconv.ParseUint(args[3], 10, 64)
			value, _ := strconv.ParseFloat(args[4], 64)
			eUint, _ := strconv.ParseUint(args[5], 10, 8)
			e := uint8(eUint)
			confidenceUint, _ := strconv.ParseUint(args[6], 10, 8)
			confidence := uint8(confidenceUint)
			dev, _ := strconv.ParseUint(args[7], 10, 32)
			return chaincode.publishSensorDataTestContext(stub, args[0], version, unit, timestamp, value, e, confidence, uint32(dev))
		},
		"registerSellerTestContext": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			windTurbinesNumber, _ := strconv.ParseUint(args[1], 10, 64)
			solarPanelsNumber, _ := strconv.ParseUint(args[2], 10, 64)
			return chaincode.registerSellerTestContext(stub, args[0], windTurbinesNumber, solarPanelsNumber)
		},
		"publishEnergyGenerationTestContext": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			t0, _ := strconv.ParseUint(args[1], 10, 64)
			t1, _ := strconv.ParseUint(args[2], 10, 64)
			if len(args)%2 == 0 || len(args) > 3+len(EnergyTypes) {
				return shim.Error("Wrong number of arguments for function publishEnergyGenerationTestContext")
			}
			energyByTypeGeneratedKWH := make(map[string]float64)
			for i := 3; i < len(args); i += 2 {
				energyByTypeGeneratedKWH[args[i]], _ = strconv.ParseFloat(args[i+1], 64)
			}
			return chaincode.publishEnergyGenerationTestContext(stub, args[0], t0, t1, energyByTypeGeneratedKWH)
		},
		"registerSellBidTestContext": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			amountKWH, _ := strconv.ParseFloat(args[1], 64)
			pricePerKWH, _ := strconv.ParseFloat(args[2], 64)
			return chaincode.registerSellBidTestContext(stub, args[0], amountKWH, pricePerKWH, args[3])
		},
		"getEnergyTransactionsFromSellBidNumbersTestContext": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			return chaincode.getEnergyTransactionsFromSellBidNumbersTestContext(stub, args[0], args[1:len(args)])
		},
		"validateBuyBidTestContext": func(stub shim.ChaincodeStubInterface, args []string) pb.Response {
			return chaincode.validateBuyBidTestContext(stub, args[0], args[1])
		},
	}
}

func main() {
	chaincode := new(EnergyChaincode)

	averageFunctionTimes = make(map[string]*FunctionStats)

	initFuncMap(chaincode)

	// making lists to store the stats of each function, since go map does not allow multiple reads and writes
	// even on different keys
	for functionName := range functionMap {
		averageFunctionTimes[functionName] = &FunctionStats{0, 0}
	}

	channelAverageCalculator = make(chan *FunctionAndDuration)
	go recalculateFunctionAverageTime()

	err := shim.Start(chaincode)
	if err != nil {
		printf("Error starting Simple chaincode: %s", err)
	}
}

/////////////////////////////////// TESTING FUNCTIONS ///////////////////////////////////////////
func (chaincode *EnergyChaincode) registerMultipleSellers(stub shim.ChaincodeStubInterface, nSellers int) pb.Response {
	println("---- registerMultipleSellers function beggining ----")
	for i := 0; i < nSellers; i++ {
		chaincode.registerSeller(stub, "seller"+strconv.Itoa(i), "metermsp", "meter"+strconv.Itoa(i), 2, 2)
	}
	return shim.Success(nil)
}

func (chaincode *EnergyChaincode) registerSellBidWithIndex(stub shim.ChaincodeStubInterface, amountKWH float64, pricePerKWH float64, energyType string, sellBidNum uint64) pb.Response {
	println("---- registerSellBidWithIndex function beggining ----")

	//get seller MSPID and their ID
	mspIDSeller, err := cid.GetMSPID(stub)
	sellerID, err := cid.GetID(stub)

	sellBid := st.SellBid{
		//IsSellBid:         true, //delete later
		MspIDSeller:       mspIDSeller,
		SellerID:          sellerID,
		SellerBidNumber:   sellBidNum,
		EnergyQuantityKWH: amountKWH,
		PricePerKWH:       pricePerKWH,
		EnergyType:        energyType,
	}

	//generate compositekey for the sellbid
	lastBidIDStr := strconv.FormatUint(uint64(sellBidNum), 10)
	keySellBid, err := stub.CreateCompositeKey("SellBid", []string{mspIDSeller, sellerID, energyType, lastBidIDStr})

	sellBidBytes, err := proto.Marshal(&sellBid)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutState(keySellBid, sellBidBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func (chaincode *EnergyChaincode) registerMultipleSellBids(stub shim.ChaincodeStubInterface, nSellBids uint64, minAmountKWH float64, maxAmountKWH float64, minPricePerKWH float64, maxPricePerKWH float64, energyType string) pb.Response {
	println("---- registerMultipleSellBids function beggining ----")
	for i := uint64(0); i < nSellBids; i++ {
		randomAmountKWH := minAmountKWH + rand.Float64()*(maxAmountKWH-minAmountKWH)
		randomPricePerKWH := minPricePerKWH + rand.Float64()*(maxPricePerKWH-minPricePerKWH)
		response := chaincode.registerSellBidWithIndex(stub, randomAmountKWH, randomPricePerKWH, energyType, i)
		if response.Status == shim.ERROR {
			return response
		}
	}

	key, _ := stub.CreateCompositeKey("SellerInfo", []string{"UFSC", "eDUwOTo6Q049c2VsbGVyMS11ZnNjLE9VPWNsaWVudCtPVT11ZnNjLE89VUZTQyxMPUZsb3JpYW5vcG9saXMsU1Q9U0MsQz1CUjo6Q049cmNhLWNhLE9VPUZhYnJpYyxPPUh5cGVybGVkZ2VyLFNUPU5vcnRoIENhcm9saW5hLEM9VVM="})
	sellerInfoBytes, _ := stub.GetState(key)
	var sellerInfo st.SellerInfo
	proto.Unmarshal(sellerInfoBytes, &sellerInfo)
	sellerInfo.LastBidID = nSellBids - 1
	updateSellerInfo(stub, sellerInfo)
	return shim.Success(nil)
}

func (chaincode *EnergyChaincode) registerMultipleBuyBids(stub shim.ChaincodeStubInterface, nBuyBids int, mspIDPaymentCompany string, minAmountKWH float64, maxAmountKWH float64, minPricePerKWH float64, maxPricePerKWH float64, energyType string) pb.Response {
	println("---- registerMultipleBuyBids function beggining ----")
	for i := 0; i < nBuyBids; i++ {
		randomAmountKWH := minAmountKWH + rand.Float64()*(maxAmountKWH-minAmountKWH)
		randomPricePerKWH := minPricePerKWH + rand.Float64()*(maxPricePerKWH-minPricePerKWH)
		token := "tokentest" + strconv.Itoa(i)
		response := chaincode.registerBuyBid(stub, mspIDPaymentCompany, token, "utility", randomAmountKWH, randomPricePerKWH, energyType)
		if response.Status == shim.ERROR {
			return response
		}
	}
	return shim.Success(nil)
}

func (chaincode *EnergyChaincode) validateMultipleBuyBids(stub shim.ChaincodeStubInterface, nBuyBids int) pb.Response {
	println("---- validateMultipleBuyBids function beggining ----")
	for i := 0; i < nBuyBids; i++ {
		token := "tokentest" + strconv.Itoa(i)
		response := chaincode.validateBuyBid(stub, token, 1000000)
		if response.Status == shim.ERROR {
			return response
		}
	}
	return shim.Success(nil)
}

func (chaincode *EnergyChaincode) clearSellBids(stub shim.ChaincodeStubInterface) pb.Response {
	println("---- clearSellBids function beggining ----")
	//get SellBids
	queryString := fmt.Sprintf(`{"selector":{"issellbid":true}}`)
	println("Query string: " + queryString)
	sellBidsIterator, err := stub.GetQueryResult(queryString)
	if err != nil {
		return shim.Error(err.Error())
	}

	for sellBidsIterator.HasNext() {
		queryResult, err := sellBidsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		err = stub.DelState(queryResult.Key)
		if err != nil {
			return shim.Error(err.Error())
		}
	}
	return shim.Success(nil)

}

func (chaincode *EnergyChaincode) clearBuyBids(stub shim.ChaincodeStubInterface) pb.Response {
	println("---- clearBuyBids function beggining ----")
	//get VALIDATED BuyBids
	queryString := fmt.Sprintf(`{"selector":{"$or": [{"validated":true},{"validated":false}]}}`)
	println("Query string: " + queryString)
	buyBidsIterator, err := stub.GetQueryResult(queryString)
	if err != nil {
		return shim.Error(err.Error())
	}

	for buyBidsIterator.HasNext() {
		queryResult, err := buyBidsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		err = stub.DelState(queryResult.Key)
		if err != nil {
			return shim.Error(err.Error())
		}
	}

	return shim.Success(nil)
}

func (chaincode *EnergyChaincode) printDataQuantityByPartialCompositeKey(stub shim.ChaincodeStubInterface, objectType string, keys []string) pb.Response {
	println("---- printDataQuantityByPartialCompositeKey function beggining ----")

	dataIterator, err := stub.GetStateByPartialCompositeKey(objectType, keys)
	if err != nil {
		return shim.Error(err.Error())
	}

	dataQuantity := 0
	var messageAux anypb.Any
	for dataIterator.HasNext() {
		queryResult, err := dataIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		proto.Unmarshal(queryResult.Value, &messageAux)
		println(messageAux.String())
		dataQuantity++
	}

	return shim.Success([]byte("The number of states with key beginning with " + objectType + " is " + strconv.Itoa(dataQuantity)))
}

func (chaincode *EnergyChaincode) deleteDataByPartialCompositeKey(stub shim.ChaincodeStubInterface, objectType string, keys []string) pb.Response {
	println("---- deleteDataByPartialCompositeKey function beggining ----")

	dataIterator, err := stub.GetStateByPartialCompositeKey(objectType, keys)
	if err != nil {
		return shim.Error(err.Error())
	}

	dataQuantity := 0
	for dataIterator.HasNext() {
		queryResult, err := dataIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		err = stub.DelState(queryResult.Key)
		if err != nil {
			return shim.Error(err.Error())
		}
		dataQuantity++
	}

	return shim.Success([]byte("The number of states with key beginning with " + objectType + " deleted was " + strconv.Itoa(dataQuantity)))
}

func (chaincode *EnergyChaincode) printDataQuantityByPartialSimpleKey(stub shim.ChaincodeStubInterface, partialSimpleKey string) pb.Response {
	println("---- printDataQuantityByPartialSimpleKey function beggining ----")

	dataIterator, err := stub.GetStateByRange(partialSimpleKey, "")
	if err != nil {
		return shim.Error(err.Error())
	}

	dataQuantity := 0
	var messageAux anypb.Any
	for dataIterator.HasNext() {
		queryResult, err := dataIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		proto.Unmarshal(queryResult.Value, &messageAux)
		println(messageAux.String())
		dataQuantity++
	}

	return shim.Success([]byte("The number of states with key beginning with " + partialSimpleKey + " is " + strconv.Itoa(dataQuantity)))
}

func (chaincode *EnergyChaincode) deleteDataByPartialSimpleKey(stub shim.ChaincodeStubInterface, partialSimpleKey string) pb.Response {
	println("---- deleteDataByPartialSimpleKey function beggining ----")

	dataIterator, err := stub.GetStateByRange(partialSimpleKey, "")
	if err != nil {
		return shim.Error(err.Error())
	}

	dataQuantity := 0
	for dataIterator.HasNext() {
		queryResult, err := dataIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		err = stub.DelState(queryResult.Key)
		if err != nil {
			return shim.Error(err.Error())
		}
		dataQuantity++
	}

	return shim.Success([]byte("The number of states with key beginning with " + partialSimpleKey + " deleted was " + strconv.Itoa(dataQuantity)))
}

/////////////////////////////////// DATABASE TESTING FUNCTIONS ///////////////////////////////////////////

func getDatabaseType() string {
	return "goleveldb"
}

func (chaincode *EnergyChaincode) measureSpeedSmartDataQuery(stub shim.ChaincodeStubInterface, repeatQuery int, nearTrustedActiveSensors *[]st.ActiveSensor, t0 uint64, t1 uint64) pb.Response {
	println("---- measureSpeedSmartDataQuery function beggining ----")

	dbType := getDatabaseType()

	println("dbType: " + dbType)

	if dbType != "" {

		tStart := time.Now()

		var queryIterators []shim.StateQueryIteratorInterface
		for i := 0; i < repeatQuery; i++ {

			for _, nearTrustedActiveSensor := range *nearTrustedActiveSensors {
				startKey := "SmartData" + nearTrustedActiveSensor.MspID + nearTrustedActiveSensor.SensorID + getMaxUint64CharsStrTimestamp(t0)
				endKey := "SmartData" + nearTrustedActiveSensor.MspID + nearTrustedActiveSensor.SensorID + getMaxUint64CharsStrTimestamp(t1)
				queryIterator, err := stub.GetStateByRange(startKey, endKey)
				queryIterators = append(queryIterators, queryIterator)
				if err != nil {
					return shim.Error(err.Error())
				}
			}

		}
		duration := time.Since(tStart)

		numberOfSmartDataFetched := 0
		for _, queryIterator := range queryIterators {
			for queryIterator.HasNext() {
				_, _ = queryIterator.Next()
				numberOfSmartDataFetched++
			}
			queryIterator.Close()
		}
		printf("numberOfSmartDataFetched: %d\n", numberOfSmartDataFetched)

		printf("%s StateByRange QUERY TIME: %s for %d database queries\n", dbType, duration.String(), repeatQuery)

		if dbType == "CouchDB" {

			tStart := time.Now()

			for i := 0; i < repeatQuery; i++ {
				//assetsIDs := "["
				//for _, nearTrustedActiveSensor := range *nearTrustedActiveSensors {
				//	assetsIDs += `"` + nearTrustedActiveSensor.MspID + nearTrustedActiveSensor.SensorID + `",`
				//}
				//assetsIDs = assetsIDs[:len(assetsIDs)-1] + "]"
				//
				//queryString := fmt.Sprintf(`{"selector":{"timestamp":{"$gt": %d},"timestamp":{"$lt": %d},"assetid":{ "$in": %s }}}`, t0, t1, assetsIDs)
				//println("Query string: " + queryString)

				for _, nearTrustedActiveSensor := range *nearTrustedActiveSensors {
					assetID := nearTrustedActiveSensor.MspID + nearTrustedActiveSensor.SensorID
					queryString := fmt.Sprintf(`{"selector":{"timestamp":{"$gt": %d},"timestamp":{"$lt": %d},"assetid":"%s"}}`, t0, t1, assetID)

					_, err := stub.GetQueryResult(queryString)
					if err != nil {
						return shim.Error(err.Error())
					}
				}
			}

			duration := time.Since(tStart)

			printf("%s JSON QUERY TIME: %s for %d database queries\n", dbType, duration.String(), repeatQuery)
		}

	} else {
		return shim.Error("database type not supported")
	}

	//return shim.Success([]byte("The number of states with key beginning with " + objectType + " deleted was " + strconv.Itoa(dataQuantity)))
	return shim.Success(nil)
}

func (chaincode *EnergyChaincode) measureGetSellerInfoRelatedToSmartMeter(stub shim.ChaincodeStubInterface, repeatQuery int, meterMspID string, meterID string) (st.SellerInfo, error) {
	println("---- measureGetSellerInfoRelatedToSmartMeter function beggining ----")

	var sellerInfo st.SellerInfo
	var meterSeller st.MeterSeller
	var sellerInfoBytes []byte

	dbType := getDatabaseType()

	println("dbType: " + dbType)

	if dbType != "" {

		tStart := time.Now()

		for i := 0; i < repeatQuery; i++ {
			key, err := stub.CreateCompositeKey("MeterSeller", []string{meterMspID, meterID})
			meterSellerBytes, err := stub.GetState(key)
			if meterSellerBytes == nil {
				return sellerInfo, fmt.Errorf("No meter of MSP %s and ID %s", meterMspID, meterID)
			}
			err = proto.Unmarshal(meterSellerBytes, &meterSeller)

			key, err = stub.CreateCompositeKey("SellerInfo", []string{meterSeller.MspIDSeller, meterSeller.SellerID})
			sellerInfoBytes, err = stub.GetState(key)
			if sellerInfoBytes == nil {
				return sellerInfo, fmt.Errorf("No seller related to the meter of MSP %s and of Smart Meter ID %s", meterMspID, meterID)
			}

			err = proto.Unmarshal(sellerInfoBytes, &sellerInfo)

			if err != nil {
				return sellerInfo, err
			}
		}

		duration := time.Since(tStart)
		printf("%s getSellerInfoRelatedToSmartMeter with MeterSeller struct QUERY TIME: %s for %d database queries\n", dbType, duration.String(), repeatQuery)

		if dbType == "CouchDB" {
			tStart := time.Now()

			for i := 0; i < repeatQuery; i++ {
				queryString := fmt.Sprintf(`{"selector":{"mspsmartmeter":"%s","smartmeterid":"%s"}}`, meterMspID, meterID)
				queryIterator, err := stub.GetQueryResult(queryString)

				if err != nil {
					return sellerInfo, err
				}

				if queryIterator.HasNext() {
					queryResult, _ := queryIterator.Next()
					sellerInfoBytes = queryResult.Value
				} else {
					queryIterator.Close()
					return sellerInfo, fmt.Errorf("No seller related to the meter of MSP %s and of Smart Meter ID %s", meterMspID, meterID)
				}

				queryIterator.Close()
			}

			duration := time.Since(tStart)
			printf("%s getSellerInfoRelatedToSmartMeter with JSON QUERY TIME: %s for %d database queries\n", dbType, duration.String(), repeatQuery)
		}
	}

	return sellerInfo, nil
}

func (chaincode *EnergyChaincode) measureTimeDifferentAuctions(stub shim.ChaincodeStubInterface, repeatQuery int) pb.Response {
	println("---- measureTimeDifferentAuctions function beggining ----")

	dbType := getDatabaseType()

	println("dbType: " + dbType)

	if dbType != "" {

		tStart := time.Now()

		for i := 0; i < repeatQuery; i++ {
			response := chaincode.auction(stub)
			if response.Status == shim.ERROR {
				return response
			}
		}

		duration := time.Since(tStart)
		printf("%s auction with chaincode sorting TIME: %s for %d auctions\n", dbType, duration.String(), repeatQuery)

		if dbType == "CouchDB" {
			tStart := time.Now()

			for i := 0; i < repeatQuery; i++ {
				response := chaincode.auctionSortedQueries(stub)
				if response.Status == 500 {
					return response
				}
			}

			duration := time.Since(tStart)
			printf("%s auctionSortedQueries with CouchDB sorting TIME: %s for %d auctions\n", dbType, duration.String(), repeatQuery)
		}
	}
	return shim.Error("Time test only")
}

func (chaincode *EnergyChaincode) testWorldStateLogic(stub shim.ChaincodeStubInterface) pb.Response {
	key := "a"
	stub.PutState(key, []byte("TESTE"))
	testeBytes, err := stub.GetState(key)
	if err != nil {
		return shim.Error(err.Error())
	}

	println(testeBytes)
	err = stub.DelState(key)
	if err != nil {
		return shim.Error(err.Error())
	}

	testeBytes, err = stub.GetState(key)
	if testeBytes != nil {
		return shim.Error("key was not deleted yet")
	}

	return shim.Success(nil)
}

/////////////////////////////// TEST FUNCTIONS BYPASSING cid.GetID() to avoid generating thousands of x509 certificates ///////////////////////////////////////////////

func (chaincode *EnergyChaincode) sensorDeclareActiveTestContext(stub shim.ChaincodeStubInterface, sensorID string) pb.Response {
	println("---- sensorDeclareActiveTestContext function beggining ----")

	//check if caller is a sensor
	err := cid.AssertAttributeValue(stub, "energy.sensor", "true")
	if err != nil {
		return shim.Error(err.Error())
	}

	//get the SensorID and the MSP ID of the sensor to create the composite state key
	mspID, err := cid.GetMSPID(stub)

	//check if sensor is already in the database
	key, err := stub.CreateCompositeKey("ActiveSensor", []string{mspID, sensorID})
	isActive, err := stub.GetState(key)

	if err != nil {
		return shim.Error(err.Error())
	}

	println("Key: " + key)
	println(fmt.Sprintf("Key (hex): %x", key))
	println("State: " + string(isActive))

	if isActive == nil && err == nil {
		//setting sensor to ACTIVE
		//getting sensor coordinates and the influence radius from the certificate
		xCert, _, _ := cid.GetAttributeValue(stub, "energy.x")
		yCert, _, _ := cid.GetAttributeValue(stub, "energy.y")
		zCert, _, _ := cid.GetAttributeValue(stub, "energy.z")
		radiusCert, _, _ := cid.GetAttributeValue(stub, "energy.radius")

		x, err := strconv.Atoi(xCert)
		y, err := strconv.Atoi(yCert)
		z, err := strconv.Atoi(zCert)
		radius, err := strconv.ParseFloat(radiusCert, 64)

		//putting the info in the struct ActiveSensor
		activityData := st.ActiveSensor{
			MspID:    mspID,
			SensorID: sensorID,
			IsActive: true,
			X:        int32(x),
			Y:        int32(y),
			Z:        int32(z),
			Radius:   radius,
		}

		//the struct will be saved as a Marshalled json
		activityDataBytes, err := proto.Marshal(&activityData)
		err = stub.PutState(key, activityDataBytes)
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success(nil)
	}

	return shim.Error("SENSOR IS DISABLED or ALREADY ACTIVE!")

}

func (chaincode *EnergyChaincode) publishSensorDataTestContext(stub shim.ChaincodeStubInterface, sensorID string, version int8, unit uint32, timestamp uint64, value float64, e uint8, confidence uint8, dev uint32) pb.Response {
	println("---- publishSensorDataTestContext function beggining ----")

	//verify if data is still valid based on timestamp
	currentTime := uint64(time.Now().Unix())
	if currentTime > timestamp+acceptedDelay {
		return shim.Error("The data timestamp is too old!")
	}

	//check if caller is a sensor
	err := cid.AssertAttributeValue(stub, "energy.sensor", "true")
	if err != nil {
		return shim.Error(err.Error())
	}

	var activityData st.ActiveSensor

	//get sensor id from the certificate
	mspID, err := cid.GetMSPID(stub)
	assetID := mspID + sensorID

	//verify if sensor is active
	key, err := stub.CreateCompositeKey("ActiveSensor", []string{mspID, sensorID})
	activityDataBytes, err := stub.GetState(key)
	err = proto.Unmarshal(activityDataBytes, &activityData)
	if err != nil || activityData.IsActive != true {
		return shim.Error("SENSOR IS NOT ACTIVE!")
	}

	//xCert, _, _ := cid.GetAttributeValue(stub, "energy.x")
	//yCert, _, _ := cid.GetAttributeValue(stub, "energy.y")
	//zCert, _, _ := cid.GetAttributeValue(stub, "energy.z")

	//x, err := strconv.Atoi(xCert)
	//y, err := strconv.Atoi(yCert)
	//z, err := strconv.Atoi(zCert)

	asset := st.SmartData{
		AssetID:    assetID,
		Version:    int32(version), //because of proto
		Unit:       unit,
		Timestamp:  timestamp,
		Value:      value,
		Error:      uint32(e),          //because of proto
		Confidence: uint32(confidence), //because of proto
		//X:          x,
		//Y:          y,
		//Z:          z,
		Dev: dev,
	}

	assetJSON, err := proto.Marshal(&asset)
	//key, err = stub.CreateCompositeKey(objectType, []string{mspID, sensorID, strconv.FormatUint(timestamp, 10)})
	//we do not use CompositeKey, because CompositeKeys are not supported for the method shim.ChaincodeStubInterface.GetStateByRange()
	key = "SmartData" + assetID + getMaxUint64CharsStrTimestamp(timestamp)
	stub.PutState(key, assetJSON)
	return shim.Success(nil)
}

func (chaincode *EnergyChaincode) registerSellerTestContext(stub shim.ChaincodeStubInterface, sellerID string, windTurbinesNumber uint64, solarPanelsNumber uint64) pb.Response {
	println("---- registerSellerTestContext function beggining ----")

	//test if timestamp is recent enough comparing with the transaction creation timestamp
	timestampStruct, err := stub.GetTxTimestamp()
	if err != nil {
		return shim.Error(err.Error())
	}
	if uint64(time.Now().Unix()-timestampStruct.Seconds) > acceptedDelay {
		return shim.Error("current timestamp provided is too old!")
	}

	//get seller MSPID and their ID
	mspIDSeller, err := cid.GetMSPID(stub)

	//generate compositekey
	key, err := stub.CreateCompositeKey("SellerInfo", []string{mspIDSeller, sellerID})
	sellerInfoBytes, err := stub.GetState(key)

	//check if seller is not already registered
	if sellerInfoBytes != nil {
		return shim.Error("Seller is ALREADY registered!")
	}

	energyToSellByType := make(map[string]float64)
	energyToSellByType["solar"] = 0.0

	sellerInfo := st.SellerInfo{
		MspIDSeller:             mspIDSeller,
		SellerID:                sellerID,
		WindTurbinesNumber:      windTurbinesNumber,
		SolarPanelsNumber:       solarPanelsNumber,
		EnergyToSellByType:      energyToSellByType,
		LastGenerationTimestamp: uint64(timestampStruct.Seconds), //REACTIVATE THIS LINE!!!
		//LastGenerationTimestamp: 0, //DELETE THIS LINE!!!
		//CoinBalance:             0,
		LastBidID: 0,
	}

	//save seller info to the World State
	sellerInfoBytes, err = proto.Marshal(&sellerInfo)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutState(key, sellerInfoBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func (chaincode *EnergyChaincode) publishEnergyGenerationTestContext(stub shim.ChaincodeStubInterface, sellerID string, t0 uint64, t1 uint64, energyByTypeGeneratedKWH map[string]float64) pb.Response {
	println("---- publishEnergyGenerationTestContext function beggining ----")

	//test if t0 is greater than t1
	if t0 >= t1 {
		return shim.Error("t1 MUST be greater than t0")
	}

	//test if t1 is greater or equal NOW
	currentTimestamp := uint64(time.Now().Unix())
	printf("t1: %d\n", t1)
	printf("Current timestamp: %d\n", currentTimestamp)
	if t1 > currentTimestamp+acceptedClockDrift {
		return shim.Error(fmt.Sprintf("t1: %d MUST be less or equal than the CURRENT TIME %d + the accepted clock drift of %ds", t1, currentTimestamp, acceptedClockDrift))
	}

	//get Meter MSP and MeterID
	sellerMspID, err := cid.GetMSPID(stub)

	println("Seller MSP ID: " + sellerMspID)
	println("Seller ID: " + sellerID)

	//get SellerInfo
	sellerInfo, err := getSellerInfo(stub, sellerMspID, sellerID)

	if err != nil {
		return shim.Error(err.Error())
	}
	printf("sellerInfo: %+v\n", sellerInfo)

	//test seller is not trying to generate energy twice for the same time interval
	if t0 < sellerInfo.LastGenerationTimestamp || t1 < sellerInfo.LastGenerationTimestamp {
		return shim.Error("Seller already registered energy generation for this time interval")
	}

	//test if energy generated is positive
	for _, energyGeneratedKWH := range energyByTypeGeneratedKWH {
		if energyGeneratedKWH <= 0 {
			return shim.Error("The energy generated MUST be greater than 0")
		}
	}

	//get Meter Location
	xCert, _, _ := cid.GetAttributeValue(stub, "energy.x")
	yCert, _, _ := cid.GetAttributeValue(stub, "energy.y")
	zCert, _, _ := cid.GetAttributeValue(stub, "energy.z")

	x, err := strconv.Atoi(xCert)
	y, err := strconv.Atoi(yCert)
	z, err := strconv.Atoi(zCert)
	println(x)
	println(y)
	println(z)

	//get Active Sensors
	_, activeSensorsDataList, err := getActiveSensorsList(stub, "")

	printf("Active Sensors Data List %+v\n", activeSensorsDataList)
	//definir criterios de aceitacao. EX: 3 organizacoes precisam ter sensores a distancia X
	//usar X, Y e Z para calcular a distancia
	var nearActiveSensorsList []st.ActiveSensor

	for _, activeSensorData := range activeSensorsDataList {
		distanceBetweenSensorAndGenerator := math.Sqrt(math.Pow(float64(activeSensorData.X-int32(x)), 2) + math.Pow(float64(activeSensorData.Y-int32(y)), 2))
		printf("distanceBetweenSensorAndGenerator: %f\n", distanceBetweenSensorAndGenerator)
		printf("activeSensorData.Radius: %f\n", activeSensorData.Radius)
		if distanceBetweenSensorAndGenerator <= activeSensorData.Radius {
			nearActiveSensorsList = append(nearActiveSensorsList, activeSensorData)
		}
	}

	printf("nearActiveSensorsList: %+v\n", nearActiveSensorsList)
	//chamar alguma funcao que puxe do banco de dados os criterios de cada validador (EXECUTAR DE ACORDO COM A ORGANIZACAO A QUAL O PEER EXECUTANDO PERTENCE)
	peerOrg, err := GetMSPID()
	println("Peer executing MSP: " + peerOrg)

	//mspTrustedSensors, _, err := getMspTrustedSensorsMap(stub, peerOrg)
	allMspsTrustedSensorsMaps, _ := getAllMspsTrustedSensorsMaps(stub)
	mspTrustedSensors := allMspsTrustedSensorsMaps[peerOrg]

	printf("mspTrustedSensors: %+v\n", mspTrustedSensors)

	var nearTrustedActiveSensors []st.ActiveSensor

	/*for _, nearActiveSensor := range nearActiveSensorsList {
		printf("nearActiveSensor: %+v\n Trusted: %t\n", nearActiveSensor, mspTrustedSensors[nearActiveSensor.MspID+nearActiveSensor.SensorID])
		if mspTrustedSensors[nearActiveSensor.MspID+nearActiveSensor.SensorID] {
			nearTrustedActiveSensors = append(nearTrustedActiveSensors, nearActiveSensor)
		}
	}*/

	nearTrustedActiveSensors = nearActiveSensorsList

	printf("nearTrustedActiveSensors: %+v\n", nearTrustedActiveSensors)

	//timeTestFunction
	//t.measureSpeedSmartDataQuery(stub, 100, &nearTrustedActiveSensors, t0, t1)

	//get nearTrustedActiveSensors smart data in time interval [t0,t1]
	var nearTrustedSensorsSmartData []st.SmartData
	nearTrustedSensorsSmartData, err = getSmartDataBySensorsInInterval(stub, &nearTrustedActiveSensors, t0, t1)
	if err != nil {
		return shim.Error(err.Error())
	}

	//check if energy could have been generated considering the nearTrustedActiveSensors published data
	// JUST TEMPORARY! CHECK POLICIES!
	var maxPossibleGeneratedEnergy float64
	for energyType, energyGeneratedKWH := range energyByTypeGeneratedKWH {

		switch energyType {
		case "solar":
			maxPossibleGeneratedEnergy = getMaxPossibleGeneratedSolarEnergyInInterval(stub, &nearTrustedSensorsSmartData, sellerInfo.SolarPanelsNumber, t0, t1)
		case "wind":
			maxPossibleGeneratedEnergy = getMaxPossibleGeneratedWindEnergyInInterval(stub, &nearTrustedSensorsSmartData, sellerInfo.WindTurbinesNumber, t0, t1)
		case "tidal":
			return shim.Error("Not implemented YET!")
		case "hydro":
			return shim.Error("Not implemented YET!")
		case "geothermal":
			return shim.Error("Not implemented YET!")
		default:
			return shim.Error(energyType + " is an INVALID energy type!")

		}

		if energyGeneratedKWH > maxPossibleGeneratedEnergy {
			return shim.Error("Network ruled the alleged generation of " + energyType + " energy INVALID!")
		}
		sellerInfo.EnergyToSellByType[energyType] += energyGeneratedKWH

	}

	sellerInfo.LastGenerationTimestamp = uint64(t1)
	err = updateSellerInfo(stub, sellerInfo)
	if err != nil {
		return shim.Error(err.Error())
	}

	printf("Energy available for selling by type: %+v\n", energyByTypeGeneratedKWH)

	successMessage := fmt.Sprintf("%+v\n successfully available for selling by the seller %s%s with the smart meter of ID: %s%s", energyByTypeGeneratedKWH, sellerInfo.MspIDSeller, sellerInfo.SellerID, sellerInfo.MspIDSmartMeter, sellerInfo.SmartMeterID)

	return shim.SuccessWithPriorityBypassPhantomReadCheck([]byte(successMessage), pb.Priority_MEDIUM)
	//return shim.Error("Testing time")
}

func (chaincode *EnergyChaincode) registerSellBidTestContext(stub shim.ChaincodeStubInterface, sellerID string, quantityKWH float64, pricePerKWH float64, energyType string) pb.Response {
	println("---- registerSellBidTestContext function beggining ----")

	var sellerInfo st.SellerInfo

	//only sellers can execute this function
	err := cid.AssertAttributeValue(stub, "energy.seller", "true")
	if err != nil {
		return shim.Error(err.Error())
	}

	//get seller MSPID and their ID
	mspIDSeller, err := cid.GetMSPID(stub)

	//get SellerInfo from the seller
	keySellerInfo, err := stub.CreateCompositeKey("SellerInfo", []string{mspIDSeller, sellerID})
	sellerInfoBytes, err := stub.GetState(keySellerInfo)

	//check if seller is registered
	if sellerInfoBytes == nil || err != nil {
		return shim.Error("Seller COULD NOT be fetched from the ledger!")
	}

	err = proto.Unmarshal(sellerInfoBytes, &sellerInfo)
	if err != nil {
		return shim.Error(err.Error())
	}

	//check if seller has already produced the amount of energy type
	printf("sellerInfo.EnergyToSellByType['%s']: %f\n", energyType, sellerInfo.EnergyToSellByType[energyType])
	printf("amountKWH: %f\n", quantityKWH)
	if sellerInfo.EnergyToSellByType[energyType] < quantityKWH {
		return shim.Error("Seller does not have the indicated amount of " + energyType + " energy to sell!")
	}

	//subtract energy to be sold from SellerInfo
	sellerInfo.EnergyToSellByType[energyType] -= quantityKWH
	//update lastbid
	sellerInfo.LastBidID++
	//store updated SellerInfo
	err = updateSellerInfo(stub, sellerInfo)
	if err != nil {
		return shim.Error(err.Error())
	}

	sellBid := st.SellBid{
		//IsSellBid:       true, //delete later
		MspIDSeller:       mspIDSeller,
		SellerID:          sellerID,
		SellerBidNumber:   sellerInfo.LastBidID,
		EnergyQuantityKWH: quantityKWH,
		PricePerKWH:       pricePerKWH,
		EnergyType:        energyType,
	}

	//generate compositekey for the sellbid
	lastBidIDStr := strconv.FormatUint(uint64(sellerInfo.LastBidID), 10)
	keySellBid, err := stub.CreateCompositeKey("SellBid", []string{mspIDSeller, sellerID, lastBidIDStr})

	sellBidBytes, err := proto.Marshal(&sellBid)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutState(keySellBid, sellBidBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func getSellerInfo(stub shim.ChaincodeStubInterface, sellerMspID string, sellerID string) (st.SellerInfo, error) {
	println("---- getSellerInfo function beggining ----")

	var sellerInfo st.SellerInfo
	var sellerInfoBytes []byte

	key, err := stub.CreateCompositeKey("SellerInfo", []string{sellerMspID, sellerID})
	sellerInfoBytes, err = stub.GetState(key)
	if err != nil {
		return sellerInfo, err
	}

	err = proto.Unmarshal(sellerInfoBytes, &sellerInfo)

	if err != nil {
		return sellerInfo, err
	}

	return sellerInfo, nil
}

func (chaincode *EnergyChaincode) getEnergyTransactionsFromSellBidNumbersTestContext(stub shim.ChaincodeStubInterface, sellerID string, sellBidNumbers []string) pb.Response {
	println("---- getEnergyTransactionsFromSellBidNumbersTestContext function beggining ----")

	err := cid.AssertAttributeValue(stub, "energy.seller", "true")
	if err != nil {
		return shim.Error(err.Error())
	}

	mspIDSeller, err := cid.GetMSPID(stub)

	energyTransactionsJSON := ""
	for _, sellBidNumber := range sellBidNumbers {
		pbResponse := chaincode.getEnergyTransactionsFromFullSellBidKey(stub, mspIDSeller, sellerID, sellBidNumber)
		if pbResponse.Status == shim.OK {
			energyTransactionsJSONSellBid := string(pbResponse.GetPayload())
			energyTransactionsJSON += energyTransactionsJSONSellBid + ","
		}
	}
	if len(energyTransactionsJSON) > 0 {
		energyTransactionsJSON = "[" + energyTransactionsJSON[:len(energyTransactionsJSON)-1] + "]"
	} else {
		energyTransactionsJSON = "[" + "]"
	}

	return shim.Success([]byte(energyTransactionsJSON))
}

func (chaincode *EnergyChaincode) validateBuyBidTestContext(stub shim.ChaincodeStubInterface, paymentCompanyMspID string, token string) pb.Response {
	println("---- validateBuyBidTestContext function beggining ----")

	var buyBid st.BuyBid

	key, err := stub.CreateCompositeKey("BuyBid", []string{"false", paymentCompanyMspID, token})
	buyBidBytes, err := stub.GetState(key)
	if buyBidBytes == nil {
		return shim.Error("Error retriving BuyBid of token " + token)
	}

	err = proto.Unmarshal(buyBidBytes, &buyBid)
	if err != nil {
		return shim.Error("Error unmarshaling BuyBid of token " + token)
	}

	if buyBid.Token != token {
		return shim.Error("Argument 'token' does not match BuyBid Token")
	}

	buyBid.Validated = true
	buyBidBytes, err = proto.Marshal(&buyBid)
	if err != nil {
		return shim.Error(err.Error())
	}

	//delete BuyBid with the composite key "BuyBidfalse..."
	err = stub.DelState(key)
	if err != nil {
		return shim.Error(err.Error())
	}

	//... and replace with the composite key "BuyBidtrue..."
	key, err = stub.CreateCompositeKey("BuyBid", []string{"true", paymentCompanyMspID, token})

	err = stub.PutState(key, buyBidBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	//send it to auction
	printf("Validated BuyBid: %+v\n", buyBid)

	return shim.Success([]byte("BuyBid of token " + token + " validated!"))
}

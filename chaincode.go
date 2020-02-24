package main

import (
	"os/exec"
	"sync"

	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"

//This packages are for the code that reads the ovs-db - meaning the readLine fucntion
//	"log"
	"bufio"
	"io"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

type flow struct {
	ObjectType	string `json:"docType"` //docType is used to distinguish the various types of objects in state database
	InPort		string `json:"in_port"`    //the fieldtags are needed to keep case from bouncing around
	Actions		string `json:"actions"`
	Priority	int    `json:"priority"`
	NetworkSourceIP string `json:"nw_src"`
	Cookie          string `json:"cookie"`	//normally this field requires some hex data type, because it can have only hex values
	Duration	int    `json:"duration"`
	Table		int    `json:"table"`
	N_packets	int    `json:"n_packets"`
	N_bytes		int    `json:"n_bytes"`
	Idle_age	int    `json:"idle_age"`
}

// This function is mine
// It was added for the reason that i wanted somehow external commands to be executed outside the source code when a branching condition was met
func exe_cmd(cmd string, wg *sync.WaitGroup) {
    fmt.Println(cmd)
    out, err := exec.Command("sh", "-c", cmd).Output()
    if err != nil {
        fmt.Println("failed !!!!!!! with %s\n", err)
    }
    fmt.Printf("%s", out)
    wg.Done()
}

//This function is from https://gist.github.com/jgfrancisco/6610078040b15b7e9611 and we need it in order to read ovs-db
func readLine(reader *bufio.Reader) (strLine string, err error) {
        buffer := new(bytes.Buffer)
        for {
                var line []byte
                var isPrefix bool
                line, isPrefix, err = reader.ReadLine()
        //      log.Printf("[DEBUG] Read Len: %d, isPrefix: %t, Error: %v\n", len(line), isPrefix, err)

                if err != nil && err != io.EOF {
                        return "", err
                }

                buffer.Write(line)

                if !isPrefix {
        //              log.Println("[INFO] EOL found")
                        break
                }
        }
//      log.Println("[DEBUG] End of line")
        return buffer.String(), err
}


// ===================================================================================
// Main
// ===================================================================================
func main() {
	fmt.Println("Chaincode is now starting@@")
	err := shim.Start(new(SimpleChaincode))
	fmt.Println("Chaincode has been started@@")

	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// Init initializes chaincode
// ===========================
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {

	fmt.Println("Init is about to retun here")
	return shim.Success(nil)
}

// Invoke - Our entry point for Invocations
// ========================================
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "initFlow" { //create a new flow
		return t.initFlow(stub, args)
	} else if function == "transferFlow" { //change nw_src of a specific flow
		return t.transferFlow(stub, args)
	} else if function == "transferFlowsBasedOnActions" { //transfer all flows of a certain actions
		return t.transferFlowsBasedOnActions(stub, args)
	} else if function == "delete" { //delete a flow
		return t.delete(stub, args)
	} else if function == "readFlow" { //read a flow
		return t.readFlow(stub, args)
	} else if function == "queryFlowsByNetworkSourceIP" { //find flows for nw_src X using rich query
		return t.queryFlowsByNetworkSourceIP(stub, args)
	} else if function == "queryFlows" { //find flows based on an ad hoc rich query
		return t.queryFlows(stub, args)
	} else if function == "getFlowsByRange" { //get flows based on range query
		return t.getFlowsByRange(stub, args)
	}

//the fmt.Println call output is printed in the docker container of the cc logs (docker logs net-peer*)
	fmt.Println("invoke did not find func: " + function) //error
//the shim.Error call output is printed in the stdout when we invoke a function from the Invokefunction packets.
	return shim.Error("Received unknown function invocation")
}

// ============================================================
// initFlow - create a new flow, store into chaincode state
// ============================================================
func (t *SimpleChaincode) initFlow(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error

	if len(args) != 10 {
		return shim.Error("Incorrect number of arguments. Expecting 10")
	}

	// ==== Input sanitation ====
	fmt.Println("- start init flow")
	if len(args[0]) <= 0 {
		return shim.Error("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return shim.Error("2nd argument must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return shim.Error("3rd argument must be a non-empty string")
	}
	if len(args[3]) <= 0 {
		return shim.Error("4th argument must be a non-empty string")
	}
	if len(args[4]) <= 0 {
		return shim.Error("5th argument must be a possitive integer")
	}
	if len(args[5]) < 0 {
		return shim.Error("6th argument must be a possitive integer")
	}
	if len(args[6]) < 0 {
		return shim.Error("7th argument must be a possitive integer")
	}
	if len(args[7]) < 0 {
		return shim.Error("8th argument must be a possitive integer")
	}
	if len(args[8]) <= 0 {
		return shim.Error("9th argument must be a possitive integer")
	}
	if len(args[9]) <= 0 {
		return shim.Error("10th argument must be a non-empty string")
	}

	flowInPort := args[0]
	actions := strings.ToLower(args[1])
	nw_src := strings.ToLower(args[3])
	priority, err := strconv.Atoi(args[2])
	cookie := strings.ToLower(args[4])
	duration, err := strconv.Atoi(args[5])
	table, err:= strconv.Atoi(args[6])
	n_packets, err:= strconv.Atoi(args[7])
	n_bytes, err:= strconv.Atoi(args[8])
	idle_age, err:= strconv.Atoi(args[9])

	if err != nil {
		return shim.Error("3rd argument must be a numeric string")
	}

	// ==== Check if flow already exists ====
	flowAsBytes, err := stub.GetState(flowInPort)
	if err != nil {
		return shim.Error("Failed to get flow: " + err.Error())
	} else if flowAsBytes != nil {
		fmt.Println("This flow already exists: " + flowInPort)
		return shim.Error("This flow already exists: " + flowInPort)
	}

	// ==== Create flow object and marshal to JSON ====
	objectType := "flow"

// when executing ./install.sh this line causes an error, we have added some more literals that need to be added here too.
	flow := &flow{objectType, flowInPort, actions, priority, nw_src, cookie, duration, table, n_packets, n_bytes, idle_age}


	flowJSONasBytes, err := json.Marshal(flow)
	if err != nil {
		return shim.Error(err.Error())
	}

	// === Save flow to state ===
	err = stub.PutState(flowInPort, flowJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	indexInPort := "actions~in_port"
	actionsInPortIndexKey, err := stub.CreateCompositeKey(indexInPort, []string{flow.Actions, flow.InPort})
	if err != nil {
		return shim.Error(err.Error())
	}
	//  Save index entry to state. Only the key in_port is needed, no need to store a duplicate copy of the flow.
	//  Note - passing a 'nil' value will effectively delete the key from state, therefore we pass null character as value
	value := []byte{0x00}
	stub.PutState(actionsInPortIndexKey, value)

	// ==== Flow saved and indexed. Return success ====
	fmt.Println("- end init flow")
	return shim.Success(nil)
}

// ===============================================
// readFlow - read a flow from chaincode state
// ===============================================
func (t *SimpleChaincode) readFlow(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var in_port, jsonResp string
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting in_port of the flow to query")
	}

	in_port = args[0]
	valAsbytes, err := stub.GetState(in_port) //get the flow from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + in_port + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Flow does not exist: " + in_port + "\"}"
		return shim.Error(jsonResp)
	}

	return shim.Success(valAsbytes)
}

// ==================================================
// delete - remove a flow key/value pair from state
// ==================================================
func (t *SimpleChaincode) delete(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var jsonResp string
	var flowJSON flow
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}
	flowInPort := args[0]

	// to maintain the actions~in_port index, we need to read the flow first and get its actions
	valAsbytes, err := stub.GetState(flowInPort) //get the flow from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + flowInPort + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Flow does not exist: " + flowInPort + "\"}"
		return shim.Error(jsonResp)
	}

	err = json.Unmarshal([]byte(valAsbytes), &flowJSON)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to decode JSON of: " + flowInPort + "\"}"
		return shim.Error(jsonResp)
	}

	err = stub.DelState(flowInPort) //remove the flow from chaincode state
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}

	// maintain the index
	indexInPort := "actions~in_port"
	actionsInPortIndexKey, err := stub.CreateCompositeKey(indexInPort, []string{flowJSON.Actions, flowJSON.InPort})
	if err != nil {
		return shim.Error(err.Error())
	}

	//  Delete index entry to state.
	err = stub.DelState(actionsInPortIndexKey)
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}
	return shim.Success(nil)
}

// ===========================================================
// transfer a flow by setting a new nw_src in_port on the flow
// ===========================================================
func (t *SimpleChaincode) transferFlow(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	flowInPort := args[0]
	newNetworkSourceIP := strings.ToLower(args[1])
	fmt.Println("- start transferFlow ", flowInPort, newNetworkSourceIP)

	flowAsBytes, err := stub.GetState(flowInPort)
	if err != nil {
		return shim.Error("Failed to get flow:" + err.Error())
	} else if flowAsBytes == nil {
		return shim.Error("Flow does not exist")
	}

	flowToTransfer := flow{}
	err = json.Unmarshal(flowAsBytes, &flowToTransfer) //unmarshal it aka JSON.parse()
	if err != nil {
		return shim.Error(err.Error())
	}
	flowToTransfer.NetworkSourceIP = newNetworkSourceIP //change the nw_src

	flowJSONasBytes, _ := json.Marshal(flowToTransfer)
	err = stub.PutState(flowInPort, flowJSONasBytes) //rewrite the flow
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Println("- end transferFlow (success)")
	return shim.Success(nil)
}

// ===========================================================================================
// constructQueryResponseFromIterator constructs a JSON array containing query results from
// a given result iterator
// ===========================================================================================
func constructQueryResponseFromIterator(resultsIterator shim.StateQueryIteratorInterface) (*bytes.Buffer, error) {
	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	return &buffer, nil
}

func (t *SimpleChaincode) getFlowsByRange(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	startKey := args[0]
	endKey := args[1]

	resultsIterator, err := stub.GetStateByRange(startKey, endKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	buffer, err := constructQueryResponseFromIterator(resultsIterator)
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Printf("- getFlowsByRange queryResult:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

func (t *SimpleChaincode) transferFlowsBasedOnActions(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	actions := args[0]
	newNetworkSourceIP := strings.ToLower(args[1])
	fmt.Println("- start transferFlowsBasedOnActions ", actions, newNetworkSourceIP)

	// Query the actions~in_port index by actions
	// This will execute a key range query on all keys starting with 'actions'
	actionsedFlowResultsIterator, err := stub.GetStateByPartialCompositeKey("actions~in_port", []string{actions})
	if err != nil {
		return shim.Error(err.Error())
	}
	defer actionsedFlowResultsIterator.Close()

	// Iterate through result set and for each flow found, transfer to newNetworkSourceIP
	var i int
	for i = 0; actionsedFlowResultsIterator.HasNext(); i++ {
		// Note that we don't get the value (2nd return variable), we'll just get the flow in_port from the composite key
		responseRange, err := actionsedFlowResultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		// get the actions and in_port from actions~in_port composite key
		objectType, compositeKeyParts, err := stub.SplitCompositeKey(responseRange.Key)
		if err != nil {
			return shim.Error(err.Error())
		}
		returnedActions := compositeKeyParts[0]
		returnedFlowInPort := compositeKeyParts[1]
		fmt.Printf("- found a flow from index:%s actions:%s in_port:%s\n", objectType, returnedActions, returnedFlowInPort)

		// Now call the transfer function for the found flow.
		// Re-use the same function that is used to transfer individual flows
		response := t.transferFlow(stub, []string{returnedFlowInPort, newNetworkSourceIP})
		// if the transfer failed break out of loop and return error
		if response.Status != shim.OK {
			return shim.Error("Transfer failed: " + response.Message)
		}
	}

	responsePayload := fmt.Sprintf("Transferred %d %s flows to %s", i, actions, newNetworkSourceIP)
	fmt.Println("- end transferFlowsBasedOnActions: " + responsePayload)
	return shim.Success([]byte(responsePayload))
}

func (t *SimpleChaincode) queryFlowsByNetworkSourceIP(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	nw_src := strings.ToLower(args[0])

	queryString := fmt.Sprintf("{\"selector\":{\"docType\":\"flow\",\"nw_src\":\"%s\"}}", nw_src)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

func (t *SimpleChaincode) queryFlows(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	queryString := args[0]

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

func getQueryResultForQueryString(stub shim.ChaincodeStubInterface, queryString string) ([]byte, error) {

	fmt.Printf("- getQueryResultForQueryString queryString:\n%s\n", queryString)

	resultsIterator, err := stub.GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	buffer, err := constructQueryResponseFromIterator(resultsIterator)
	if err != nil {
		return nil, err
	}

	fmt.Printf("- getQueryResultForQueryString queryResult:\n%s\n", buffer.String())

	return buffer.Bytes(), nil
}

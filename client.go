package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/azyablov/srljrpc"
	"github.com/azyablov/srljrpc/apierr"
	"github.com/azyablov/srljrpc/formats"
	"github.com/azyablov/srljrpc/yms"
)

var (
	host   = "clab-evpn-leaf1"
	user   = "admin"
	pass   = "NokiaSrl1!"
	port   = 443
	hostOC = "clab-evpn-spine3"
)

func main() {
	// Create a new JSON RPC client with credentials and port (used 443 as default for the sake of demo).
	c, err := srljrpc.NewJSONRPCClient(&host, srljrpc.WithOptCredentials(&user, &pass), srljrpc.WithOptPort(&port))
	if err != nil {
		panic(err)
	}
	fmt.Printf("Target hostname: %s\nTarget system version: %s\n", c.GetHostname(), c.GetSysVer())

	// GET method example.
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("Get() example:")
	fmt.Println(strings.Repeat("=", 80))

	getResp, err := c.Get(`/network-instance[name="MAC-VRF 1"]`, `/system/lldp`)
	if err != nil {
		panic(err)
	}
	rStr, err := json.MarshalIndent(getResp.Result, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Response: %s\n", string(rStr))

	// Getting stats.
	fmt.Println("State() example:")
	fmt.Println(strings.Repeat("=", 80))

	stateResp, err := c.State("/system/json-rpc-server")
	if err != nil {
		panic(err)
	}
	outHelper(stateResp.Result)

	// Updating/Replacing/Deleting config
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("Update()/Delete()/Replace() example:")
	fmt.Println(strings.Repeat("=", 80))

	pvs := []srljrpc.PV{
		{Path: `/interface[name=ethernet-1/51]/subinterface[index=0]/description`, Value: "UPDATE"},
		{Path: `/system/banner`, Value: "DELETE"},
		{Path: `/interface[name=mgmt0]/description`, Value: "REPLACE"},
	}
	// Getting existing config for the sake of demo.
	for _, pv := range pvs {
		getResp, err := c.Get(pv.Path)
		if err != nil {
			panic(err)
		}
		outHelper(getResp.Result)
	}

	mdmResp, err := c.Update(0, pvs[0]) // setting 0 as confirmation timeout to apply changes immediately.
	if err != nil {
		panic(err)
	}
	outHelper(mdmResp.Result)
	mdmResp, err = c.Delete(0, pvs[1].Path) // setting 0 as confirmation timeout to apply changes immediately.
	if err != nil {
		panic(err)
	}
	outHelper(mdmResp.Result)
	mdmResp, err = c.Replace(0, pvs[2]) // setting 0 as confirmation timeout to apply changes immediately.
	if err != nil {
		panic(err)
	}
	outHelper(mdmResp.Result)

	// CLI with different formats: JSON and TABLE.
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("CLI() example:")
	fmt.Println(strings.Repeat("=", 80))
	cliResp, err := c.CLI([]string{"show version", "show network-instance summary"}, formats.JSON)
	if err != nil {
		panic(err)
	}
	outHelper(cliResp.Result)

	cliResp, err = c.CLI([]string{"show system lldp neighbor"}, formats.TABLE)
	if err != nil {
		panic(err)
	}
	type Table []string
	var t Table
	b, _ := cliResp.Result.MarshalJSON()
	err = json.Unmarshal(b, &t)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", t[0])

	// Tools usage example to clear interface counters.
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("Tools() example:")
	fmt.Println(strings.Repeat("=", 80))
	toolsResp, err := c.Tools(srljrpc.PV{
		Path:  "/interface[name=ethernet-1/1]/ethernet/statistics/clear",
		Value: srljrpc.CommandValue("")})
	if err != nil {
		panic(err)
	}
	outHelper(toolsResp.Result)

	// Then for the sake of example we will use DIFF method with Bulk update: TestBulkDiffCandidate.
	// DiffCandidate method is more simple and intended to use in cases you require only one action out of three: UPDATE, DELETE, REPLACE.
	// That's essentially Bulk update with different operations: UPDATE, DELETE, REPLACE, while using yang-models of OpenConfig.
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("BulkDiff() example with error:")
	fmt.Println(strings.Repeat("=", 80))

	pvs = []srljrpc.PV{
		{Path: `/system/config/login-banner`, Value: "DELETE"},
		{Path: `/interfaces/interface[name=mgmt0]/config/description`, Value: "REPLACE"},
		{Path: `/interfaces/interface[name=ethernet-1/11]/subinterfaces/subinterface[index=0]/config/description`, Value: "UPDATE"},
	}
	bulkDiffResp, err := c.BulkDiff(pvs[0:1], pvs[1:2], pvs[2:], yms.OC)
	if err != nil {
		if cerr, ok := err.(apierr.ClientError); ok {
			fmt.Printf("ClientError error: %s\n", cerr) // ClientError
			if cerr.Code == apierr.ErrClntJSONRPC {     // We expect JSON RPC error here and checking via the message code.
				outHelper(bulkDiffResp)
				// Output supposed to be something like this:
				// {
				// 	"jsonrpc": "2.0",
				// 	"id": 568258505525892051,
				// 	"error": {
				// 	  "id": 0,
				// 	  "message": "Server down or restarting"
				// 	}
				//   }
				// This is an indication OC is not supported on the target system, so we will use another target system spine3.
			}
		} else {
			panic(err) // Unexpected outcome.
		}
	} else {
		outHelper(bulkDiffResp.Result)
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("BulkDiff() example with error:")
	fmt.Println(strings.Repeat("=", 80))

	pvs = []srljrpc.PV{
		{Path: `/system/config/login-banner`, Value: "DELETE"},
		{Path: `/interfaces/interface[name=mgmt0]/config/description`, Value: ""}, // Empty value will cause an error.
		{Path: `/interfaces/interface[name=ethernet-1/11]/subinterfaces/subinterface[index=0]/config/description`, Value: "UPDATE"},
	}
	// Change target hostname to spine3, which supports OC.
	// Create a new JSON RPC client with credentials and port (used 443 as default for the sake of demo).
	cOC, err := srljrpc.NewJSONRPCClient(&hostOC, srljrpc.WithOptCredentials(&user, &pass), srljrpc.WithOptPort(&port))
	if err != nil {
		panic(err)
	}

	bulkDiffResp, err = cOC.BulkDiff(pvs[0:1], pvs[1:2], pvs[2:], yms.OC)
	if err != nil {
		// Unwrapping error to investigate a root cause.
		if cerr, ok := err.(apierr.ClientError); ok {
			fmt.Printf("ClientError error: %s\n", cerr)                   // ClientError
			for uerr := err.(apierr.ClientError).Unwrap(); uerr != nil; { // We expect ClientError here, so we can unwrap it.
				fmt.Printf("Underlaying error: %s\n", uerr.Error())
				if u2err, ok := uerr.(interface{ Unwrap() error }); ok {
					uerr = u2err.Unwrap()
				} else {
					break
				}
			}
		}
		// }

	} else {
		outHelper(bulkDiffResp.Result)
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("BulkDiff() example w/o error:")
	fmt.Println(strings.Repeat("=", 80))
	// Adding changes into PV pairs to fix our artificial error and do things right ))
	pvs = []srljrpc.PV{
		{Path: `/system/config/login-banner`, Value: "DELETE"},
		{Path: `/interfaces/interface[name=mgmt0]/config/description`, Value: "REPLACE"},
		{Path: `/interfaces/interface[name=ethernet-1/11]/subinterfaces/subinterface[index=0]/config/description`, Value: "UPDATE"},
	}

	bulkDiffResp, err = cOC.BulkDiff(pvs[0:1], pvs[1:2], pvs[2:], yms.OC)
	if err != nil {
		outHelper(bulkDiffResp)
		panic(err)
	}
	// Parsing JSON response to get the message.
	var data []interface{}
	err = json.Unmarshal(bulkDiffResp.Result, &data)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
	}
	message := data[0].(string)
	fmt.Println(message)

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("BulkSetCallBack() with cancellation:")
	fmt.Println(strings.Repeat("=", 80))
	empty := []srljrpc.PV{}
	sysInfPath := "/interface[name=system0]/description"
	initVal := []srljrpc.PV{{Path: sysInfPath, Value: srljrpc.CommandValue("INITIAL")}}

	_, err = c.Update(0, initVal[0]) // should be no error and system0 interface description should be set to "INITIAL".
	if err != nil {
		panic(err)
	}

	getResp, err = c.Get(sysInfPath)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Get() against %s before BulkSetCallBack():\n", sysInfPath)
	fmt.Println(strings.Repeat("=", 80))
	outHelper(getResp.Result)

	chResp := make(chan *srljrpc.Response) // Channel for response.
	chErr := make(chan error)              // Channel for error.
	go func() {
		newValueToConfirm := []srljrpc.PV{{Path: sysInfPath, Value: srljrpc.CommandValue("System Loopback")}}
		// setting confirmation timeout to 30 seconds to allow comfortable time to verify changes. Setting 27 seconds as time to exec call back function.
		// confirmCallBack is a function to be called after confirmation timeout is expired to confirm or cancel changes as per logic of the implementation.
		resp, err := c.BulkSetCallBack(empty, empty, newValueToConfirm, yms.SRL, 8, 5, confirmCallBack)

		// sending response and error to channels back to main thread.
		chResp <- resp
		chErr <- err
	}()
	// Meanwhile we can do something else in main thread.
	// For example, we can get current value of the interface.
	time.Sleep(2 * time.Second) // Allow 3 seconds to apply changes.
	getResp, err = c.Get(sysInfPath)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Get() against %s:\n", sysInfPath)
	fmt.Println(strings.Repeat("=", 80))
	outHelper(getResp.Result)

	// Waiting for response and error from channel.
	resp := <-chResp
	err = <-chErr
	if err != nil {
		panic(err)
	}
	// We expect response to be nil, as we set confirmation timeout to 30 seconds and call back function to 27 seconds.
	if resp != nil {
		fmt.Println("Unexpected response. Expected nil.")
		outHelper(resp) // Unexpected outcome.
	}
	time.Sleep(30 * time.Second) // Allow enough time to rollback changes.
	getResp, err = c.Get(sysInfPath)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Get() against %s after confirmation timeout expired:\n", sysInfPath)
	fmt.Println(strings.Repeat("=", 80))
	outHelper(getResp.Result)
}

func outHelper(v any) {
	rStr, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", string(rStr))
	fmt.Println(strings.Repeat("=", 80))
}

func confirmCallBack(req *srljrpc.Request, resp *srljrpc.Response) (bool, error) {
	// This is a callback function to be called after confirmation timeout is expired.
	// It is supposed to be used to confirm or cancel changes as per logic of the implementation.
	// In this example we will just print out request and response to console and confirm changes - for the sake ot example that's replace sophisticated logic.
	fmt.Println("Request:")
	outHelper(req)
	fmt.Println("Response:")
	outHelper(resp)
	return false, nil
}

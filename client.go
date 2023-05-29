package main

import (
	"encoding/json"
	"fmt"
	"strings"

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
	fmt.Println("c.Get() example:")
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
	fmt.Println("c.State() example:")
	fmt.Println(strings.Repeat("=", 80))

	stateResp, err := c.State("/system/json-rpc-server")
	if err != nil {
		panic(err)
	}
	outHelper(stateResp.Result)

	// Updating/Replacing/Deleting config
	fmt.Println("c.Update()/Delete()/Replace() example:")
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

	mdmResp, err := c.Update(pvs[0])
	if err != nil {
		panic(err)
	}
	outHelper(mdmResp.Result)
	mdmResp, err = c.Delete(pvs[1].Path)
	if err != nil {
		panic(err)
	}
	outHelper(mdmResp.Result)
	mdmResp, err = c.Replace(pvs[2])
	if err != nil {
		panic(err)
	}
	outHelper(mdmResp.Result)

	// CLI with different formats: JSON and TABLE.
	fmt.Println("c.CLI() example:")
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
	fmt.Println("c.Tools() example:")
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
	fmt.Println("c.TestBulkDiffCandidate() example with error:")
	fmt.Println(strings.Repeat("=", 80))

	pvs = []srljrpc.PV{
		{Path: `/system/config/login-banner`, Value: "DELETE"},
		{Path: `/interfaces/interface[name=mgmt0]/config/description`, Value: "REPLACE"},
		{Path: `/interfaces/interface[name=ethernet-1/11]/subinterfaces/subinterface[index=0]/config/description`, Value: "UPDATE"},
	}
	bulkDiffResp, err := c.BulkDiffCandidate(pvs[0:1], pvs[1:2], pvs[2:], yms.OC)
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

	fmt.Println("c.TestBulkDiffCandidate() example with error:")
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
	fmt.Printf("Target hostname: %s\nTarget system version: %s\n", c.GetHostname(), c.GetSysVer())

	bulkDiffResp, err = cOC.BulkDiffCandidate(pvs[0:1], pvs[1:2], pvs[2:], yms.OC)
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

	// Adding changes into PV pairs to fix our artificial error and do things right ))
	pvs = []srljrpc.PV{
		{Path: `/system/config/login-banner`, Value: "DELETE"},
		{Path: `/interfaces/interface[name=mgmt0]/config/description`, Value: "REPLACE"},
		{Path: `/interfaces/interface[name=ethernet-1/11]/subinterfaces/subinterface[index=0]/config/description`, Value: "UPDATE"},
	}

	bulkDiffResp, err = cOC.BulkDiffCandidate(pvs[0:1], pvs[1:2], pvs[2:], yms.OC)
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

}

func outHelper(v any) {
	rStr, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", string(rStr))
	fmt.Println(strings.Repeat("=", 80))
}

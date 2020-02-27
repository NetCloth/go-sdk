package contract_call_test

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"

	"github.com/netcloth/go-sdk/client"
	"github.com/netcloth/go-sdk/util"
	"github.com/netcloth/netcloth-chain/hexutil"
	sdk "github.com/netcloth/netcloth-chain/types"
	"github.com/stretchr/testify/require"
)

const (
	yamlPath           = "/Users/sun/go/src/github.com/netcloth/go-sdk/config/sdk.yaml"
	contractBech32Addr = "nch1xtuytfypszvyqd0md07jjcsv6wnqx9her4l4tv"
)

var (
	amount = sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(0))
)

func Test_ContractCall(t *testing.T) {
	const (
		functionSig     = "aaa88185" // the first 4 bytes of sig of function: recall
		payloadTemplate = "%s00000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000180000000000000000000000000%s%064x%064x%s%s%064x0000000000000000000000000000000000000000000000000000000000000042%s0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000042%s000000000000000000000000000000000000000000000000000000000000"

		fromPubkeyHexString = "02fc950f1e62b9b2b369448f422808af7d57dbd6ffc0fdbbf2f5849b847285eda8"
		toPubkeyHexString   = "02fc950f1e62b9b2b369448f422808af7d57dbd6ffc0fdbbf2f5849b847285eda8"
		fromAddr            = "AC46A441FAA26708B7783EC48D8742C74C9F7927"
		recallType          = 1
		timestamp           = 100
		rHexString          = "830f66a98feb664f312593f0c3fc9b19eb24d67baae894554c1f44ed3aad5a8e"
		sHexString          = "0fa9a3e5b2a9356c887624750539753cdcc2934867503225e7af5608534444c4"
		v                   = 0x1c
	)

	client, err := client.NewNCHClient(yamlPath)
	t.Log(err)
	require.True(t, err == nil)

	// 构造合约的payload
	payloadStr := fmt.Sprintf(payloadTemplate, functionSig, fromAddr, recallType, timestamp, rHexString, sHexString, v, hexutil.Encode([]byte(fromPubkeyHexString)), hexutil.Encode([]byte(toPubkeyHexString)))
	fmt.Println(fmt.Sprintf("payload: %s", payloadStr))
	payload, err := hex.DecodeString(payloadStr)
	require.NoError(t, err)

	res, err := client.ContractCall(contractBech32Addr, payload, amount, true)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(util.ToJsonIgnoreErr(res))
	}
}

const txHash = "4D71E8E82AF77F9AE0D681053B3D6BBB78D7BCCCE675D6B0905FC8CB3982FF73"

func Test_ContractQuery(t *testing.T) {
	client, err := client.NewNCHClient(yamlPath)
	require.True(t, err == nil)

	txId, err := hexutil.Decode(txHash)
	r, err := client.QueryContractLog(txId)
	require.True(t, err == nil)

	t.Log(r.Result.Logs[0].Data)

	item := r.Result.Logs[0].Data

	revokeTypeStr := item[128:192]
	timestampStr := item[192:256]
	fromPubkeyStr := item[320:452]
	toPubkeyStr := item[576:708]

	t.Log(revokeTypeStr)
	t.Log(timestampStr)
	t.Log(fromPubkeyStr)
	t.Log(toPubkeyStr)
}

func Test_QueryContractEvents(t *testing.T) {
	const (
		startBlockNum = 1
		endBlockNum   = 200
	)

	client, err := client.NewNCHClient(yamlPath)
	require.True(t, err == nil)

	res, err := client.QueryContractEvents(contractBech32Addr, startBlockNum, endBlockNum)
	require.True(t, err == nil)
	t.Log(res)

	for _, item := range res {
		t.Log(item)

		revokeTypeStr := item[128:192]
		timestampStr := item[192:256]
		fromPubkeyStr := item[320:452]
		toPubkeyStr := item[576:708]

		revokeType, _ := strconv.ParseUint(revokeTypeStr, 16, 64)
		timestamp, _ := strconv.ParseUint(timestampStr, 16, 64)

		t.Log(fromPubkeyStr)
		t.Log(toPubkeyStr)
		t.Log(revokeTypeStr)
		t.Log(timestampStr)

		t.Log(revokeType)
		t.Log(timestamp)
	}
}

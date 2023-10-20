package hubclient_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"testing"
	"time"
)

type M1Args struct {
	P1 string
	P2 int
}
type M1Res struct {
	R1 string
}

const senderID = "sender1"

var testContext = hubclient.ServiceContext{
	Context:  context.Background(),
	ClientID: senderID,
}

func Method1Ref(ctx hubclient.ServiceContext, args *M1Args) (*M1Res, error) {
	//slog.Info("Method1 called.", "p1", args.P1)
	res := M1Res{R1: args.P1}
	return &res, nil
}
func Method1Val(ctx hubclient.ServiceContext, args M1Args) (M1Res, error) {
	slog.Info("Method1 called.", "p1", args.P1)
	res := M1Res{R1: args.P1}
	return res, nil
}

func Method2NoArgs() {
	slog.Info("Method2 called.")
}
func Method3ErrorResult(ctx hubclient.ServiceContext) error {
	slog.Info("Method3 called, returning error")
	return fmt.Errorf("method3 returns an error")
}
func Method4DataAndErrorResult(ctx hubclient.ServiceContext) (M1Res, error) {
	res := M1Res{R1: "data and err"}
	return res, fmt.Errorf("data and error")
}
func Method5StringArg(ctx hubclient.ServiceContext, arg1 string) (string, error) {
	slog.Info("received string arg", "arg1", arg1)
	return arg1, nil
}
func Method6IntArg(ctx hubclient.ServiceContext, arg1 int) (int, error) {
	slog.Info("received int arg", "arg1", arg1)
	return arg1, nil
}
func Method7ByteArrayArg(ctx hubclient.ServiceContext, arg1 []byte) ([]byte, error) {
	slog.Info("received array arg", "arg1", arg1)
	return arg1, nil
}
func Method8TwoArgsFail(ctx hubclient.ServiceContext, arg1 string, args2 string) (string, error) {
	slog.Info("received 2 args doesn't work", "arg1", arg1)
	return arg1, nil
}
func Method9ThreeResFail(ctx hubclient.ServiceContext, arg1 string) (string, string, error) {
	slog.Info("returning 3 results should fail", "arg1", arg1)
	return arg1, arg1, nil
}

//---

func TestHandleRequestMessage(t *testing.T) {
	// pass args by reference
	args := M1Args{
		P1: "agent1",
		P2: 5,
	}
	argsJson, _ := ser.Marshal(args)

	data, err := hubclient.HandleRequestMessage(testContext, Method1Ref, argsJson)
	require.NoError(t, err)
	m1res := M1Res{}
	err = json.Unmarshal(data, &m1res)
	require.NoError(t, err)
	assert.Equal(t, args.P1, m1res.R1)

	// pass args by value
	args = M1Args{
		P1: "agent2",
		P2: 6,
	}
	argsJson, _ = ser.Marshal(args)
	data, err = hubclient.HandleRequestMessage(testContext, Method1Val, argsJson)
	require.NoError(t, err)
	m1res = M1Res{}
	err = json.Unmarshal(data, &m1res)
	require.NoError(t, err)
	assert.Equal(t, args.P1, m1res.R1)
}

func TestHandleRequestNoArgs(t *testing.T) {
	// pass args by reference
	data, err := hubclient.HandleRequestMessage(testContext, Method2NoArgs, nil)
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestErrorResult(t *testing.T) {
	// pass args by reference
	data, err := hubclient.HandleRequestMessage(testContext, Method3ErrorResult, nil)
	require.Error(t, err)
	assert.Nil(t, data)
}

func TestDataAndErrorResult(t *testing.T) {
	// check this doesnt fail somehow
	_, err := hubclient.HandleRequestMessage(testContext, Method4DataAndErrorResult, nil)
	require.Error(t, err)
}

func TestStringArgs(t *testing.T) {
	// check this doesnt fail somehow
	sargJson, _ := json.Marshal("Hello world")
	data, err := hubclient.HandleRequestMessage(testContext, Method5StringArg, sargJson)
	require.NoError(t, err)

	var result string
	err = json.Unmarshal(data, &result)
	require.Equal(t, "Hello world", result)
	require.NoError(t, err)
}

func TestIntArgs(t *testing.T) {
	// check this doesnt fail somehow
	sargJson, _ := json.Marshal(25)
	data, err := hubclient.HandleRequestMessage(testContext, Method6IntArg, sargJson)
	require.NoError(t, err)

	var result int
	err = json.Unmarshal(data, &result)
	require.Equal(t, 25, result)
	require.NoError(t, err)
}
func TestByteArrayArgs(t *testing.T) {
	args := []byte{1, 2, 3}
	argJson, _ := json.Marshal(args)
	data, err := hubclient.HandleRequestMessage(testContext, Method7ByteArrayArg, argJson)
	require.NoError(t, err)

	var result []byte
	err = json.Unmarshal(data, &result)
	require.Equal(t, args, result)
	require.NoError(t, err)
}
func TestTwoArgsFail(t *testing.T) {
	sargJson, _ := json.Marshal("Hello world")
	// this method has 2 args, we only pass 1. Does it blow up?
	data, err := hubclient.HandleRequestMessage(testContext, Method8TwoArgsFail, sargJson)
	assert.Error(t, err)
	assert.Nil(t, data)
}
func TestThreeResFail(t *testing.T) {
	sargJson, _ := json.Marshal("Hello world")
	// this method has 3 results. Does it blow up?
	data, err := hubclient.HandleRequestMessage(testContext, Method9ThreeResFail, sargJson)
	assert.Error(t, err)
	assert.Nil(t, data)
}
func Benchmark_Overhead(b *testing.B) {
	m1args := M1Args{
		P1: "agent1",
		P2: 5,
	}
	count1 := uint64(0)
	count2 := uint64(0)
	b.Run("direct call, no marshalling",
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				// pass args by reference
				m1res, err := Method1Ref(testContext, &m1args)
				_ = err
				_ = m1res
			}
		})
	t1 := time.Now()
	b.Run("direct call, with marshalling",
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				count1++
				// a remote call would marshal and unmarshal the request parameters
				argsJson, _ := ser.Marshal(m1args)
				m1args2 := M1Args{}
				_ = json.Unmarshal(argsJson, &m1args2)
				m1res, err := Method1Ref(testContext, &m1args)
				// a remote call would marshal and unmarshal the result
				data, err := json.Marshal(m1res)
				_ = err
				_ = m1res
				_ = json.Unmarshal(data, &m1res)
			}
		})
	t2 := time.Now()
	b.Run("indirect call",
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				count2++
				// pass args by reference
				argsJson, _ := ser.Marshal(m1args)
				data, err := hubclient.HandleRequestMessage(testContext, Method1Ref, argsJson)
				_ = err
				m1res := M1Res{}
				err = json.Unmarshal(data, &m1res)
			}
		})
	t3 := time.Now()
	d1 := uint64(t2.Sub(t1)) / count1
	d2 := uint64(t3.Sub(t2)) / count2
	overhead := (d2 - d1)
	fmt.Printf("HandleRequestMessage overhead: %d nsec per call;  marshalling/unmarshalling: %d nsec/call\n", overhead, d1)
}

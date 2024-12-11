package hubagent_test

import (
	"fmt"
	"github.com/hiveot/hub/lib/hubagent"
	jsoniter "github.com/json-iterator/go"
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

func Method1Ref(args *M1Args) (*M1Res, error) {
	//slog.Info("Method1 called.", "p1", args.P1)
	res := M1Res{R1: args.P1}
	return &res, nil
}
func Method1Val(args M1Args) (M1Res, error) {
	slog.Info("Method1 called.", "p1", args.P1)
	res := M1Res{R1: args.P1}
	return res, nil
}

func Method2NoArgs() {
	slog.Info("Method2 called.")
}
func Method3ErrorResult() error {
	slog.Info("Method3 called, returning error")
	return fmt.Errorf("method3 returns an error")
}
func Method4DataAndErrorResult() (M1Res, error) {
	res := M1Res{R1: "data and err"}
	return res, fmt.Errorf("data and error")
}
func Method5StringArg(sender string, arg1 string) (string, error) {
	slog.Info("received string arg", "arg1", arg1)
	return arg1, nil
}
func Method6IntArg(sender string, arg1 int) (int, error) {
	slog.Info("received int arg", "arg1", arg1)
	return arg1, nil
}
func Method7ByteArrayArg(sender string, arg1 []byte) ([]byte, error) {
	slog.Info("received array arg", "arg1", arg1)
	return arg1, nil
}
func Method8TwoArgs(sender string, arg1 string, arg2 int) (string, error) {
	// this fails as arg2 is an int
	slog.Info("received 2 args", "arg1", arg1, "arg2", arg2)
	return arg1, nil
}
func Method9ThreeRes(sender string, arg1 string) (string, string, error) {
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
	data, err := hubagent.HandleRequestMessage(senderID, Method1Ref, args)
	require.NoError(t, err)
	m1resp, ok := data.(*M1Res)
	require.True(t, ok)

	require.Equal(t, args.P1, m1resp.R1)

	// pass args and response by value
	args = M1Args{
		P1: "agent2",
		P2: 6,
	}
	data, err = hubagent.HandleRequestMessage(senderID, Method1Val, args)
	require.NoError(t, err)
	m1resVal, ok := data.(M1Res)
	require.True(t, ok)
	require.NoError(t, err)
	assert.Equal(t, args.P1, m1resVal.R1)
}

func TestHandleRequestNoArgs(t *testing.T) {
	// pass args by reference
	data, err := hubagent.HandleRequestMessage(senderID, Method2NoArgs, "")
	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestErrorResult(t *testing.T) {
	// pass args by reference
	data, err := hubagent.HandleRequestMessage(senderID, Method3ErrorResult, "")
	require.Error(t, err)
	assert.Empty(t, data)
}

func TestDataAndErrorResult(t *testing.T) {
	// check this doesnt fail somehow
	_, err := hubagent.HandleRequestMessage(senderID, Method4DataAndErrorResult, "")
	require.Error(t, err)
}

func TestStringArgs(t *testing.T) {
	// check this doesn't fail somehow
	sargs := "Hello world"

	data, err := hubagent.HandleRequestMessage(senderID, Method5StringArg, sargs)
	require.NoError(t, err)
	result := data.(string)

	require.Equal(t, "Hello world", result)
	require.NoError(t, err)
}

func TestIntArgs(t *testing.T) {
	// check this doesnt fail somehow
	sarg := 25
	data, err := hubagent.HandleRequestMessage(senderID, Method6IntArg, sarg)
	require.NoError(t, err)
	result := data.(int)

	require.Equal(t, 25, result)
	require.NoError(t, err)
}
func TestByteArrayArgs(t *testing.T) {
	args := []byte{1, 2, 3}
	data, err := hubagent.HandleRequestMessage(senderID, Method7ByteArrayArg, args)
	require.NoError(t, err)

	result := data.([]byte)
	require.Equal(t, args, result)
	require.NoError(t, err)
}
func TestTwoArgsFail(t *testing.T) {
	sargJson, _ := jsoniter.Marshal("Hello world")
	// this method has 2 args, we only pass 1. Does it blow up?
	data, err := hubagent.HandleRequestMessage(senderID, Method8TwoArgs, string(sargJson))
	assert.Error(t, err)
	assert.Empty(t, data)
}
func TestThreeResFail(t *testing.T) {
	sarg := "Hello world"
	// this method has 3 results. Does it blow up?
	data, err := hubagent.HandleRequestMessage(senderID, Method9ThreeRes, sarg)
	assert.Error(t, err)
	assert.Empty(t, data)
}
func Benchmark_Overhead(b *testing.B) {
	m1args := M1Args{
		P1: "agent1",
		P2: 5,
	}
	count1 := uint64(0)
	count2 := uint64(0)
	// 0.32 ns/op
	b.Run("direct call, no marshalling",
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				// pass args by reference
				m1res, err := Method1Ref(&m1args)
				_ = err
				_ = m1res
			}
		})
	t1 := time.Now()
	// 2673 ns/op  (2.5 usec with json, 1.1 usec with jsoniter)
	b.Run("direct call, with marshalling",
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				count1++
				// a remote call would marshal and unmarshal the request parameters
				argsJson, _ := jsoniter.Marshal(m1args)
				m1args2 := M1Args{}
				_ = jsoniter.Unmarshal(argsJson, &m1args2)
				m1res, err := Method1Ref(&m1args)
				// a remote call would marshal and unmarshal the result
				data, err := jsoniter.Marshal(m1res)
				_ = err
				_ = m1res
				_ = jsoniter.Unmarshal(data, &m1res)
			}
		})
	t2 := time.Now()
	// 3545 ns/op (3.5 usec for reflection w json, 2.0 usec jsoniter)
	b.Run("indirect call",
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				count2++
				// pass args by reference
				data, err := hubagent.HandleRequestMessage(senderID, Method1Ref, &m1args)
				_ = err
				m1res, ok := data.(*M1Res)
				require.True(b, ok)
				require.NotEmpty(b, m1res)
			}
		})
	t3 := time.Now()
	d1 := uint64(t2.Sub(t1)) / count1
	d2 := uint64(t3.Sub(t2)) / count2
	overhead := (d2 - d1)
	fmt.Printf("HandleRequestMessage overhead: %d nsec per call;  marshalling/unmarshalling: %d nsec/call\n", overhead, d1)
}

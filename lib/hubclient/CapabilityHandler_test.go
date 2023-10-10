package hubclient_test

import (
	"encoding/json"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"reflect"
	"testing"
)

type M1Args struct {
	P1 string
	P2 int
}
type M1Res struct {
	R1 string
}

// test method 1
const Method1Name = "method1"

func Method1(args *M1Args) (M1Res, error) {
	slog.Info("Method1 called.", "p1", args.P1)
	res := M1Res{R1: args.P1}
	return res, nil
}

func TestHandlerMap(t *testing.T) {
	args := M1Args{
		P1: "agent1",
		P2: 5,
	}
	argsJson, _ := ser.Marshal(args)

	capHdlr := hubclient.CapabilityHandler{
		ArgsType: M1Args{},
		RespType: M1Res{},
		Method:   reflect.ValueOf(Method1),
	}
	data, err := capHdlr.HandleMessage(argsJson)
	m1res := M1Res{}
	json.Unmarshal(data, &m1res)

	require.NoError(t, err)
	assert.Equal(t, args.P1, m1res.R1)
}

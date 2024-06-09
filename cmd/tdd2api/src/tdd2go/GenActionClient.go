package tdd2go

import (
	"fmt"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
	"regexp"
)

// GenServiceClient generates a client function for invoking a Thing action.
//
// This client will marshal the API parameters into an action argument struct and
// invoke the method using the provided messaging transport.
//
// The TD document must be a digital twin received version
func GenServiceClient(l *utils.L, serviceTitle string, td *things.TD) {

	l.Add("// %sClient client for talking to the '%s' service", serviceTitle, td.ID)
	l.Add("type %sClient struct {", serviceTitle)
	l.Add("   dThingID string")
	l.Add("   hc hubclient.IHubClient")
	l.Add("}")

	actionKeys := utils.OrderedMapKeys(td.Actions)
	for _, key := range actionKeys {
		action := td.Actions[key]
		GenActionMethod(l, serviceTitle, key, action)
	}
	l.Add("")
	l.Add("// New%sClient creates a new client for invoking %s methods.", serviceTitle, td.Title)
	l.Add("func New%sClient(hc hubclient.IHubClient) *%sClient {", serviceTitle, serviceTitle)
	l.Add("	cl := %sClient {", serviceTitle)
	l.Add("		hc: hc,")
	l.Add("		dThingID: \"%s\",", td.ID)
	l.Add("	}")
	l.Add("	return &cl")
	l.Add("}")

}

// GenActionMethod generates a client function from an action affordance.
//
//	serviceTitle  title-case thingID of the service without the agent prefix
//	key with the service action method.
//	action affordance describing the input and output parameters
func GenActionMethod(l *utils.L, serviceTitle string, key string, action *things.ActionAffordance) {
	argsString := ""
	respString := "err error"
	invokeArgs := "nil"
	invokeResp := "nil"

	methodName := Key2ID(key)
	if action.Input != nil {
		argName := getParamName("args", action.Input)
		goType := ""
		// client arguments are in alpha name order
		// create a list of fields from the input
		if action.Input.Type == "object" && action.Input.Properties != nil {
			goType = fmt.Sprintf("%s%sArgs", serviceTitle, methodName)
		} else {
			goType = GoTypeFromSchema(action.Input)
		}
		argsString = fmt.Sprintf("%s %s", argName, goType)
		invokeArgs = "&" + argName
	}
	// add a response struct to arguments
	if action.Output != nil {
		respName := getParamName("resp", action.Output)
		goType := ""
		if action.Output.Type == "object" && action.Output.Properties != nil {
			goType = fmt.Sprintf("%s%sResp", serviceTitle, methodName)
		} else {
			goType = GoTypeFromSchema(action.Output)
		}
		respString = fmt.Sprintf("%s %s, err error", respName, goType)
		invokeResp = "&" + respName
	}
	// Function declaration
	l.Indent = 0
	l.Add("")
	l.Add("// %s client method - %s.", methodName, action.Title)
	if len(action.Description) > 0 {
		l.Add("// %s", action.Description)
	}
	l.Add("func (svc *%sClient) %s(%s)(%s){", serviceTitle, methodName, argsString, respString)
	l.Indent++

	l.Add("err = svc.hc.Rpc(svc.dThingID, %s%sMethod, %s, %s)", serviceTitle, methodName, invokeArgs, invokeResp)
	l.Add("return")
	l.Indent--
	l.Add("}")
}

// Generate a parameter name from the schema title.
// Parameter names start with lower case and consist only of alpha-num chars
// Intended to make the api more readable.
func getParamName(defaultName string, ds *things.DataSchema) string {
	if ds.Title == "" {
		return defaultName
	}
	str := regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(ds.Title, "")
	str = FirstToLower(str)
	return str
}

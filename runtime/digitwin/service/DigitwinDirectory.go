package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/wot/tdd"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
)

// Digital Twin Directory Service
type DigitwinDirectoryService struct {
	dtwStore *DigitwinStore
	// The handler for replacing the forms in the given TD to access the
	// digital twin instead.
	addTDForms func(td *tdd.TD) error
}

// MakeDigitalTwinTD returns the digital twin from a agent provided TD
func (svc *DigitwinDirectoryService) MakeDigitalTwinTD(
	agentID string, tdJSON string) (thingTD *tdd.TD, dtwTD *tdd.TD, err error) {

	_ = jsoniter.Unmarshal([]byte(tdJSON), &thingTD)
	_ = jsoniter.Unmarshal([]byte(tdJSON), &dtwTD)

	dtwTD.ID = tdd.MakeDigiTwinThingID(agentID, thingTD.ID)

	// remove all existing forms and auth info
	dtwTD.Forms = make([]tdd.Form, 0)
	dtwTD.Security = ""
	dtwTD.SecurityDefinitions = make(map[string]tdd.SecurityScheme)
	for _, aff := range dtwTD.Properties {
		aff.Forms = make([]tdd.Form, 0)
	}
	for _, aff := range dtwTD.Events {
		aff.Forms = make([]tdd.Form, 0)
	}
	for _, aff := range dtwTD.Actions {
		aff.Forms = make([]tdd.Form, 0)
	}

	if svc.addTDForms != nil {
		err = svc.addTDForms(dtwTD)
	}
	return thingTD, dtwTD, err
}

// QueryDTDs returns a list of JSON encoded TD documents
func (svc *DigitwinDirectoryService) QueryDTDs(
	senderID string, args digitwin.DirectoryQueryDTDsArgs) (tdDocuments []string, err error) {
	//svc.DtwStore.QueryDTDs(args)
	return nil, fmt.Errorf("Not yet implemented")
}

// ReadDTD returns a JSON encoded TD document
func (svc *DigitwinDirectoryService) ReadDTD(senderID string, dThingID string) (tdJSON string, err error) {
	dtd, err := svc.dtwStore.ReadDThing(dThingID)
	if err == nil {
		var tdByte []byte
		tdByte, err = jsoniter.Marshal(dtd)
		tdJSON = string(tdByte)
	}
	return tdJSON, err
}

// ReadAllDTDs returns a batch of TD documents
// This returns a list of JSON encoded digital twin TD documents
func (svc *DigitwinDirectoryService) ReadAllDTDs(
	senderID string, args digitwin.DirectoryReadAllDTDsArgs) (tdList []string, err error) {

	dtdList, err := svc.dtwStore.ReadDTDs(args.Offset, args.Limit)
	if err == nil {
		tdList = make([]string, 0, len(dtdList))
		for _, dtd := range dtdList {
			tdByte, err2 := jsoniter.Marshal(dtd)
			if err2 == nil {
				tdList = append(tdList, string(tdByte))
			}
		}
	}
	return tdList, err
}

// RemoveDTD removes a Thing TD document from the digital twin directory
func (svc *DigitwinDirectoryService) RemoveDTD(senderID string, dThingID string) error {
	err := svc.dtwStore.RemoveDTW(dThingID)
	return err
}

// UpdateDTD updates the digitwin TD from an agent supplied TD
// This transforms the given TD into the digital twin instance.
func (svc *DigitwinDirectoryService) UpdateDTD(agentID string, tdJson string) error {
	slog.Info("UpdateDTD")

	// transform the agent provided TD into a digital twin's TD
	thingTD, digitalTwinTD, err := svc.MakeDigitalTwinTD(agentID, tdJson)
	svc.dtwStore.UpdateTD(agentID, thingTD, digitalTwinTD)
	return err
}

// NewDigitwinDirectoryService creates a new instance of the directory service
// using the given store.
func NewDigitwinDirectoryService(
	dtwStore *DigitwinStore, addTDForms func(td *tdd.TD) error) *DigitwinDirectoryService {
	dirSvc := &DigitwinDirectoryService{
		dtwStore:   dtwStore,
		addTDForms: addTDForms,
	}

	// verify service interface matches the TD generated interface
	var s digitwin.IDirectoryService = dirSvc
	_ = s

	return dirSvc
}

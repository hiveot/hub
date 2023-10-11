package capserializer

import (
	"capnproto.org/go/capnp/v3"

	"github.com/hiveot/hub/api/go/hubapi"
	"github.com/hiveot/hub/lib/caphelp"
	"github.com/hiveot/hub/pkg/history"
)

func MarshalRetList(retList []history.EventRetention) hubapi.EventRetention_List {
	_, seg, _ := capnp.NewMessage(capnp.SingleSegment(nil))
	capRetList, _ := hubapi.NewEventRetention_List(seg, int32(len(retList)))

	for i := 0; i < len(retList); i++ {
		ret := retList[i]
		//logrus.Infof("ret name=%s", ret.ID)
		capRet := MarshalEventRetention(ret)
		capRetList.Set(i, capRet)
	}
	return capRetList
}

func MarshalEventRetention(retention history.EventRetention) (capRet hubapi.EventRetention) {
	_, seg, _ := capnp.NewMessage(capnp.SingleSegment(nil))
	capRet, _ = hubapi.NewEventRetention(seg)
	_ = capRet.SetName(retention.Name)
	_ = capRet.SetExclude(caphelp.MarshalStringList(retention.Exclude))
	_ = capRet.SetPublishers(caphelp.MarshalStringList(retention.Publishers))
	_ = capRet.SetThings(caphelp.MarshalStringList(retention.Things))
	capRet.SetRetentionDays(int32(retention.RetentionDays))
	return capRet
}

func UnmarshalRetList(capRetList hubapi.EventRetention_List) []history.EventRetention {
	retList := make([]history.EventRetention, 0, capRetList.Len())
	for i := 0; i < capRetList.Len(); i++ {
		capRet := capRetList.At(i)
		ret := UnmarshalEventRetention(capRet)
		retList = append(retList, ret)
	}
	return retList
}

func UnmarshalEventRetention(capRet hubapi.EventRetention) history.EventRetention {
	name, _ := capRet.Name()
	capPub, _ := capRet.Publishers()
	capThings, _ := capRet.Things()
	capExcludes, _ := capRet.Exclude()

	ret := history.EventRetention{
		Name:          name,
		Publishers:    caphelp.UnmarshalStringList(capPub),
		Things:        caphelp.UnmarshalStringList(capThings),
		Exclude:       caphelp.UnmarshalStringList(capExcludes),
		RetentionDays: int(capRet.RetentionDays()),
	}
	return ret
}

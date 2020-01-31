package proto

import (
	"encoding/json"
	"testing"
)

func TestEnc(t *testing.T) {
	spec, err := Parse([]string{"b2b-api.proto"}, []string{"/Users/avasyukin/Work/b2b-api/api", "/Users/avasyukin/Work/b2b-api/vendor.pb"})
	if err != nil {
		t.Error(err)
	}
	// t.Log(spec)
	rpc, err := spec.RPC("ozon.travel.api.b2b", "B2BAPI", "RequestReconciliationReportForPeriod")
	if err != nil {
		t.Error(err)
	}
	t.Log(rpc)
	req, err := rpc.RequestType.New()
	if err != nil {
		t.Error(err)
	}
	t.Log(req)
	err = json.Unmarshal([]byte(`{"email":"korror@yandex.ru","from":"10.09.2019","to":"10.10.2019"}`), req)
	if err != nil {
		t.Error(err)
	}
	t.Log(req)
}

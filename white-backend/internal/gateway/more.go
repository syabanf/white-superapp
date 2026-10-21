package gateway

import (
	"context"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

var emptyPB = emptypb.Empty{}

// registerMore holds registrations for services that land after the first
// cut (publishing, seosuite, reports). Kept separate so each service's
// agent adds one line here.
func registerMore(ctx context.Context, mux *runtime.ServeMux, u *Upstreams) error {
	return nil
}

func writeProtoJSON(w http.ResponseWriter, m proto.Message) {
	w.Header().Set("Content-Type", "application/json")
	b, _ := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(m)
	_, _ = w.Write(b)
}

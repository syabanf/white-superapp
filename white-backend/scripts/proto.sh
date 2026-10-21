#!/usr/bin/env bash
# Generates Go stubs, grpc-gateway handlers and OpenAPI JSON for every
# white/**/v1/*.proto. Run from the repo root: ./scripts/proto.sh
set -euo pipefail

cd "$(dirname "$0")/.."
export PATH="$PATH:$(go env GOPATH)/bin"

OUT=gen/go
OPENAPI=gen/openapi
rm -rf "$OUT" "$OPENAPI"
mkdir -p "$OUT" "$OPENAPI"

PROTOC_INCLUDE="$(dirname "$(dirname "$(command -v protoc)")")/include"

FILES=()
while IFS= read -r f; do FILES+=("$f"); done < <(find api/proto/white -name '*.proto' | sort)

protoc \
  -I api/proto \
  -I api/proto/third_party \
  -I "$PROTOC_INCLUDE" \
  --go_out="$OUT" --go_opt=module=github.com/syabanf/white-superapp/white-backend/gen/go \
  --go-grpc_out="$OUT" --go-grpc_opt=module=github.com/syabanf/white-superapp/white-backend/gen/go \
  --grpc-gateway_out="$OUT" --grpc-gateway_opt=module=github.com/syabanf/white-superapp/white-backend/gen/go \
  --grpc-gateway_opt=generate_unbound_methods=false \
  --openapiv2_out="$OPENAPI" \
  --openapiv2_opt=allow_merge=true,merge_file_name=white,json_names_for_fields=true,disable_default_errors=true \
  "${FILES[@]}"

echo "generated $(find "$OUT" -name '*.pb.go' | wc -l | tr -d ' ') Go files"
cp "$OPENAPI/white.swagger.json" internal/gateway/openapi/white.swagger.json

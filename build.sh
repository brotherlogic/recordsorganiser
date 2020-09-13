protoc --proto_path ../../../ -I=./proto --go_out=plugins=grpc:./proto proto/organise.proto
mv proto/github.com/brotherlogic/recordsorganiser/proto/* ./proto

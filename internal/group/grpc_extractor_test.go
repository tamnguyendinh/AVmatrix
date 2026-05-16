package group

import (
	"reflect"
	"testing"
)

func TestExtractGRPCProtoProviders(t *testing.T) {
	tmpDir := t.TempDir()
	writeGroupFile(t, tmpDir, "proto/auth.proto", `syntax = "proto3";
package auth;
service AuthService {
  rpc Login (LoginRequest) returns (LoginResponse);
}`)
	writeGroupFile(t, tmpDir, "api/user.proto", `syntax = "proto3";
package hr.user.v1;
service UserService {
  rpc GetUser (GetUserRequest) returns (UserResponse);
  rpc ListUsers (ListUsersRequest) returns (ListUsersResponse);
  rpc DeleteUser (DeleteUserRequest) returns (Empty);
}`)
	writeGroupFile(t, tmpDir, "service.proto", `syntax = "proto3";
service HealthCheck { rpc Check (HealthRequest) returns (HealthResponse); }`)

	contracts, err := ExtractGRPCContracts(tmpDir)
	if err != nil {
		t.Fatalf("ExtractGRPCContracts() error = %v", err)
	}
	for _, want := range []string{
		"grpc::auth.AuthService/Login",
		"grpc::hr.user.v1.UserService/DeleteUser",
		"grpc::hr.user.v1.UserService/GetUser",
		"grpc::hr.user.v1.UserService/ListUsers",
		"grpc::HealthCheck/Check",
	} {
		if findContract(contracts, want) == nil {
			t.Fatalf("grpc providers missing %s\nall=%v", want, contractIDs(contracts))
		}
	}
	provider := findContract(contracts, "grpc::auth.AuthService/Login")
	if provider == nil || provider.Confidence != 0.85 || provider.SymbolRef.FilePath != "proto/auth.proto" {
		t.Fatalf("auth proto provider = %#v", provider)
	}
}

func TestExtractGRPCProtoParserHandlesNestedCommentsStringsAndMalformedServices(t *testing.T) {
	tmpDir := t.TempDir()
	writeGroupFile(t, tmpDir, "api/gateway.proto", `syntax = "proto3";
package gateway.v1;
service GatewayService {
  rpc GetUser (GetUserRequest) returns (UserResponse) {
    option (google.api.http) = {
      get: "/v1/users/{user_id}"
    };
  }
  rpc CreateUser (CreateUserRequest) returns (UserResponse) {
    option (google.api.http) = {
      post: "/v1/users"
      body: "*"
    };
  }
}`)
	writeGroupFile(t, tmpDir, "api/strings.proto", `syntax = "proto3";
package strings;
service TrickyService {
  rpc First (Req) returns (Res) {
    option (google.api.http).additional_bindings = {
      post: "/v1/first";
    };
  }
  option deprecated_reason = "use NewService { instead";
  rpc Second (Req) returns (Res);
  rpc Third (Req) returns (Res);
}`)
	writeGroupFile(t, tmpDir, "api/commented.proto", `syntax = "proto3";
package commented;
service Svc {
  // TODO: move { or } from this comment
  /* A block comment with { unbalanced braces } */
  rpc Alpha (Req) returns (Res);
  // }} end of the method block (in comment)
  rpc Beta (Req) returns (Res);
}`)
	writeGroupFile(t, tmpDir, "api/broken.proto", `syntax = "proto3";
package broken;
service IncompleteService {
  rpc SomeMethod (Req) returns (Res);`)

	contracts, err := ExtractGRPCContracts(tmpDir)
	if err != nil {
		t.Fatalf("ExtractGRPCContracts() error = %v", err)
	}
	for _, want := range []string{
		"grpc::gateway.v1.GatewayService/GetUser",
		"grpc::gateway.v1.GatewayService/CreateUser",
		"grpc::strings.TrickyService/First",
		"grpc::strings.TrickyService/Second",
		"grpc::strings.TrickyService/Third",
		"grpc::commented.Svc/Alpha",
		"grpc::commented.Svc/Beta",
	} {
		if findContract(contracts, want) == nil {
			t.Fatalf("grpc contracts missing %s\nall=%v", want, contractIDs(contracts))
		}
	}
	if findContract(contracts, "grpc::broken.IncompleteService/SomeMethod") != nil {
		t.Fatalf("malformed service should have been skipped: %v", contractIDs(contracts))
	}
}

func TestBuildProtoMapAndResolveProtoConflict(t *testing.T) {
	tmpDir := t.TempDir()
	writeGroupFile(t, tmpDir, "proto/user.proto", `syntax = "proto3";
package com.example;
service UserService {
  rpc GetUser (GetUserRequest) returns (GetUserResponse);
  rpc ListUsers (ListUsersRequest) returns (ListUsersResponse);
}`)
	writeGroupFile(t, tmpDir, "proto/shared/package.proto", "package auth.v1;\nmessage LoginRequest {}")
	writeGroupFile(t, tmpDir, "proto/services/auth.proto", `import "../shared/package.proto";
service AuthService { rpc Login (LoginRequest) returns (LoginRequest); }`)
	writeGroupFile(t, tmpDir, "a/svc.proto", "package pkg.a;\nservice Svc { rpc Do (R) returns (R); }")
	writeGroupFile(t, tmpDir, "b/svc.proto", "package pkg.b;\nservice Svc { rpc Do (R) returns (R); }")

	protoMap, err := BuildProtoMap(tmpDir)
	if err != nil {
		t.Fatalf("BuildProtoMap() error = %v", err)
	}
	userEntries := protoMap["UserService"]
	if len(userEntries) != 1 || userEntries[0].Package != "com.example" || !reflect.DeepEqual(userEntries[0].Methods, []string{"GetUser", "ListUsers"}) {
		t.Fatalf("UserService entries = %#v", userEntries)
	}
	authEntries := protoMap["AuthService"]
	if len(authEntries) != 1 || authEntries[0].Package != "auth.v1" {
		t.Fatalf("AuthService inherited package entries = %#v", authEntries)
	}
	if len(protoMap["Svc"]) != 2 {
		t.Fatalf("Svc conflicts = %#v", protoMap["Svc"])
	}

	winner := ResolveProtoConflict("Svc", "a/src/server.go", protoMap["Svc"])
	if winner == nil || winner.Package != "pkg.a" {
		t.Fatalf("ResolveProtoConflict winner = %#v", winner)
	}
	if got := ResolveProtoConflict("Svc", "src/main.go", protoMap["Svc"]); got != nil {
		t.Fatalf("ambiguous conflict should return nil, got %#v", got)
	}
	if got := ResolveProtoConflict("Svc", "src/main.go", nil); got != nil {
		t.Fatalf("empty candidates should return nil, got %#v", got)
	}
}

func TestExtractGRPCSourceScannersUseProtoContext(t *testing.T) {
	tmpDir := t.TempDir()
	writeGroupFile(t, tmpDir, "proto/user.proto", "package com.example;\nservice UserService { rpc GetUser (R) returns (R); }")
	writeGroupFile(t, tmpDir, "src/server.go", "package main\nfunc init() { pb.RegisterUserServiceServer(srv, &impl{}) }")
	writeGroupFile(t, tmpDir, "src/client.go", "package main\nfunc init() { client := pb.NewUserServiceClient(conn) }")
	writeGroupFile(t, tmpDir, "src/main/java/UserGrpcService.java", `@GrpcService
public class UserGrpcService extends UserServiceGrpc.UserServiceImplBase {
    @Override
    public void getUser(GetUserRequest req, StreamObserver<GetUserResponse> obs) {}
}`)
	writeGroupFile(t, tmpDir, "client.py", `import grpc
channel = grpc.insecure_channel('localhost:50051')
stub = UserServiceStub(channel)`)
	writeGroupFile(t, tmpDir, "src/controller.ts", "@GrpcMethod('UserService', 'GetUser')\nasync getUser() {}")
	writeGroupFile(t, tmpDir, "src/auth.client.ts", `import type { ClientGrpc } from '@nestjs/microservices';
export function createAuthClient(client: ClientGrpc) {
  return client.getService<UserService>('UserService');
}
export const authClient = new UserServiceClient('localhost:50051');`)

	contracts, err := ExtractGRPCContracts(tmpDir)
	if err != nil {
		t.Fatalf("ExtractGRPCContracts() error = %v", err)
	}
	for _, want := range []string{
		"grpc::com.example.UserService/GetUser",
		"grpc::com.example.UserService/*",
	} {
		if findContract(contracts, want) == nil {
			t.Fatalf("grpc source contracts missing %s\nall=%v", want, contractIDs(contracts))
		}
	}
	for _, contract := range contracts {
		if contract.Meta["source"] == "go_register" && contract.ContractID != "grpc::com.example.UserService/*" {
			t.Fatalf("go provider contract = %#v", contract)
		}
		if contract.Meta["source"] == "go_client" && contract.Confidence != 0.75 {
			t.Fatalf("go client confidence = %#v", contract)
		}
	}
}

func TestExtractGRPCWithoutProtoFallsBackAndAmbiguousSourceSkips(t *testing.T) {
	noProto := t.TempDir()
	writeGroupFile(t, noProto, "src/server.go", "package main\nfunc init() { pb.RegisterFooServer(srv, &impl{}) }")
	contracts, err := ExtractGRPCContracts(noProto)
	if err != nil {
		t.Fatalf("ExtractGRPCContracts(noProto) error = %v", err)
	}
	got := findContract(contracts, "grpc::Foo/*")
	if got == nil || got.Confidence != 0.65 {
		t.Fatalf("fallback provider = %#v all=%v", got, contractIDs(contracts))
	}

	ambiguous := t.TempDir()
	writeGroupFile(t, ambiguous, "billing-team/proto/user.proto", "package billing.v1;\nservice UserService { rpc GetUser (R) returns (R); }")
	writeGroupFile(t, ambiguous, "auth-team/proto/user.proto", "package auth.v1;\nservice UserService { rpc GetUser (R) returns (R); }")
	writeGroupFile(t, ambiguous, "apps/gateway/client.go", "package main\nfunc init() { client := pb.NewUserServiceClient(conn) }")
	contracts, err = ExtractGRPCContracts(ambiguous)
	if err != nil {
		t.Fatalf("ExtractGRPCContracts(ambiguous) error = %v", err)
	}
	for _, contract := range contracts {
		if contract.Meta["source"] == "go_client" && contract.Meta["service"] == "UserService" {
			t.Fatalf("ambiguous source contract should be skipped: %#v", contract)
		}
	}
}

func TestServiceContractID(t *testing.T) {
	if got := ServiceContractID("com.example", "UserService"); got != "grpc::com.example.UserService/*" {
		t.Fatalf("ServiceContractID(with package) = %q", got)
	}
	if got := ServiceContractID("", "UserService"); got != "grpc::UserService/*" {
		t.Fatalf("ServiceContractID(without package) = %q", got)
	}
}

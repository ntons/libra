# envoy_service_auth_v3

Copy from https://github.com/envoyproxy/go-control-plane@v0.9.9/envoy/service/auth/v3/\*.go

```
diff --git a/envoy/service/auth/v3/external_auth.pb.go b/envoy/service/auth/v3/external_auth.pb.go
index dbd400cc..fe1db510 100755
--- a/envoy/service/auth/v3/external_auth.pb.go
+++ b/envoy/service/auth/v3/external_auth.pb.go
@@ -648,7 +648,7 @@ func (*UnimplementedAuthorizationServer) Check(context.Context, *CheckRequest) (
        return nil, status1.Errorf(codes.Unimplemented, "method Check not implemented")
 }
 
-func RegisterAuthorizationServer(s *grpc.Server, srv AuthorizationServer) {
+func RegisterAuthorizationServer(s *grpc.ServiceRegistrar, srv AuthorizationServer) {
        s.RegisterService(&_Authorization_serviceDesc, srv)
 }
```




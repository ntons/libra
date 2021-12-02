# envoy_service_auth_v3

Copy from https://github.com/envoyproxy/go-control-plane@v0.9.9/envoy/service/auth/v3/\*.go

```
651,652c651,652
< func RegisterAuthorizationServer(s grpc.ServiceRegistrar, srv AuthorizationServer) {
<       s.RegisterService(&Authorization_ServiceDesc, srv)
---
> func RegisterAuthorizationServer(s *grpc.Server, srv AuthorizationServer) {
>       s.RegisterService(&_Authorization_serviceDesc, srv)
673c673
< var Authorization_ServiceDesc = grpc.ServiceDesc{
---
> var _Authorization_serviceDesc = grpc.ServiceDesc{
```


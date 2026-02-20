- From otel collector contrib v0.146.0

- Update to go 1.25.7 to fix vulnerability
```
Vulnerability #1: GO-2025-4010
    Insufficient validation of bracketed IPv6 hostnames in net/url
  More info: https://pkg.go.dev/vuln/GO-2025-4010 
  Standard library
    Found in: net/url@go1.25
    Fixed in: net/url@go1.25.2
    Example traces found:
      #1: sharedcomponent.go:71:15: sharedcomponent.SharedComponent.Shutdown calls sync.Once.Do, which eventually calls url.Parse
```

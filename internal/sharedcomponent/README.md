Sharedcomponent from opentelemetry-collector-contrib v0.143.0

Update to go 1.25.5 to fix vulncheck

```
Vulnerability #1: GO-2025-4011
    Parsing DER payload can cause memory exhaustion in encoding/asn1
  More info: https://pkg.go.dev/vuln/GO-2025-4011
  Standard library
    Found in: encoding/asn1@go1.25
    Fixed in: encoding/asn1@go1.25.2
    Example traces found:
Error:       #1: sharedcomponent.go:71:15: sharedcomponent.SharedComponent.Shutdown calls sync.Once.Do, which eventually calls asn1.Unmarshal

Vulnerability #2: GO-2025-4010
    Insufficient validation of bracketed IPv6 hostnames in net/url
  More info: https://pkg.go.dev/vuln/GO-2025-4010
  Standard library
    Found in: net/url@go1.25
    Fixed in: net/url@go1.25.2
    Example traces found:
Error:       #1: sharedcomponent.go:71:15: sharedcomponent.SharedComponent.Shutdown calls sync.Once.Do, which eventually calls url.Parse

Vulnerability #3: GO-2025-4009
    Quadratic complexity when parsing some invalid inputs in encoding/pem
  More info: https://pkg.go.dev/vuln/GO-2025-4009
  Standard library
    Found in: encoding/pem@go1.25
    Fixed in: encoding/pem@go1.25.2
    Example traces found:
Error:       #1: sharedcomponent.go:71:15: sharedcomponent.SharedComponent.Shutdown calls sync.Once.Do, which eventually calls pem.Decode

Vulnerability #4: GO-2025-4007
    Quadratic complexity when checking name constraints in crypto/x509
  More info: https://pkg.go.dev/vuln/GO-2025-4007
  Standard library
    Found in: crypto/x509@go1.25
    Fixed in: crypto/x509@go1.25.3
    Example traces found:
Error:       #1: sharedcomponent.go:71:15: sharedcomponent.SharedComponent.Shutdown calls sync.Once.Do, which eventually calls x509.CertPool.AppendCertsFromPEM
Error:       #2: sharedcomponent.go:71:15: sharedcomponent.SharedComponent.Shutdown calls sync.Once.Do, which eventually calls x509.ParseCertificate
```

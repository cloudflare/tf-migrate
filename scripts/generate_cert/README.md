# Test Certificate Generation

This directory contains a script to generate valid X.509 certificates for integration testing.

## Background

The integration test fixtures require valid X.509 certificates to work with the Cloudflare API. Previously, the test files used a truncated/invalid certificate that only worked for testing migration logic, but failed when used with actual API calls.

## Generated Certificate

The current test certificate was generated on 2025-12-19 and has the following properties:

- **Subject**: `C=US, ST=CA, L=San Francisco, O=Test-Integration, CN=integration-test.example.com`
- **Issuer**: Self-signed (same as subject)
- **Serial Number**: 1766173443 (0x6945ab03)
- **Valid From**: Dec 19 19:44:03 2025 GMT
- **Valid Until**: Dec 19 19:44:03 2026 GMT
- **Key Size**: 2048-bit RSA
- **Is CA**: Yes
- **Key Usage**: Key Encipherment, Digital Signature, Certificate Sign
- **Extended Key Usage**: Server Authentication

## Generating a New Certificate

To generate a new test certificate:

```bash
cd tf-migrate/scripts
go run generate_test_cert.go > /tmp/new_cert.pem
```

This will create a valid X.509 certificate that can be used in the integration test fixtures.

## Updating Test Fixtures

To update the integration test fixtures with a new certificate:

1. Generate a new certificate (as shown above)
2. Update the `test_cert` variable in:
   - `integration/v4_to_v5/testdata/zero_trust_access_mtls_certificate/input/zero_trust_access_mtls_certificate.tf`
3. Update the certificate in the state file:
   - `integration/v4_to_v5/testdata/zero_trust_access_mtls_certificate/input/terraform.tfstate`
4. Re-run the migration to update expected outputs:
   ```bash
   rm -rf /tmp/test && mkdir -p /tmp/test
   cp -r integration/v4_to_v5/testdata/zero_trust_access_mtls_certificate/input/* /tmp/test/
   ./tf-migrate --config-dir /tmp/test --state-file /tmp/test/terraform.tfstate migrate
   cp /tmp/test/* integration/v4_to_v5/testdata/zero_trust_access_mtls_certificate/expected/
   ```
5. Run tests to verify:
   ```bash
   make test
   ```

## Python Helper Script

A Python script is available to automate updating certificates in JSON state files:

```python
#!/usr/bin/env python3
import json
import sys

# Read the new certificate
with open('/tmp/new_cert.txt', 'r') as f:
    new_cert = f.read().strip()

# Read and update the state file
state_file = sys.argv[1]
with open(state_file, 'r') as f:
    state = json.load(f)

for resource in state.get('resources', []):
    for instance in resource.get('instances', []):
        if 'attributes' in instance and 'certificate' in instance['attributes']:
            instance['attributes']['certificate'] = new_cert

with open(state_file, 'w') as f:
    json.dump(state, f, indent=2)
```

## Important Notes

- The certificate is self-signed and intended for testing only
- Each certificate is valid for 1 year from generation
- The certificate includes CA capabilities for maximum compatibility
- The certificate uses standard test values (Test-Integration organization, integration-test.example.com domain)

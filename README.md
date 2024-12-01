# Cloudflare DDNS
[![GitHub Release](https://img.shields.io/github/v/release/humweb/cloudflare-ddns)](https://github.com/humweb/cloudflare-ddns/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/humweb/cloudflare-ddns.svg)](https://pkg.go.dev/github.com/humweb/cloudflare-ddns)
[![go.mod](https://img.shields.io/github/go-mod/go-version/humweb/cloudflare-ddns)](go.mod)
[![LICENSE](https://img.shields.io/github/license/humweb/cloudflare-ddns)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/humweb/cloudflare-ddns/build.yml?branch=main)](https://github.com/humweb/cloudflare-ddns/actions?query=workflow%3Abuild+branch%3Amain)
[![Go Report Card](https://goreportcard.com/badge/github.com/humweb/cloudflare-ddns)](https://goreportcard.com/report/github.com/humweb/cloudflare-ddns)
[![Codecov](https://codecov.io/gh/humweb/cloudflare-ddns/branch/main/graph/badge.svg)](https://codecov.io/gh/humweb/cloudflare-ddns)

Add and update your cloudflare records with your dynamic ip.

### Generate secret
Update `config.json` file with your cloudflare API key, zone id, and records.
Then generate the kubernetes secret with the following command.
```bash
./k8s/generate-secret.sh

# Apply the secret
kubectl apply -f ./k8s/config-cf-ddns-Secret.yaml
```

### DDNS Deployment

```bash
kubectl apply -f ./k8s/cf-ddns.yaml
```


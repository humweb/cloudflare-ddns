# Cloudflare DDns
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


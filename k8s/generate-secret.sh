kubectl create secret generic config-cf-ddns --from-file=config.json --dry-run=client -oyaml -n ddns > ./k8s/config-cf-ddns-Secret.yaml

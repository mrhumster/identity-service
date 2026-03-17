# deploy to local k8s for dev

## namespace

```bash
kubectl create namespace go-app
```

## cert-manager

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.19.4/cert-manager.yaml
```

## issuer and ca-secret

```bash
kubectl apply -f ~/projects/identity-service/deploy/k8s/cert-manger/issuer.yaml
kubectl apply -f ~/projects/identity-service/deploy/k8s/cert-manger/ca-secret.yaml
```

## ingress controller

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/cloud/deploy.yaml
```

## postgres

```bash
helm -n go-app install postgresql oci://registry-1.docker.io/bitnamicharts/postgresql -f ~/projects/identity-service/deploy/k8s/postgres/values.yaml
```

## redis for casbin

```bash
helm install casbin-redis oci://registry-1.docker.io/bitnamicharts/redis --namespace go-app --set architecture=standalone --set auth.enabled=true --set auth.password=password --set master.persistence.enabled=false
```

## minio

```bash
kubectl apply -f ~/projects/stream-service/deploy/k8s/minio/.
$(eval MINIO_POD=$(shell kubectl get pods -n $(NAMESPACE) -l app=minio -o jsonpath='{.items[0].metadata.name}'))
kubectl exec -n $(NAMESPACE) $(MINIO_POD) -- /bin/sh -c "\
                mc alias set local http://localhost:9000 $(MINIO_ROOT_USER) $(MINIO_ROOT_PASS) && \
                mc admin user add local $(MINIO_USER_KEY) $(MINIO_USER_SECRET) || true && \
                mc admin policy attach local readwrite --user=$(MINIO_USER_KEY) && \
                mc mb local/$(MINIO_BUCKET) || true"
```

## auth service

```bash
kubectl apply -f ~/projects/identity-service/deploy/k8s/base/secret.yaml
kubectl apply -f ~/projects/identity-service/deploy/k8s/base/ingress-class.yaml
kubectl apply -f ~/projects/identity-service/deploy/k8s/base/deployment.yml
kubectl apply -f ~/projects/identity-service/deploy/k8s/base/service.yml
kubectl apply -f ~/projects/identity-service/deploy/k8s/scaling/hpa.yaml
```

## stream-service

```bash
kubectl apply -f ~/projects/stream-service/deploy/k8s/base/.
kubectl apply -f ~/projects/stream-service/deploy/k8s/scaling/.
```

## transcoder-service

```bash
kubectl apply -f ~/projects/transcoder-service/deploy/k8s/transcoder/.
```

## front

```bash
kubectl apply -f ~/projects/stream-service-web/k8s/.
```

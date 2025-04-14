# Runtime Engine  

## 核心功能
1. 统一Metrics
2. 支持组内拓扑发现
3. 支持加载/卸载LoRA

## 统一Metrics
```bash
kubectl apply -f samples/runtime/runtime-metric.yaml
```
获取Metrics
```bash
POD_NAME=`kubectl get po --no-headers -ocustom-columns=:metadata.name|grep runtime-metric-example`
kubectl port-forward $POD_NAME 8080:8080

curl http://localhost:8080/metrics
```

## 组内拓扑发现
```bash
kubectl apply -f samples/pd-disagg/llingjun.yaml
```
查看组内拓扑结构
```bash
kubectl exec -it pd-schedule-scheduler-0  -c scheduler -- cat /etc/patio/instance-config.yaml
```
预期输出
```text
- endpoint: 10.198.210.73:8000
- endpoint: 10.201.246.127:8000
- endpoint: 10.82.36.206:8000
- endpoint: 10.198.210.74:8000
```
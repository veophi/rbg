# Prefill-Decode Disaggrating

## Dynamo
### Deploy
至少需要3*A10节点 + 一个8c16g节点。部署一个Processor，一个Decoder，2个Prefill
```
kubectl apply -f samples/pd-disagg/Dynamo.yaml 

kubectl apply -f- <<EOF
apiVersion: v1
kind: Service
metadata:
  name: dynamo-service
spec:
  type: ClusterIP
  ports:
  - port: 8000
    protocol: TCP
    targetPort: 8000
  selector:
    rolebasedgroup.workloads.x-k8s.io/role: processor
 EOF
```
### Send Requests
```bash
kubectl port-forward svc/dynamo-service 8000:8000
# 发送两次请求，请求分别在两个pod中处理
curl localhost:8000/v1/chat/completions   -H "Content-Type: application/json"   -d '{
    "model": "qwen",
    "messages": [
    {
        "role": "user",
        "content": "Tell me about the absolute legend Roger Federer"
    }
    ],
    "stream":false,
    "max_tokens": 30
  }'

```
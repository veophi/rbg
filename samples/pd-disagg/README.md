# Prefill-Decode Disaggrating

## 1. Prepare
Model: Qwen2.5-7B-Instruct  
Nodes:
- cpu: 3 * 8vCPU 32GB。 推荐机型 ecs.g8i.xlarge
- gpu: 3 * A10 (DRAM 1x24GB)。推荐机型 ecs.gn7i-c16g1.4xlarge


### Model Download
修改samples/model-download/model-download-job.yaml文件，下载模型并创建pv&pvc  
```bash
kubectl apply -f samples/model-download/model-download-job.yaml 
```
## 2. LingJun
### Deploy
```bash
kubectl create configmap scheduler-run --from-file=samples/pd-disagg/scheduler-run.sh

kubectl apply -f- <<EOF
apiVersion: v1
kind: Service
metadata:
  name: lingjun-service
spec:
  type: ClusterIP
  ports:
  - port: 8008
    protocol: TCP
    targetPort: 8008
  selector:
    rolebasedgroup.workloads.x-k8s.io/name: lingjun-pd
    rolebasedgroup.workloads.x-k8s.io/role: scheduler
EOF
```
### Send Requests
```bash
kubectl port-forward svc/lingjun-service 8008:8008

curl http://localhost:8008/v1/completions -H "Content-Type: application/json" -d '{
"model": "/models/Qwen2.5-7B-Instruct/", 
"prompt": "San Francisco is a",
"max_tokens": 30, 
"temperature": 0
}'
```
预期输出
```text
{"id":"cmpl-697a57281bcc40c5b5dadab26ffb7d32","object":"text_completion","created":1744612537,"model":"/models/Qwen2.5-7B-Instruct/","choices":[{"index":0,"text":" city of neighborhoods, each with its own distinct character and charm. From the historic Haight-Ashbury to the trendy Mission District, there's something","logprobs":null,"finish_reason":"length","stop_reason":null,"prompt_logprobs":null}],"usage":{"prompt_tokens":4,"total_tokens":34,"completion_tokens":30,"prompt_tokens_details":null}}
```
查看scheduler日志
```bash
kubectl logs lingjun-pd-scheduler-0

# 预期输出
2025-04-14 06:35:37.5412 Completion: received vllm request , scheduler request: f467d137-1355-48dd-8a15-d79e51ff8974
2025-04-14 06:35:37.5412 schedule request: , scheduler request id: f467d137-1355-48dd-8a15-d79e51ff8974, input: 4, output: 30, prefill instance: 10.96.142.112:8000, decode instance: 10.5.125.3:8000, serve mode: SplitWise
2025-04-14 06:35:38.5606 finish prefill stage of request: f467d137-1355-48dd-8a15-d79e51ff8974
2025-04-14 06:35:38.5607 connection of vllm request: , scheduler request: f467d137-1355-48dd-8a15-d79e51ff8974 closed without finish
```
## 3. Dynamo
### Deploy
```bash
kubectl apply -f samples/pd-disagg/dependency/etcd.yaml
kubectl apply -f samples/pd-disagg/dependency/nats.yaml

kubectl apply -f samples/pd-disagg/dynamo.yaml

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
    rolebasedgroup.workloads.x-k8s.io/name: dynamo-pd
    rolebasedgroup.workloads.x-k8s.io/role: processor
EOF
```
### Send Requests
```bash
kubectl port-forward svc/dynamo-service 8000:8000

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
预期返回
```text
{"id":"2c0da9d2-d472-48c2-999e-aabf0fa2d97b","choices":[{"index":0,"message":{"content":"Roger Federer is widely regarded as one of the greatest tennis players of all time, known for his exceptional talent, grace, and sportsmanship both on","refusal":null,"tool_calls":null,"role":"assistant","function_call":null,"audio":null},"finish_reason":"length","logprobs":null}],"created":1744602609,"model":"qwen","service_tier":null,"system_fingerprint":null,"object":"chat.completion","usage":null}
```
### kv cache aware路由
发送5次相同请求
```bash
for i in {1..5}
do
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
done
```
查看decode日志
```bash
kubectl logs dynamo-pd-decoder-0 |grep "Prefix cache hit rate"
```
预期输出
```text
INFO 04-14 05:42:50 metrics.py:471] Prefix cache hit rate: GPU: 0.00%, CPU: 0.00%
INFO 04-14 05:42:59 metrics.py:471] Prefix cache hit rate: GPU: 50.00%, CPU: 0.00%
INFO 04-14 05:43:04 metrics.py:471] Prefix cache hit rate: GPU: 75.00%, CPU: 0.00%
INFO 04-14 05:43:17 metrics.py:471] Prefix cache hit rate: GPU: 83.33%, CPU: 0.00%
INFO 04-14 05:43:27 metrics.py:471] Prefix cache hit rate: GPU: 83.33%, CPU: 0.00%
```
可以从日志中看到Prefix cache hit

## Mooncake
### Deploy
```bash
# 安装etcd
kubectl apply -f samples/pd-disagg/dependency/etcd.yaml

kubectl apply -f samples/pd-disagg/mooncake.yaml

kubectl apply -f- <<EOF
apiVersion: v1
kind: Service
metadata:
  name: mooncake-service
spec:
  type: ClusterIP
  ports:
  - port: 8000
    protocol: TCP
    targetPort: 8000
  selector:
    rolebasedgroup.workloads.x-k8s.io/name: mooncake-pd
    rolebasedgroup.workloads.x-k8s.io/role: scheduler
EOF
```
启动scheduler
```bash
kubectl exec -it mooncake-pd-scheduler-0 -- bash
# TODO 改为runtime container自动获取
python3 disagg_proxy_demo.py --model /models/Qwen2.5-7B-Instruct/ --prefill 10.96.142.112:8000 --decode 10.5.125.1:8000 --port 8000
```
### Send Requests
```bash
kubectl port-forward svc/mooncake-service 8000:8000

curl http://localhost:8000/v1/completions -H "Content-Type: application/json" -d '{
"model": "/models/Qwen2.5-7B-Instruct/", 
"prompt": "San Francisco is a",
"max_tokens": 30, 
"temperature": 0
}'
```
预期返回
```text
{"id":"cmpl-28ca01f953a34c629731fc26266b226c","object":"text_completion","created":1744631891,"model":"/models/Qwen2.5-7B-Instruct/","choices":[{"index":0,"text":" city of neighborhoods, each with its own distinct character and charm. From the historic Haight-Ashbury to the trendy Mission District, there's something","logprobs":null,"finish_reason":"length","stop_reason":null,"prompt_logprobs":null}],"usage":{"prompt_tokens":4,"total_tokens":34,"completion_tokens":30,"prompt_tokens_details":null}}
```
查看decode日志
```bash
kubectl logs mooncake-pd-decode-0 |grep "skip model forwarding"
```
预期输出
```text
INFO 04-14 11:58:11 [engine.py:310] Added request cmpl-28ca01f953a34c629731fc26266b226c-0.
DEBUG 04-14 11:58:11 [mooncake_store_connector.py:202] [rank0]: Successfully received all KVs and hidden states, skip model forwarding.
INFO 04-14 11:58:11 [metrics.py:488] Avg prompt throughput: 0.5 tokens/s, Avg generation throughput: 0.1 tokens/s, Running: 1 reqs, Swapped: 0 reqs, Pending: 0 reqs, GPU KV cache usage: 0.0%, CPU KV cache usage: 0.0%.
INFO:     10.96.142.111:41758 - "POST /v1/completions HTTP/1.1" 200 OK
```



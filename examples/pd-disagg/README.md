# PD分离推理

## 前提条件
1. 已创建包含GPU的Kubernetes集群。具体操作，请参见[创建GPU集群](https://help.aliyun.com/zh/ack/ack-managed-and-ack-dedicated/user-guide/create-an-ack-managed-cluster-with-gpu-accelerated-nodes?spm=a2c4g.11186623.0.0.7ffc73dduiOGpt)。
推荐配置：
- cpu: 3 * 8vCPU 32GB。 推荐机型 ecs.g8i.xlarge
- gpu: 3 * A10 (DRAM 1x24GB)。推荐机型 ecs.gn7i-c16g1.4xlarge

2. 下载模型并创建PVC
   1) 修改samples/model-download/model-download-job.yaml，替换oss bucket及AK等信息
   2) 执行以下命令下载模型并创建PV，PVC
   ```bash
   kubectl apply -f samples/model-download/model-download-job.yaml
   ```

## 部署sglang+mooncake PD分离推理服务
1) 构建镜像
参考[文档](https://github.com/kvcache-ai/Mooncake/blob/main/doc/en/sglang-integration-v1.md)构建SGLang+Mooncake镜像，
并将[sglang_mooncake.yaml](sglang_mooncake.yaml)中`<your-sglang-mooncake-image>`替换为建出的镜像地址
2) 执行以下命令部署PD分离推理服务
```bash
kubectl apply -f ./sglang_mooncake.yaml 
```
3) 发送请求
```bash
kubectl port-forward svc/pd-service 8008:8008

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



## 部署vLLM+mooncake PD分离推理服务

1. 构建镜像
参考[文档](https://github.com/kvcache-ai/Mooncake/blob/main/doc/en/vllm-integration-v1.md)构建vLLM+Mooncake镜像，
并将[vllm_mooncake.yaml](vllm_mooncake.yaml)中`<your-vllm-mooncake-image>`替换为建出的镜像地址

2. 安装依赖
   1. 部署etcd服务
   2. 安装推理运行时patio-runtime
      ```bash
      kubectl apply -f examples/runtime/patio-runtime.yaml
      ```

3. 部署PD分离推理服务
```bash
kubectl apply -f vllm_mooncake.yaml
```

4. 发送请求
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

## 3部署Dynamo PD分离推理服务
1. 参考[文档](https://github.com/ai-dynamo/dynamo/blob/main/README.md#building-the-dynamo-base-image)构建Dynamo镜像,
并将[dynamo.yaml](dynamo.yaml)中`<your-dynamo-image>`替换为建出的镜像地址

2. 安装依赖
   1. 部署etcd服务
   2. 部署nats服务
   
3. 部署Dynamo PD分离部署
```bash
kubectl apply -f samples/pd-disagg/dynamo.yaml
```

4. 发送请求
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


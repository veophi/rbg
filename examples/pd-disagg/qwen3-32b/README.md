# PD分离推理

## 前提条件

1. 已创建包含GPU的Kubernetes集群。具体操作，请参见[创建GPU集群](https://help.aliyun.com/zh/ack/ack-managed-and-ack-dedicated/user-guide/create-an-ack-managed-cluster-with-gpu-accelerated-nodes?spm=a2c4g.11186623.0.0.7ffc73dduiOGpt)。
2. 下载Qwen3-32B模型文件
   1. 执行以下命令从ModelScope下载Qwen3-32B模型。
   ```shell
   git lfs install
   GIT_LFS_SKIP_SMUDGE=1 git clone git clone https://www.modelscope.cn/Qwen/Qwen3-32B.git
   cd Qwen3-32B/
   git lfs pull
   ```
   2. 在OSS中创建目录，将模型上传至OSS。
   ```shell
   ossutil mkdir oss://<your-bucket-name>/models/Qwen3-32B
   ossutil cp -r ./Qwen3-32B oss://<your-bucket-name>/models/DeepSeek-R1-Distill-Qwen-7B
   ```
   3. 创建PV和PVC。为目标集群配置名为llm-model的存储卷PV和存储声明PVC。
   ```shell
   # 替换yaml中的OSS相关变量
   kubectl apply -f model.yaml
   ```
   
## 部署Dynamo PD分离推理服务

1. 构建镜像
    1. 参考[文档](https://github.com/ai-dynamo/dynamo/blob/main/README.md#building-the-dynamo-base-image)构建Dynamo镜像
    2. 修改examples中配置文件
   ```Dockerfile
   # localhost/dynamo:latest 替换为您构建出的dynamo镜像
   FROM localhost/dynamo:latest 
   COPY processor_router.py /workspace/examples/llm/graphs/
   ```
    3. 将[dynamo.yaml](dynamo.yaml)中`<your-dynamo-image>`替换为建出的镜像地址

2. 安装依赖
    1. 部署etcd服务
    2. 部署nats服务

3. 部署Dynamo PD分离部署服务

```bash
kubectl create configmap dynamo-configs --from-file=./qwen3.yaml
kubectl apply -f ./dynamo-rbg.yaml
```

4. 验证推理服务

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

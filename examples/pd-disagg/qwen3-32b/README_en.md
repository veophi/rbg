# Prefill-Decode-Disaggrating Inference

## Prerequisites

1. An ACK cluster that contains GPU-accelerated nodes is created. For more information, see [Create an ACK cluster with GPU-accelerated nodes](https://www.alibabacloud.com/help/en/ack/ack-managed-and-ack-dedicated/user-guide/create-an-ack-managed-cluster-with-gpu-accelerated-nodes?spm=a2c63.p38356.0.i5).

2. Prepare the Qwen3-32B model files 
   1. Run the following command to download the Qwen3-32B model from ModelScope:
   ```shell
   git lfs install
   GIT_LFS_SKIP_SMUDGE=1 git clone git clone https://www.modelscope.cn/Qwen/Qwen3-32B.git
   cd Qwen3-32B/
   git lfs pull
   ```
   2. Create an Object Storage Service (OSS) directory and upload the model files to the directory.
   ```shell
   ossutil mkdir oss://<your-bucket-name>/models/Qwen3-32B
   ossutil cp -r ./Qwen3-32B oss://<your-bucket-name>/models/Qwen3-32B
   ```
   3. Create a persistent volume (PV) and a persistent volume claim (PVC). Create a PV named llm-model and a PVC in the cluster. 
   ```shell
   # replace OSS variables in model.yaml
   kubectl apply -f model.yaml
   ```

## Deploy Dynamo Inference Service

1. Build Image
    1. Build the Dynamo image according to the [document](https://github.com/ai-dynamo/dynamo/blob/main/README.md#building-the-dynamo-base-image).
    2. Modify files to support deploying the processor component independently
   ```Dockerfile
   # replace localhost/dynamo:latest with the built image
   FROM localhost/dynamo:latest 
   COPY processor_router.py /workspace/examples/llm/graphs/
   ```
    3. Replace `<your-dynamo-image>` with the built image in [dynamo-rbg.yaml](dynamo-rbg.yaml)

2. Install dependencies
   1. Deploy etcd service
   2. Deploy NATS service

3. Deploy PD Disaggregation with Dynamo
```bash
kubectl create configmap dynamo-configs --from-file=./qwen3.yaml
kubectl apply -f dynamo-rbg.yaml
```

4. Verify the inference service

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

Expected output:
```text
{"id":"2c0da9d2-d472-48c2-999e-aabf0fa2d97b","choices":[{"index":0,"message":{"content":"Roger Federer is widely regarded as one of the greatest tennis players of all time, known for his exceptional talent, grace, and sportsmanship both on","refusal":null,"tool_calls":null,"role":"assistant","function_call":null,"audio":null},"finish_reason":"length","logprobs":null}],"created":1744602609,"model":"qwen","service_tier":null,"system_fingerprint":null,"object":"chat.completion","usage":null}
```

# SGLang Prefill-Decode-Disaggregating Inference

![](img/sgl-sequence.png)

## Prerequisites

1. A Kubernetes cluster with version >= 1.26 is Required, or it will behave unexpected.
2. Kubernetes cluster has at least 6+ CPUs with at least 32G VRAM available for the LLM Inference to run on.
3. The kubectl command-line tool has communication with your cluster. Learn how
   to [install the Kubernetes tools](https://kubernetes.io/docs/tasks/tools/).
4. Prepare the Qwen3-32B model files
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
    3. Create a persistent volume (PV) and a persistent volume claim (PVC). Create a PV named llm-model and a PVC in the
       cluster.
   ```shell
   # replace OSS variables in model.yaml
   kubectl apply -f model.yaml
   ```

## Deploy SGLang Inference Service

1. Deploy PD Disaggregation Service
   ![](img/sgl.png)

```bash
kubectl apply -f ./sglang-pd.yaml
```

2. Verify the inference service

```bash
kubectl port-forward svc/sglang-pd 8000:8000

curl http://localhost:8000/v1/chat/completions -H "Content-Type: application/json"  -d '{"model": "/models/Qwen3-32B", "messages": [{"role": "user", "content": "测试一下"}], "max_tokens": 30, "temperature": 0.7, "top_p": 0.9, "seed": 10}'
```

Expected output:

```text
{"id":"29f3fdac693540bfa7808fc1a8701758","object":"chat.completion","created":1753695366,"model":"/models/Qwen3-32B","choices":[{"index":0,"message":{"role":"assistant","content":"<think>\n好的，用户让我测试一下，我需要先确认他们的具体需求。可能他们想测试我的功能，比如回答问题、生成内容","reasoning_content":null,"tool_calls":null},"logprobs":null,"finish_reason":"length","matched_stop":null}],"usage":{"prompt_tokens":10,"total_tokens":40,"completion_tokens":30,"prompt_tokens_details":null}}
```

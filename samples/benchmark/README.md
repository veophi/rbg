## LLM Benchmark Tool
LLM Benchmark Tool is a tool for benchmarking LLM models. It is used to compare the performance of different LLM models.


### Deploy
1. 下载模型文件（如果已经有pv&pvc，可以跳过）  
修改samples/model-download/model-download-job.yaml文件，下载模型并创建pv&pvc
```bash
kubectl apply -f samples/model-download/model-download-job.yaml 
```
2. 部署压测Pod  
```bash
kubectl apply -f samples/benchmark/benchmark.yaml
```
3. 下载数据集
```bash
# 执行以下命令进入Benchmark Pod
PODNAME=$(kubectl get po -o custom-columns=":metadata.name"|grep "vllm-benchmark")
kubectl exec -it $PODNAME -c vllm-benchmark -- bash

# 下载压测数据集
pip3 install modelscope --index-url=http://mirrors.aliyun.com/pypi/simple/ --trusted-host=mirrors.aliyun.com
modelscope download --dataset gliang1001/ShareGPT_V3_unfiltered_cleaned_split ShareGPT_V3_unfiltered_cleaned_split.json --local_dir /root/
```
4. 执行压测
- vllm benchmark
```bash
# 执行压测 input_length=1024,tp=4,output_lenght=512,concurrency=8,num_prompts=80
python3 /root/vllm/benchmarks/benchmark_serving.py \
        --backend vllm \
        --model /mount/model/Qwen2.5-72B-Instruct-AWQ \
        --served-model-name /mount/model/Qwen2.5-72B-Instruct-AWQ \
        --trust-remote-code \
        --dataset-name random \
        --dataset-path /root/ShareGPT_V3_unfiltered_cleaned_split.json \
        --random-input-len 1024 \
        --random-output-len 512 \
        --random-range-ratio 0.8 \
        --num-prompts 80 \
        --max-concurrency 8 \
        --host lingjun-service \
        --port 8008 \
        --endpoint /v1/completions \
        --save-result \
        2>&1 | tee benchmark_serving.txt
```
- sglang benchmark
```bash
# 执行压测 input_length=1024,tp=4,output_lenght=512,concurrency=8,num_prompts=80
python3 -m sglang.bench_serving \
        --backend vllm \
        --model /models/Qwen2.5-7B-Instruct/ \
        --host 0.0.0.0 \
        --port 8000 \
        --dataset-name random \
        --max-concurrency 8 \
        --num-prompt 80 \
        --random-input 1024 \
        --random-output 512 \
        --random-range-ratio 1.0 \
        --dataset-path /root/ShareGPT_V3_unfiltered_cleaned_split.json > benchmark-vllm-result.txt
```

- 多轮对话
```shell
python3 /root/vllm/benchmarks/mutil-round-qa/multi-round-qa.py \
    --num-users 10 \
    --num-rounds 5 \
    --qps 0.5 \
    --shared-system-prompt 1000 \
    --user-history-prompt 2000 \
    --answer-len 100 \
    --model /root/Qwen2.5-0.5B \
    --base-url http://192.168.72.121:8008/v1
```
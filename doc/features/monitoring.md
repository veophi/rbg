# Monitoring

## Prerequisites

- A Kubernetes cluster with version >= 1.26 is Required, or it will behave unexpected.
- Install Prometheus
- Install Grafana
- Add the label `alibabacloud.com/inference-workload=xxx` for RBG instances

## Collect inference engine monitoring metrics

```bash
kubectl apply -f examples/monitoring/podmonitor.yaml
```

## Grafana

Import the inference Grafana dashboard
by [vLLM Grafana JSON](https://docs.vllm.ai/en/latest/getting_started/examples/prometheus_grafana.html#example-materials)
![](../img/vLLM-Dashboards-Grafana.png)

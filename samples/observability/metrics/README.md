# Metrics

## 前提条件
1. 安装Prometheus组件
2. 创建Grafana服务（！！不能使用共享版Grafana，共享版不支持导入仪表盘）

## 接入metrics
```bash
kubectl apply -f samples/observability/metrics/podmonitor.yaml
```

## Grafana
1. Grafana JSON 参考[文档](https://docs.vllm.ai/en/latest/getting_started/examples/prometheus_grafana.html#example-materials)  
具体导入Grafana操作，请参见[如何导出和导入Grafana仪表盘](https://help.aliyun.com/zh/grafana/support/how-to-export-and-import-the-grafana-dashboard?spm=a2c4g.11186623.0.0.567473ddyBRfv4)。  
<img src="./vLLM-Dashboards-Grafana.png">
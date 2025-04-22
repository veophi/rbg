# RoleBasedGroup API 中文文档

RoleBasedGroup：用于编排多角色协作分布式工作负载服务的 API，专注于解决 AI/ML 推理工作负载的常见部署模式。特别适用于Prefill/Decode引擎解耦场景（如prefill, decode和scheduler等角色），支持大语言模型（LLM）跨多节点设备的分布式运行。

## 📖 概述

### 背景
传统 Kubernetes 有状态集合（StatefulSet）在分布式有状态服务场景下面临多角色协调难题。本方案重点解决：
- 角色间启动顺序依赖  
- 跨角色服务发现复杂  
- 配置管理碎片化  

### 🧩 核心特性
✨ **多角色模板定义** - 将分布式有状态工作负载建模为统一 K8s 工作负载组  
🔗 **基于角色的启动控制** - 为 RoleBasedGroup 中的 ReplicatedJobs 建立角色依赖关系和启动序列  
🔍 **自动服务发现** - 通过配置文件和环境变量注入拓扑细节  
⚡ **弹性伸缩** - 支持工作组/角色级伸缩操作  
🔄 **原子化滚动更新** - 角色级更新：以角色为单元顺序升级（同一角色内所有 Pod 同步更新）  
🌐 **拓扑感知调度** - 保障工作组/角色内 Pod 在同一拓扑域共置  
🛑 **原子化故障恢复** - 同一工作组/角色内任意 Pod/容器故障时触发全角色重建  
🔧 **可定制工作负载** - 支持多种工作负载类型（如 StatefulSet、Deployment 等）  

## 🏗 概念架构图

![](rbgs-concept.png)

## 🚀 快速入门

### 安装控制器
```bash
helm install rbgs deploy/helm/rbgs -n rbgs-system --create-namespace
```

### 最小化示例
```yaml
apiVersion: workloads.x-k8s.io/v1alpha1
kind: RoleBasedGroup
metadata:
  name: nginx-cluster
spec:
  roles:
      - role: prefill
        replicas: 2
        template: { ... }
      - role: decode
        replicas: 2
        dependencies: ["prefill"]
        template: { ... }
```

## 📚 API 文档

### 关键字段说明
| 字段 | 类型 | 描述 |
|-------|------|-------------|
| `startupPolicy` | string | 启动策略 (Ordered/Parallel) |
| `dependencies` | []string | 角色依赖列表 |
| `workload` | Object | 底层工作负载类型 (默认: StatefulSet) |

完整 API 规范：[API_REFERENCE.md]()

## 🤝 参与贡献
欢迎通过提交 Issue 和 PR 参与贡献！详见[贡献指南](CONTRIBUTING.md)

## 社区交流与支持

访问 [Kubernetes 社区页面]() 了解参与方式。

项目维护者联系方式：
- [Slack 频道]()
- [邮件列表]()

### 行为准则
参与 Kubernetes 社区需遵守 [Kubernetes 行为准则](code-of-conduct.md)。

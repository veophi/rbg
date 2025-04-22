# The RoleBasedGroup API 

RoleBasedGroup: An API for for orchestrating distributed workload services with multi-role collaboration and automated service discovery. It aims to address common deployment patterns of AI/ML inference workloads, especially Prefill/Decode engine disaggregation workloads (e.g. a prefill, decode, scheduler, etc.) where the LLM will be sharded and run across multiple devices on multiple nodes. 

## 📖 Overview

### Background
Traditional Kubernetes statefulset struggle with multi-role coordination in distributed stateful service scenarios. This solution addresses:
- Startup order dependencies between roles  
- Complex cross-role service discovery  
- Fragmented configuration management  

### 🧩 Key Features
  ✨ **Multi-role Template Spec** - Model distributed stateful workloads as unified K8s workload groups.  
  🔗 **Role-based Startup Control** - Establish role dependencies and startup sequence for ReplicatedJobs in a RoleBasedGroup.  
  🔍 **Auto Service Discovery** - Inject topology details via configs and env vars.  
  ⚡ **Elastic Scaling** - Enable group/role-level scaling operations.  
  🔄 **Atomic Rollout** - Group-level rollout/update: Upgrade entire groups sequentially as single units (all pods in a group updated simultaneously).  
  🌐 **Topology-aware Placement** - Guarantee co-location of group pods within the same topology domain.  
  🛑 **Atomic Failure Recovery** - Trigger full group recreation if any pod/container fails within the group.  
  🔧 **Customizable Workload** - Support for multiple workload types (e.g. StatefulSet, Deployment, etc.).  

## 🏗 Conceptual Diagram

![](rbgs-concept.png)

## 🚀 Quick Start

### Install CRD
```bash
kubectl apply -f rolebasedgroupsets.yaml
```

### Minimal Example
```yaml
apiVersion: openpatio.io/v1alpha1
kind: RoleBasedGroupSet
metadata:
  name: demo-group
spec:
  replicas: 2
  groupTemplate:
    roles:
      - role: prefill
        replicas: 2
        template: { ... }
      - role: decode
        replicas: 2
        dependencies: ["prefill"]
        template: { ... }
```


## 📚 API Documentation

### Key Fields
| Field | Type | Description |
|-------|------|-------------|
| `startupPolicy` | string | Startup strategy (Ordered/Parallel) |
| `dependencies` | []string | Role dependencies list |
| `workload` | Object | Underlying workload type (default: StatefulSet) |

Full API spec: [API_REFERENCE.md](docs/API_REFERENCE.md)

## 🤝 Contributing
We welcome contributions through issues and PRs! See [CONTRIBUTING.md](CONTRIBUTING.md)

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page]().

You can reach the maintainers of this project at:

- [Slack]()
- [Mailing List]()

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

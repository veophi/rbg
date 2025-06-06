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
  🔄 **Atomic Rollout** - Role-level rollout/update: Upgrade entire Roles sequentially as single units (all pods in the same role updated simultaneously).  
  🌐 **Topology-aware Placement** - Guarantee co-location of group/role pods within the same topology domain.  
  🛑 **Atomic Failure Recovery** - Trigger full role recreation if any pod/container fails within the same group/role.  
  🔧 **Customizable Workload** - Support for multiple workload types (e.g. StatefulSet, Deployment, etc.) for the role.  

## 🏗 Conceptual Diagram

![](doc/rbgs-concept.png)

## 🚀 Quick Start

### Install Controller
```bash
helm install rbgs deploy/helm/rbgs -n rbgs-system --create-namespace
```

### Minimal Example

```bash
kubectl apply -f examples/base/rbg.yaml
```


## 📚 API Documentation

### Key Fields
| Field | Type | Description |
|-------|------|-------------|
| `startupPolicy` | string | Startup strategy (Ordered/Parallel) |
| `dependencies` | []string | Role dependencies list |
| `workload` | Object | Underlying workload type (default: StatefulSet) |

Full API spec: [API_REFERENCE.md]()

## 🤝 Contributing
We welcome contributions through issues and PRs! See [CONTRIBUTING.md](doc/CONTRIBUTING.md)

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page]().

You can reach the maintainers of this project at:

- [Slack]()
- [Mailing List]()

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](doc/code-of-conduct.md).

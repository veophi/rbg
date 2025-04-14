# The RoleBasedGroupSet API 

A Kubernetes operator for orchestrating distributed stateful services with multi-role collaboration and automated service discovery.

## 📖 Overview

### Background
Traditional StatefulSets struggle with multi-role coordination in distributed stateful service scenarios. This solution addresses:
- Startup order dependencies between roles
- Complex cross-role service discovery
- Fragmented configuration management

### Core Capabilities
✨ **Multi-Role Orchestration** - Define role dependencies with ordered/parallel startup  
🔍 **Auto Service Discovery** - Inject topology info via config files and environment variables  
⚡ **Elastic Scaling** - Support group/role-level scaling (future granular scaling)  
📦 **Unified Configuration** - Dual injection via YAML and environment variables

## 🏗 Architecture

```mermaid
%%{init: {'theme': 'neutral'}}%%
flowchart TD
    RBGS[RoleBasedGroupSet CRD] -->|Manages| Groups
    Groups -->|Contains| Roles
    
    subgraph Group[Worker Group]
        direction TB
        GroupCtrl[Group Controller] -->|Creates| RoleResources
        RoleResources -->|For each Role| RoleStatefulSet[Role StatefulSet]
        RoleStatefulSet -->|Creates| Pods
        RoleStatefulSet -->|Bound to| RoleService[Role Headless Service]
    end
    
    Pods -->|Mounts| ConfigMap[Cluster ConfigMap]
    Pods -->|Reads| EnvVars[Environment Variables]
    
    RBGS -->|Status Reporting| K8sAPI[Kubernetes API]
    
    classDef cluster fill:#f9f9f9,stroke:#ddd
    classDef component fill:#e6f4ff,stroke:#4da6ff
    classDef data fill:#eaf7e6,stroke:#7ccf5c
    
    class RBGS,K8sAPI,GroupCtrl component
    class RoleStatefulSet,RoleService component
    class ConfigMap,EnvVars data
    class Groups,Roles,Pods cluster
```

**Key Components**:
- `RoleBasedGroupSet CRD`: Custom resource definition for declaring service groups
- `Worker Group`: Isolated unit containing multiple roles
- `Role StatefulSet`: Workload instance for each role
- `Headless Service`: DNS record provider for role instances
- `Config Injection`: Dual configuration through ConfigMap and environment variables

## 🚀 Quick Start

### Install CRD
```bash
kubectl apply -f https://raw.githubusercontent.com/yourorg/rolebasedgroupset/main/config/crd/bases/openpatio.io_rolebasedgroupsets.yaml
```

### Minimal Example
```yaml
apiVersion: openpatio.io/v1alpha1
kind: RoleBasedGroupSet
metadata:
  name: demo-group
spec:
  replicas: 2
  startupPolicy: Ordered
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

## 🧩 Key Features

### Coordinated Role Startup
```mermaid
graph TD
    GroupSet -->|Create| Group1
    Group1 -->|Sequential Startup| RoleA[prefill]
    RoleA -->|Ready| RoleB[decode]
```

### Service Discovery Mechanism
Automatically generates two configuration formats:

**1. Config File** (`/etc/rbgs/cluster.yaml`)
```yaml
cluster:
  local:
    role: "decode"
    rank: 0
  roles:
    prefill:
      endpoints:
        - address: "prefill-0.demo-group-prefill:8080"
```

**2. Environment Variables**
```bash
RBGS_ROLES_PREFILL_0_ADDRESS=prefill-0.demo-group-prefill:8080
RBGS_LOCAL_ROLE=decode
```

### Status Management
Real-time status monitoring:
```yaml
status:
  phase: Running
  readyGroups: 2/2
  groups:
    - groupId: "0"
      phase: Running
      roles:
        - role: prefill
          readyReplicas: 2
```

## 🔧 Advanced Configuration

### Cross-Group Communication
Expose roles via Service:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: cross-group-svc
spec:
  selector:
    patio.io/rbgs-role: scheduler
  ports:
    - port: 80
      targetPort: 8080
```

### Role Extension
Add new roles to existing groups:
```yaml
groupTemplate:
  roles:
    - role: postprocess
      replicas: 1
      dependencies: ["decode"]
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

## License

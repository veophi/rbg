# RoleBasedGroup Documentation

## TOC

- Overview
    - [Introduction](../README.md)
- Installation
    - [Kubectl](./install.md)
    - [Helm](./install.md)
- Key Features
    - [Multi Roles](features/multiroles.md)
    - [Autoscaling](features/autoscaler.md)
    - [Update Strategy](features/update-strategy.md)
    - [Failure Handling](features/failure-handling.md)
    - [Gang Scheduling](features/gang-scheduling.md)
    - [Monitoring](features/monitoring.md)
- Reference
    - [Labels, Annotations and Environment Variables](reference/variables.md)
    - [RoleBasedGroup API](reference/api.md)
- Examples
    - Deploying Inference Service
        - Single Node
            - [sglang](../examples/single-node/sglang.yaml)
            - Others Inference Engines
              - [vLLM](../examples/single-node/vllm.yaml)

        - Multi Node
            - [sglang](../examples/multi-nodes/sglang.yaml)
              - Others Inference Engines
                - [vLLM](../examples/multi-nodes/vllm.yaml)

        - PD-Disagg
            - [sglang](../examples/pd-disagg/sglang/sgl.md)
            - Others Inference Engines
              - [dynamo](../examples/pd-disagg/dynamo/README.md)

    - Advanced Features
        - Multi-roles
            - [Multirole with StatefulSet and Deployment](../examples/base/rbg-base.yaml)
            - [Multirole with LeaderWorkerSet](../examples/multi-nodes/sglang.yaml)
            - [Multirole with startup dependency](../examples/base/rbg-base.yaml)
        - Update Strategy
            - [Rolling Update](../examples/base/rolling-update.yaml)
        - Failure Handling
            - [Restart Policy](../examples/base/restart-policy.yaml)
        - Scheduling
            - [Gang Scheduling](../examples/base/gang-scheduling.yaml)
        - Monitoring
            - [Prometheus](features/monitoring.md)

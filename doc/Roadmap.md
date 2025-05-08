# Roadmap

## 2025 Q1

### Core Features

- [x] Implement RoleBasedGroup controller with basic features
- [x] e2e tests & unit tests
- [x] build image & helm deploy & release artifacts
- [ ] support conditions
- [ ] remove runtime engine attribute ? name: patio/aibrix
- [ ] delete order
- [ ] support lws -- replace  podtemplate with leaderworkertemplate

### Refactor code
- [ ] semanticallyEqual uses cmp function

### UpdateStrategy & Failover

- [ ] support upgradeStrategy by rbgs.
- [ ] support restartPolicy.
> Policy: 
> - restart self. by default
> - restart dependencies. When a role is rebuilt, other roles that depend on it also need to be rebuilt. For example, if there are two
    roles, leader and worker, and the worker depends on the leader, when the leader is rebuilt, the worker also needs to
    be rebuilt.

### HPA

- [ ] support native HPA
> HPA  -> rbg role autoscaler cr replica -> rbg role replica
> user -> 
>  add rbg role autoscaler cr

### Examples
- [ ] dynamo pd-disagg metrics
- [ ] support deploy `sglang + mooncake`  PD-disagg 


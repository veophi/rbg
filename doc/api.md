## 📚 API Documentation

### Key Fields

| Field           | Type     | Description                                     |
|-----------------|----------|-------------------------------------------------|
| `startupPolicy` | string   | Startup strategy (Ordered/Parallel)             |
| `dependencies`  | []string | Role dependencies list                          |
| `workload`      | Object   | Underlying workload type (default: StatefulSet) |

Full API spec: [API_REFERENCE.md]()
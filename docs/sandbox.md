# Sandbox 沙箱架构

每个项目 = 一个 `AgentSandbox` CR,生命周期绑项目。执行链路由独立
Agent-Sandbox-Controller 兜底,VisualKB 只 apply CR + exec。桶跨项目
Archive 保留;idle 由 controller 看 `status.lastActivity` 判定。

---

## 1. 全景

```
  VisualKB (sandbox_service)
        │ ApplySandbox / WaitRunning / Suspend / TouchActivity
        ▼
  Agent-Sandbox-Controller (controller-runtime)
        │ reconcile AgentSandbox CR
        ▼
  namespace `sandbox-u-<uid>`          ← controller ensure,持久保留
  ├─ ResourceQuota(pods:20) LimitRange NetworkPolicy
  ├─ MinIO bucket `visualkb-u-{uid}-proj-{projectID}` + Secret + PV (csi-s3)
  └─ Pod (gvisor, sleep infinity, 挂 PVC + Secret)
        │
        ▼
  status.publicURL → 静态产物 iframe
```

Controller 在 reconcile 里 ensure ns + quota + policy + bucket,VisualKB
不需要提前建。

---

## 2. CRD & 状态机

```yaml
apiVersion: sandbox.sandbox.io/v1
kind: AgentSandbox
metadata:
  name: sandbox-{projectID}
  namespace: sandbox-u-{uid}
  labels:
    agentsandbox.sandbox.io/tenant: u-{uid}
    agentsandbox.sandbox.io/project: {projectID}
spec:
  tenantID: u-{uid}
  projectID: {projectID}
  runtime:
    image: alpine:3.20
    cmd: [sleep, infinity]
    env: []
  pvc:
    storageClassName: s3-csi-minio
    size: 10Gi
    volumeAttributes: {}
  resources: {}                 # requests/limits for runtime container
  ttl: 0s                       # 硬上限,0 = 无限
  idleTTL: 30m                  # 闲置后自动 Suspend
status:
  phase: Pending | Running | Suspended | Expired | Failed
  podName: sandbox-{projectID}-runtime
  bucket: visualkb-{tenant}-proj-{project}
  publicURL: https://...
  previewHost: {tenant}-{project}.preview.example.com
  lastActivity: ...
  suspendedAt: ...              # 进入 Suspended 时填;Running 为 nil
  conditions: []
```

**没有 `spec.suspend` 字段** — controller 看 `lastActivity` 与
`spec.idleTTL` 自动 reap,**VisualKB 不主动挂起**。exec 命中项目沙箱
时 controller 自动重写 `lastActivity`,续命闭环无需 VisualKB 介入。

---

## 3. 项目边界生命周期

| 时机 | VisualKB 调 | Controller 反应 |
|---|---|---|
| Project Create | `EnsureRunning` (eager) | 建 ns + quota + policy + pod,等 Ready |
| Bash 首次 | `EnsureRunning` (幂等) | 已 Ready 直返;否则重建 pod |
| Bash 调用 | `Exec` (podName, ns, cmd) | 自动 patch `lastActivity`,SPDY exec |
| idleTTL 到期 | 无 | 自动 Suspend:删 pod,CR 写 phase=Suspended |
| Project Archive | `DeleteSandbox` | 删 CR + pod + PVC;保留 ns + bucket |

Conversation Create/Archive 不动沙箱。

**不变量**:桶跨 Archive 永久保留;LRU 由 controller 看
`status.lastActivity` ASC 接管,软 cap 8/用户 + K8s `pods:20` 硬上限;
namespace 跟 bucket 一样持久化。

---

## 4. 资源隔离五层

1. **Namespace** — `sandbox-u-<uid>`,controller ensure,持久保留。
2. **ResourceQuota** — `pods: 20` 硬上限(ns 内);超限 K8s API 拒 CreatePod。
3. **LimitRange** — controller ensure,默认 + 上限防离谱声明。
4. **NetworkPolicy** — `default-deny-ingress` + `allow-probe`(kube-system:8081)。
5. **Runtime** — cluster RuntimeClass `gvisor`(Sentry 拦截 syscall),不依赖 KVM。

软:controller LRU 挤掉超额的。硬:quota 用尽 K8s 拒。

---

## 5. 启动步骤

```bash
make up                                # postgres / redis / minio / otel-lgtm
make minikube-up                       # 本地 k8s

# 独立 repo
cd ../Agent-Sandbox-Controller
make deploy                            # CRD + controller + RBAC

make backend                           # server
make frontend                          # next dev
```

第一个 CR apply 时 controller 自建 ns + quota + policy,无手动 apply 步骤。

---

## 6. 调试

```bash
# CR 状态
kubectl get agentsandbox -A
kubectl describe agentsandbox -n sandbox-u-<uid> sandbox-<projectID>

# 用户 ns 资源(controller 创建)
kubectl describe namespace sandbox-u-<uid>
kubectl get resourcequota,limitrange,networkpolicy -n sandbox-u-<uid>

# controller 日志
kubectl logs -n sandbox-system -l app=agent-sandbox-controller -f

# 按项目列 CR(看 lastActivity)
kubectl get agentsandbox -n sandbox-u-<uid> -L agentsandbox.sandbox.io/project

# 强清某用户全部 CR(下个 Project Create 自动重建,bucket + ns 仍在)
kubectl delete agentsandbox -n sandbox-u-<uid> --all

# 强清整 ns(慎用,带走所有项目 PVC)
kubectl delete ns sandbox-u-<uid>
```

---

## 7. 现状

**生命周期**:项目独占一个 CR;多会话共用同一沙箱。
活跃 = `idleTTL` 内 exec,controller 看 `status.lastActivity` 自动 Suspend。

**Eager Create**:项目创建即 best-effort 起 pod,首聊零延迟;
`idleTTL` 兜底 reap 僵尸项目。

**Archive = 删 CR**:Archive 同步删 CR + pod + PVC;桶与 ns 由 controller
finalizer 永久保留。

**exec 身份走 ctx**:`ExecCommand(ctx, convoID, messageID, cmd, timeout)`;
`userID` / `projectID` 从 `UserIDFromContext` / `ProjectIDFromContext` 取。

**LRU**:controller 按 `status.lastActivity` ASC,软 cap 8/用户 +
K8s ResourceQuota `pods:20` 硬上限。

**build**:agent 在 sandbox bash 里组合(`npm run build` / `hugo`)。

**桶保留**:`reconcileDelete` 的 finalizer 只清 Pod 与 PVC,
MinIO bucket 保留。

---

## 8. Preview URL 真值来源

`status.publicURL` 一旦 stamp 永不变;CR 跨 Suspend / Resume / idleTTL
全程留存。**写回 DB 是必须的**:

- 真相来源是 S3 bucket,不是 K8s CR(CR 仅描述,不持有数据)。
- Project Archive 不删 CR 之外的真相(bucket / ns / CR 都在);但若
  controller 重启丢 in-memory cache,首次 reconcile 才会重写 URL——等待
  不该出现在前端轮询路径上。
- 前端轮询 `/active-project` 间隔 2s;每轮打 K8s API 不合理,
  DB 单列查询 O(1) 满足。

写入时机:`ProjectService` 的 `CreateFromChat / CreateForUser /
GetActive / SetActive` 收尾处调 `stampPreviewURL(ctx, userID, p)`:
DB 当前为空且 controller 返回非空 URL 时写一次,后续读 DB。读侧
直接消费 `Project.PreviewURL`,不再二次问 controller。

## 9. Static router 路由机制

Controller **不写 Ingress**,而是通过一个独立 `Deployment
"static-router"`(image `controller:v5-tunnel-route`,cmd
`/static-router`)提供静态产物路由。

```
[iframe]  GET https://u-<uid>-<uuid>.preview.example.com/
  → operator edge (Cloudflare Worker / nginx-ingress / ELB)
  → static-router pod :8080, ns=agent-sandbox-system
    匹配 Host header → projects-routes ConfigMap
    value = "visualkb-u-<uid>-proj-<uuid>|/workspace/dist"
    → 拆 bucket/prefix,从 MinIO 拿对象
  → 200 OK,body from /workspace/dist
```

代码位置:
- `agent-sandbox/internal/controller/agentsandbox_controller.go:240-246`
  `Routes.UpsertRoute(bucket, "/workspace/dist", previewHost)`
- `agent-sandbox/internal/controller/route_registry.go:82-110`
  merge-patch 写 ConfigMap `projects-routes` (ns `agent-sandbox-system`)
- `agent-sandbox/cmd/static-router/handler.go`
  按 `r.Host` 在 routeTable lookup → MinIO fetch
- `agent-sandbox/config/static-router/deployment.yaml`
  watch CM,30s resync,listen `:8080`
- `agent-sandbox/config/static-router/service-static-router.yaml`
  ClusterIP `:8080` + NodePort `30800`(host 入口)

VisualKB 这边 iframe 直接 `preview_url` 调,backend **不代理**,
路由完全在 controller + static-router 这一侧闭环。



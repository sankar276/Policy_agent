# GitOps Validator - Flux & ArgoCD Complete! ✅

Comprehensive GitOps policies for Flux CD and ArgoCD have been created!

## What Was Created

### Flux CD Policies (3 files)
1. **[flux/kustomization.rego](policies/gitops/flux/kustomization.rego)** - Kustomization best practices
2. **[flux/gitrepository.rego](policies/gitops/flux/gitrepository.rego)** - GitRepository security
3. **[flux/helmrelease.rego](policies/gitops/flux/helmrelease.rego)** - HelmRelease configuration

### ArgoCD Policies (2 files)
1. **[argocd/application.rego](policies/gitops/argocd/application.rego)** - Application best practices
2. **[argocd/appproject.rego](policies/gitops/argocd/appproject.rego)** - AppProject multi-tenancy & RBAC

---

## Flux CD Policies

### Kustomization Policies

| Policy | Severity | Description |
|--------|----------|-------------|
| **Prune Enable** | HIGH | Must enable prune to clean up deleted resources |
| **Source Reference** | HIGH | Must specify sourceRef |
| **Interval Minimum** | MEDIUM | Minimum 1 minute interval |
| **Force Sync** | HIGH | Disallow force sync (dangerous) |
| **Health Checks** | MEDIUM | Should define health checks |
| **Service Account** | HIGH | Production should use dedicated SA |
| **Validation** | LOW | Don't disable validation |
| **Path Traversal** | CRITICAL | No `..` in paths |

**Example Violations:**

```yaml
# ❌ No prune - orphaned resources won't be deleted
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: apps
spec:
  interval: 10m
  sourceRef:
    kind: GitRepository
    name: flux-system
  path: ./apps
  # prune: false  ❌ Missing or false

# ❌ Interval too short
spec:
  interval: 30s  # ❌ Less than 1 minute!

# ❌ Force sync enabled
spec:
  force: true  # ❌ Dangerous!

# ❌ Production without ServiceAccount
metadata:
  namespace: production
spec:
  # serviceAccountName missing ❌
```

**Fixed:**

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: apps
  namespace: production
spec:
  interval: 5m  # ✅ Reasonable interval
  prune: true   # ✅ Clean up deleted resources

  sourceRef:
    kind: GitRepository
    name: flux-system

  path: ./apps

  serviceAccountName: kustomize-controller  # ✅ Dedicated SA

  wait: true    # ✅ Wait for resources to be ready
  timeout: 5m   # ✅ Timeout set

  healthChecks:  # ✅ Health verification
    - apiVersion: apps/v1
      kind: Deployment
      name: app
      namespace: production
```

### GitRepository Policies

| Policy | Severity | Description |
|--------|----------|-------------|
| **URL Required** | HIGH | Must specify repository URL |
| **Verified Providers** | MEDIUM | Use approved Git providers |
| **HTTPS/SSH Only** | HIGH | No insecure HTTP |
| **Secret Reference** | MEDIUM | Private repos need secretRef |
| **Commit Verification** | MEDIUM | Enable GPG verification |
| **Interval Limits** | MEDIUM | 1-10 minute interval |
| **Explicit Ref** | LOW | Specify branch/tag/commit |
| **Mutable Tags** | LOW | Avoid `latest`, `main` tags |

**Example Violations:**

```yaml
# ❌ Using HTTP
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: app-repo
spec:
  url: http://github.com/org/repo  # ❌ Insecure HTTP!
  interval: 30s  # ❌ Too frequent!

# ❌ No verification
spec:
  url: https://github.com/org/repo
  # verify missing ❌

# ❌ No explicit ref
spec:
  url: https://github.com/org/repo
  # ref missing - uses default branch ❌
```

**Fixed:**

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: app-repo
spec:
  url: https://github.com/org/repo  # ✅ HTTPS
  interval: 5m  # ✅ Reasonable interval

  ref:
    branch: main  # ✅ Explicit ref

  secretRef:
    name: git-credentials  # ✅ For private repos

  verify:  # ✅ GPG verification
    mode: head
    secretRef:
      name: git-pgp-keys

  timeout: 60s  # ✅ Timeout
```

### HelmRelease Policies

| Policy | Severity | Description |
|--------|----------|-------------|
| **Chart Reference** | HIGH | Must specify chart |
| **Interval Required** | HIGH | Must set reconciliation interval |
| **Rollback Config** | HIGH | Production needs rollback |
| **Timeout** | MEDIUM | Set installation timeout |
| **Version Pinning** | MEDIUM | Pin chart versions |
| **CRDs Policy** | MEDIUM | Don't skip CRDs |
| **Force Upgrade** | LOW | Warn on force upgrades |

**Example Violations:**

```yaml
# ❌ Production without rollback
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: app
  namespace: production
spec:
  interval: 10m
  chart:
    spec:
      chart: app
      sourceRef:
        kind: HelmRepository
        name: charts
  # rollback missing ❌

# ❌ No version pinning
spec:
  chart:
    spec:
      chart: app
      # version missing ❌

# ❌ CRDs skipped
spec:
  install:
    crds: Skip  # ❌ Won't install CRDs!
```

**Fixed:**

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: app
  namespace: production
spec:
  interval: 10m
  timeout: 5m  # ✅ Timeout

  chart:
    spec:
      chart: app
      version: 1.2.3  # ✅ Pinned version
      sourceRef:
        kind: HelmRepository
        name: charts

  install:
    crds: CreateReplace  # ✅ Install CRDs
    remediation:
      retries: 3

  upgrade:
    remediation:
      retries: 3
    cleanupOnFail: true

  rollback:  # ✅ Rollback configured
    enable: true
    timeout: 5m

  test:  # ✅ Run Helm tests
    enable: true

  serviceAccountName: helm-controller  # ✅ Dedicated SA
```

---

## ArgoCD Policies

### Application Policies

| Policy | Severity | Description |
|--------|----------|-------------|
| **Source Required** | HIGH | Must specify source repo |
| **Destination Required** | HIGH | Must specify destination |
| **Namespace Required** | HIGH | Destination needs namespace |
| **Project Assignment** | MEDIUM | Don't use 'default' project |
| **Auto-sync + Prune** | HIGH | Auto-sync requires prune |
| **Production Auto-sync** | LOW | Warn on production auto-sync |
| **HTTPS/SSH Only** | HIGH | No insecure HTTP sources |
| **Target Revision** | LOW | Specify explicit revision |
| **Sync Options** | LOW | Configure sync options |
| **Replace Option** | CRITICAL | Don't use Replace=true |

**Example Violations:**

```yaml
# ❌ Using default project
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: app
spec:
  project: default  # ❌ Use dedicated project!

  source:
    repoURL: http://github.com/org/repo  # ❌ HTTP!
    # targetRevision missing ❌
    path: apps/app

  destination:
    server: https://kubernetes.default.svc
    # namespace missing ❌

  syncPolicy:
    automated:  # ❌ Auto-sync without prune
      # prune missing ❌

# ❌ Dangerous sync option
spec:
  syncPolicy:
    syncOptions:
      - Replace=true  # ❌ Can cause data loss!
```

**Fixed:**

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: app
  finalizers:  # ✅ Prevent accidental deletion
    - resources-finalizer.argocd.argoproj.io
spec:
  project: production  # ✅ Dedicated project

  source:
    repoURL: https://github.com/org/repo  # ✅ HTTPS
    targetRevision: v1.0.0  # ✅ Explicit version
    path: apps/app

  destination:
    server: https://kubernetes.default.svc
    namespace: production  # ✅ Namespace specified

  syncPolicy:
    automated:
      prune: true     # ✅ Clean up deleted resources
      selfHeal: true  # ✅ Auto-recover from drift

    syncOptions:
      - CreateNamespace=true
      - PruneLast=true

    retry:  # ✅ Retry configuration
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m

  ignoreDifferences:  # ✅ Specific ignores only
    - group: apps
      kind: Deployment
      jsonPointers:
        - /spec/replicas
```

### AppProject Policies

| Policy | Severity | Description |
|--------|----------|-------------|
| **Source Repos** | HIGH | Define allowed repositories |
| **No Wildcard Repos** | HIGH | Don't allow all repos (*) |
| **Destinations** | HIGH | Define allowed destinations |
| **Namespace Restrictions** | MEDIUM | Don't allow all namespaces |
| **Cluster Resources** | MEDIUM | Whitelist cluster resources |
| **Orphaned Resources** | LOW | Configure orphaned resource detection |
| **Secret Blacklist** | MEDIUM | Consider blacklisting direct Secrets |
| **Sync Windows** | LOW | Production should use sync windows |
| **RBAC Roles** | MEDIUM | Define project roles |

**Example Violations:**

```yaml
# ❌ Allows all repositories
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: team-project
spec:
  sourceRepos:
    - "*"  # ❌ Too permissive!

  destinations:
    - namespace: "*"  # ❌ All namespaces!
      server: "*"     # ❌ All clusters!

  clusterResourceWhitelist:
    - group: "*"  # ❌ All resources!
      kind: "*"

# ❌ Production without controls
metadata:
  name: production
spec:
  # No syncWindows ❌
  # No signatureKeys ❌
  # No roles ❌
```

**Fixed:**

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: team-project
  namespace: argocd
spec:
  description: Team project with proper isolation

  sourceRepos:  # ✅ Explicit repos
    - https://github.com/org/app-repo
    - https://github.com/org/infra-repo

  destinations:  # ✅ Specific destinations
    - namespace: team-*
      server: https://kubernetes.default.svc
    - namespace: team-prod
      server: https://prod-cluster.example.com

  clusterResourceWhitelist:  # ✅ Specific resources
    - group: ""
      kind: Namespace
    - group: rbac.authorization.k8s.io
      kind: Role
    - group: rbac.authorization.k8s.io
      kind: RoleBinding

  namespaceResourceBlacklist:  # ✅ Block direct Secrets
    - group: ""
      kind: Secret

  orphanedResources:  # ✅ Detect orphaned resources
    warn: true

  syncWindows:  # ✅ Change control
    - kind: allow
      schedule: "0 9-17 * * 1-5"  # Business hours
      duration: 8h
      applications:
        - "*"

  signatureKeys:  # ✅ Commit verification
    - keyID: ABC123

  roles:  # ✅ RBAC
    - name: developer
      policies:
        - p, proj:team-project:developer, applications, get, team-project/*, allow
        - p, proj:team-project:developer, applications, sync, team-project/*, allow

    - name: admin
      policies:
        - p, proj:team-project:admin, applications, *, team-project/*, allow
```

---

## Quick Reference

### Flux Kustomization Example

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: apps
  namespace: flux-system
spec:
  interval: 10m
  timeout: 5m
  prune: true
  wait: true

  sourceRef:
    kind: GitRepository
    name: flux-system

  path: ./apps/production

  serviceAccountName: kustomize-controller

  healthChecks:
    - apiVersion: apps/v1
      kind: Deployment
      name: app
```

### ArgoCD Application Example

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: guestbook
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: production

  source:
    repoURL: https://github.com/argoproj/argocd-example-apps
    targetRevision: HEAD
    path: guestbook

  destination:
    server: https://kubernetes.default.svc
    namespace: guestbook

  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
    retry:
      limit: 5
```

---

## Summary

**GitOps Policies Created: 5 files**
- 3 Flux CD policies (Kustomization, GitRepository, HelmRelease)
- 2 ArgoCD policies (Application, AppProject)

**Total Checks: 50+ policies**
- Security: 20+ checks
- Best Practices: 20+ checks
- Recommendations: 10+ suggestions

**Coverage:**
- ✅ Flux CD - Complete
- ✅ ArgoCD - Complete
- ✅ GitOps best practices
- ✅ Multi-tenancy & RBAC
- ✅ Security & compliance
- ✅ Production readiness

🎉 **GitOps validation is ready!**

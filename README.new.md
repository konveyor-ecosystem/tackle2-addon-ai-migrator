# tackle2-addon-ai-migrator

A [Konveyor](https://www.konveyor.io/) addon that uses AI to assist with application migration. Based on [tackle2-addon-discovery](https://github.com/konveyor/tackle2-addon-discovery/tree/7f394da).

## Prerequisites

- Access to an OpenShift cluster with Konveyor/Tackle2 installed
- `oc` CLI authenticated to the cluster
- Go 1.21+ (for local development)

## Project Structure

```
cmd/                          # Addon entrypoint (main.go, discovery.go)
ai-migrator-addon-cr.yaml    # Addon Custom Resource definition
Dockerfile                    # Two-stage build (Go toolset -> UBI9 minimal)
.vscode/launch.json           # VS Code debug configuration
.env                          # Local env vars (gitignored)
```

## Setup

### 1. Port-forward the hub

The hub API isn't directly accessible through the OpenShift route (Keycloak proxy intercepts everything). Port-forward to bypass it:

```bash
oc port-forward svc/tackle-hub 8080:8080 -n konveyor-tackle
```

### 2. Disable auth for local development

The hub's task validator checks that addon pods are running in Kubernetes, which always fails when debugging locally. Disable auth via the Tackle CR (not the deployment directly -- the operator would revert it):

```bash
oc patch tackle tackle -n konveyor-tackle --type merge -p '{"spec":{"feature_auth_required":"false"}}'
oc rollout status deployment/tackle-hub -n konveyor-tackle --timeout=120s
oc set env deployment/tackle-hub --list -n konveyor-tackle | grep AUTH_REQUIRED
```

Restart the port-forward after the hub pod restarts.

To re-enable auth later:

```bash
oc patch tackle tackle -n konveyor-tackle --type merge -p '{"spec":{"feature_auth_required":"true"}}'
```

### 3. Apply the Addon CR

```bash
oc apply -f ai-migrator-addon-cr.yaml -n konveyor-tackle
```

Verify:

```bash
oc get addon ai-migrator -n konveyor-tackle -o yaml
```

### 4. Create a task instance

Task instances are created via the hub REST API (they live in the hub's database, not as Kubernetes CRs).

Find an application ID:

```bash
curl -kSs "http://localhost:8080/applications?limit=10" \
  -H "Accept: application/json" | python3 -m json.tool
```

Create a task with blank state (defaults to `Created` so the task manager won't launch a pod):

```bash
curl -kSs -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{
    "name": "ai-migrator-test",
    "addon": "ai-migrator",
    "application": {"id": <APP_ID>},
    "data": {"source":"ai-migrator"}
}'
```

Note the `id` in the response -- you'll need it for the `.env` file.

### 5. Configure `.env`

Create a `.env` file (gitignored) with:

```
TASK=<task ID from step 4>
SHARED_PATH=/path/to/this/repo/shared
HUB_BASE_URL=http://localhost:8080
```

### 6. Debug locally

Use VS Code with the included launch configuration (`Debug Addon`), or run directly:

```bash
export TASK=<task ID>
export SHARED_PATH=$(pwd)/shared
export HUB_BASE_URL=http://localhost:8080
go run ./cmd
```

## Key Concepts

- **Addon CR**: Kubernetes resource that tells the hub what container to run. Applied with `oc apply`.
- **Task instance**: Created via hub REST API (`POST /tasks`), stored in the hub database. State `Created` = task manager ignores it; `Ready` = task manager runs it.
- **`source` field**: Tags created by an addon are stamped with a source string (e.g., `"ai-migrator"`). Use a unique source so `Replace()` won't clobber tags from other addons.
- **Hub SDK**: `github.com/konveyor/tackle2-hub/shared/addon/` -- provides `addon.Run()`, `addon.DataWith()`, `addon.Application.Tags()`, `addon.Application.Facts()`, etc.

## Building

```bash
make cmd
```

Or with Docker:

```bash
podman build -t quay.io/konveyor/tackle2-addon-ai-migrator:latest .
```

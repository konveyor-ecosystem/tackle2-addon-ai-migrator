# tackle2-addon-ai-migrator

A Tackle2 addon that uses [Goose](https://github.com/block/goose) to perform AI-assisted code migrations. The addon clones an application's source repo, fetches migration rules, runs a Goose recipe, and uploads the report back to the hub.

## Prerequisites

- Auth disabled on the hub (for now)
- `oc port-forward` to the hub service
- The Addon CR applied to the cluster

## Setup

### 1. Port-forward the hub

The hub API isn't directly accessible through the OpenShift route. Port-forward to bypass it:

```bash
oc port-forward svc/tackle-hub 8080:8080 -n konveyor-tackle
```

### 2. Apply the Addon CR

```bash
oc apply -f ai-migrator-addon-cr.yaml -n konveyor-tackle
```

Verify:

```bash
oc get addon ai-migrator -n konveyor-tackle -o yaml
```

## Usage

### 1. Find the application ID

```bash
curl -sS "http://localhost:8080/applications?limit=50" \
  -H "Accept: application/json" \
  | python3 -c "
import sys, json
for app in json.load(sys.stdin):
    print(f'  id={app[\"id\"]}  name={app[\"name\"]}')
"
```

### 2. Create a task

Task instances are created via the hub REST API (not Kubernetes CRs). Leave `state` blank (defaults to `Created`) so the task manager won't try to launch the container -- for local debugging we run the addon directly.

```bash
curl -sS -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{
    "name": "ai-migrator-test",
    "addon": "ai-migrator",
    "application": {"id": 231},
    "data": {
      "profile": {"id": 1},
      "sourceTech": "PatternFly 5",
      "targetTech": "PatternFly 6"
    }
  }'
```

Save the `id` from the response (e.g. `1145`).

#### Task data fields

| Field | Description |
|-------|-------------|
| `profile` | Analysis profile reference (by ID). Populates rules from the profile's repository/files. |
| `sourceTech` | Source technology (e.g. `"PatternFly 5"`) |
| `targetTech` | Target technology (e.g. `"PatternFly 6"`) |
| `rules` | Optional. Rules directly (repository, rulesets, files). If both `profile` and `rules` are set, the profile populates first, then direct rules override. |

#### Applications on this cluster

| ID  | Name | Repo | PF Version |
|-----|------|------|------------|
| 229 | Migration-Toolkit-for-Containers | migtools/mig-ui | PF 4 |
| 230 | Migration-Toolkit-for-Virtualization | kubev2v/forklift-console-plugin | PF 6 |
| 231 | Migration-Toolkit-For-Applications | konveyor/tackle2-ui (path: client) | PF 5 |
| 232 | Migration-Tools-Shared-Library | migtools/lib-ui | PF 5 |

### 3. Run the addon locally

Update `.env` with the task ID from step 2:

```
TASK=1145
SHARED_PATH=/home/jonah/Projects/github.com/konveyor-ecosystem/tackle2-addon-ai-migrator/shared
HUB_BASE_URL=http://localhost:8080
GOOSE_BIN=/home/jonah/Projects/github.com/konveyor-ecosystem/tackle2-addon-ai-migrator/bin/fake-goose
RECIPE_PATH=/home/jonah/Projects/github.com/konveyor-ecosystem/tackle2-addon-ai-migrator/recipes/goose/recipes/migration.yaml
```

Then either hit F5 in VS Code (uses `.vscode/launch.json`) or:

```bash
source .env && go run ./cmd
```

> **Tip**: Set `GOOSE_BIN` to `bin/fake-goose` for fast iteration. Build it with `make fake-goose`.

### 4. Inspect results

**Task status:**

```bash
curl -sS http://localhost:8080/tasks/1145 \
  -H "Accept: application/json" | python3 -m json.tool
```

**Task attachments (report file):**

```bash
curl -sS http://localhost:8080/tasks/1145 \
  -H "Accept: application/json" \
  | python3 -c "
import sys, json
task = json.load(sys.stdin)
for a in (task.get('attached') or []):
    print(f'  id={a[\"id\"]}  name={a.get(\"name\",\"\")}')
"
```

**Download the report:**

```bash
curl -sS http://localhost:8080/files/<fileId> -o report.html
```

**Query facts:**

```bash
curl -sS http://localhost:8080/applications/231/facts/ai-migrator: \
  -H "Accept: application/json" \
  | python3 -c "
import sys, json
facts = json.load(sys.stdin)
for k, v in facts.items():
    if isinstance(v, list):
        print(f'{k}: {bytes(v).decode()}')
    else:
        print(f'{k}: {v}')
"
```

> [!NOTE]: Fact values come back as byte arrays due to a hub serialization issue. The python snippet above decodes them. The primary report access path is the task attachment (`addon.Attach`), not facts.

**Goose output log:**

The hub SDK's `command.New()` automatically captures goose stdout/stderr and uploads it as a task attachment. Find the goose output file in the attachments list (it will be named `goose.output` or `fake-goose.output`), then download it by file ID:

```bash
curl -sS http://localhost:8080/files/<fileId>
```

## Building

```bash
make cmd             # build the addon binary
make fake-goose      # build the fake goose binary for testing
make image-podman    # build the container image
```

## Addon Flow

```
1. addon.DataWith(&data)         -- parse task JSON into Data struct
2. applyProfile(data)            -- if profile ID set, fetch it and populate Rules
3. FetchRepository(application)  -- clone the app's git repo
4. Rules.Build()                 -- fetch rules from hub (files, rulesets, git repos)
5. Goose.Run()                   -- goose run --recipe ... --params ...
6. uploadReport()                -- addon.File.Post() + addon.Attach() + store facts
```

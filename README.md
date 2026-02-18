# tackle2-addon-ai-migrator

## Auth

To make sure `oc` works, we need to get to the tackle-hub service. The hub API isn't directly accessible through the OpenShift route (Keycloak proxy intercepts everything). Port-forward to bypass it:

```bash
oc port-forward svc/tackle-hub 8080:8080 -n konveyor-tackle
```

Login and get a token:

```bash
curl -kSs -X POST http://localhost:8080/auth/login \
  -H 'Content-Type:application/x-yaml' \
  -H 'Accept:application/json' \
  -d 'user: ${CLUSTER_USERNAME}
password: "${CLUSTER_PASSWORD}"' > token.json
```

Add the token as `TOKEN` in your env, preferably in a `.env` file. Note that the token expires in 5 minutes.

Query the API:

```bash

curl -kSs http://localhost:8080/tasks?limit=5 \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Accept: application/json" > tasks.json
```

## Applying CRs

For now, runs a skill to migrate the given application

Apply the Addon CR:

```bash
oc apply -f ai-migrator-addon-cr.yaml -n konveyor-tackle
```

Verify:

```bash
oc get addon ai-migrator -n konveyor-tackle -o yaml
```

## Creating Tasks

> ![NOTE] We are not creating a Task CR right now. This is wrong. The Task manager only runs tasks with a state of "ready". For debugging, we will simply create one without a state field. Blank gets mapped to "created".
>
> Why do this? Rapid development, credentials get weird when uploading and running image from quay. Hardcoding the LLM credentials locally for now.

To list applications and find IDs:

```bash
curl -kSs "http://localhost:8080/applications?limit=10" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Accept: application/json" > applications.json
```

Task instances are NOT Kubernetes CRs. They are created via the hub REST API and live in the hub's database.

Leave `state` blank (defaults to `Created`) so the task manager won't try to launch the container:

```bash
source .env
curl -kSs -X POST http://localhost:8080/tasks \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{
    "name": "ai-migrator-test",
    "addon": "ai-migrator",
    "application": {"id": 229},
    "data": {"source":"ai-migrator"}
}'
```

The response will include the task `id` (e.g., `1141`). Use this for local debugging.

To verify it was created, look in the UI or do:

```bash
curl -kSs http://localhost:8080/tasks/1141 \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Accept: application/json"
```
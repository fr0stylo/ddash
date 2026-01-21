# GitHub App Setup + Webhook Ingestion (Draft)

## 1) GitHub App setup (step-by-step)
1) GitHub → Settings → Developer settings → GitHub Apps → New GitHub App.
2) App name + Homepage URL.
3) Webhook URL: `https://your-domain.com/github/webhook`
4) Webhook secret: generate a strong secret (store in env).
5) Permissions (minimum):
   - Deployments: Read & write
   - Repository metadata: Read-only
   - (Optional) Actions: Read-only (for workflow context)
6) Subscribe to events:
   - `deployment`
   - `deployment_status`
   - (Optional) `workflow_run`
7) Install the app on a test org/repo.

## 2) Go webhook handler skeleton (Echo)
Below is a minimal handler that verifies the `X-Hub-Signature-256` and routes events.

```go
package githubwebhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
)

func verifyGitHubSignature(secret []byte, body []byte, signature string) bool {
	const prefix = "sha256="
	if len(signature) <= len(prefix) || signature[:len(prefix)] != prefix {
		return false
	}
	sigBytes, err := hex.DecodeString(signature[len(prefix):])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)
	return hmac.Equal(sigBytes, expected)
}

func Handler(secret []byte) echo.HandlerFunc {
	return func(c echo.Context) error {
		event := c.Request().Header.Get("X-GitHub-Event")
		signature := c.Request().Header.Get("X-Hub-Signature-256")

		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.NoContent(http.StatusBadRequest)
		}

		if !verifyGitHubSignature(secret, body, signature) {
			return c.NoContent(http.StatusUnauthorized)
		}

		switch event {
		case "deployment":
			// TODO: parse deployment payload
			// TODO: store deployment record
		case "deployment_status":
			// TODO: parse deployment status payload
			// TODO: update deployment status record
		default:
			// ignore unsupported events
		}

		return c.NoContent(http.StatusOK)
	}
}
```

### Hook into Echo
```go
// in server.go
// e.POST("/github/webhook", githubwebhook.Handler([]byte(os.Getenv("GITHUB_WEBHOOK_SECRET"))))
```

## 3) Minimal payload structs (draft)
These are minimal fields you likely need for the UI.

```go
type RepoRef struct {
	FullName string `json:"full_name"`
}

type DeploymentPayload struct {
	Repository RepoRef `json:"repository"`
	Deployment struct {
		ID          int64  `json:"id"`
		Environment string `json:"environment"`
		Ref         string `json:"ref"`
		Sha         string `json:"sha"`
		CreatedAt   string `json:"created_at"`
	} `json:"deployment"`
}

type DeploymentStatusPayload struct {
	Repository RepoRef `json:"repository"`
	Deployment struct {
		ID int64 `json:"id"`
	} `json:"deployment"`
	DeploymentStatus struct {
		State     string `json:"state"`
		LogURL    string `json:"log_url"`
		UpdatedAt string `json:"updated_at"`
	} `json:"deployment_status"`
}
```

## 4) Minimal DB schema (draft)
- installations(id, org, repos, created_at)
- deployments(repo, environment, deployment_id, sha, ref, status, url, updated_at)

## 5) Optional “custom action” ingestion API
For teams without deployments:
- `POST /api/deployments`
  - body: `{ service, env, status, sha, url, time }`


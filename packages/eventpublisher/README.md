# Event Publisher package

This module provides a stable import path for publishing CDEvents to DDash webhook endpoints.

Import path:

```go
import "github.com/fr0stylo/ddash/packages/eventpublisher"
```

Exposed API:

- `eventpublisher.Client`
- `eventpublisher.Event`
- `eventpublisher.BuildEventBody`

The implementation is backed by `github.com/fr0stylo/ddash/pkg/eventpublisher`.

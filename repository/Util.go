package repository

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// idSeq disambiguates GetIdToRecord calls that land in the same nanosecond
// tick — time.Now().UnixNano() alone isn't guaranteed unique across rapid
// successive calls on some platforms/clocks, and the previous version used
// whole-second (UnixUnix()) resolution, which collided on any second call
// within the same wall-clock second (seen in practice: two OTPs requested in
// a tight loop got the same id and the second INSERT failed on the PK).
var idSeq uint64

func GetIdToRecord(code string) string {
	seq := atomic.AddUint64(&idSeq, 1)
	return fmt.Sprintf("%s%d%d", code, time.Now().UnixNano(), seq)
}

// LoggedInUserID reads the authenticated user id middleware attached via
// c.Locals("user_id"). Callers pass ctx.Context() (a *fasthttp.RequestCtx,
// which implements context.Context.Value by delegating to fiber's per-request
// Locals store) — so this is safe under concurrent requests, unlike the old
// globals.CurrentLoggedInUser package-level variable it replaces (that was
// written by every request's auth middleware with no per-request isolation,
// so one request's audit-trail writes could pick up a different concurrent
// request's user id). Returns "" if no request-scoped user id is set (e.g.
// this ctx came from a background job rather than an HTTP request).
func LoggedInUserID(ctx context.Context) string {
	if v, ok := ctx.Value("user_id").(string); ok {
		return v
	}
	return ""
}

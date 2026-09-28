package globals

import (
	"github.com/gofiber/fiber/v2/middleware/session"
)

// Store is the cookie-session store used by a few legacy favourites endpoints
// (see repository/user_repository.go) to read back the request's own userID
// from its own session cookie — this is safe, request-scoped state, unlike
// the process-wide "current user" singleton that used to live in this file
// (removed: every concurrent request wrote to it, so one request's
// audit-trail writes could pick up a different request's user id). The
// authenticated user id for everything else flows through
// c.Locals("user_id") / repository.LoggedInUserID(ctx.Context()).
var Store = session.New(session.Config{
	Expiration: 0, // Session lasts until the browser is closed or refreshed
})

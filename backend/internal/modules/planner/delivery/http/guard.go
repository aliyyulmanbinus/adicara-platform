package http

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/ginutil"
	authhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/delivery/http"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/usecase"
)

const projectContextKey = "plannerProject"

// ParseID reads a canonical UUID (36 characters). Anything else, including the
// braced and urn forms uuid.Parse also accepts, is "not a project id" and
// ends as 404, never as a query.
func ParseID(s string) (uuid.UUID, bool) {
	if len(s) != 36 {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(s)

	return id, err == nil
}

// ProjectGuard loads the project of :id for the authenticated user. An id that
// is not a UUID (for example "summary" on a route that is not the static one),
// an unknown project and somebody else's project all answer 404, so the
// existence of a project is never revealed. Must run after BearerAuth.
func ProjectGuard(svc *usecase.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := ParseID(c.Param("id"))
		if !ok {
			ginutil.Error(c, apierr.NotFound("project"))
			c.Abort()
			return
		}

		p, err := svc.ProjectForUser(c.Request.Context(), id, authhttp.CurrentUserID(c))
		if err != nil {
			ginutil.Error(c, err)
			c.Abort()
			return
		}

		c.Set(projectContextKey, p)
		c.Next()
	}
}

// CurrentProject returns the project ProjectGuard stored.
func CurrentProject(c *gin.Context) domain.Project {
	v, _ := c.Get(projectContextKey)
	p, _ := v.(domain.Project)

	return p
}

// An archived project is read-only except for reopening it and deleting it.
// Matched on the route pattern, not on a URL suffix.
var writableWhenArchived = map[string]bool{
	"PATCH /v1/planner/projects/:id/status": true,
	"DELETE /v1/planner/projects/:id":       true,
}

// WriteGuard rejects writes to an archived project with 409 project_read_only.
// Must run after ProjectGuard.
func WriteGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case "GET", "HEAD", "OPTIONS":
			c.Next()
			return
		}

		if CurrentProject(c).Status == domain.StatusArchived && !writableWhenArchived[c.Request.Method+" "+c.FullPath()] {
			ginutil.Error(c, apierr.Conflict("project_read_only", "the project is archived; restore it to make changes"))
			c.Abort()
			return
		}

		c.Next()
	}
}

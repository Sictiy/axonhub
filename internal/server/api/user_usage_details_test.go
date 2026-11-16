// 职责: 验证用户用量明细接口的输入校验。
// 关联文件:
//
//	internal/server/api/user_usage_details.go
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUserUsageDetailsInvalidInputs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, query := range []string{
		"userId=invalid",
		"userId=gid%3A%2F%2Faxonhub%2FUser%2F1&from=bad",
		"userId=gid%3A%2F%2Faxonhub%2FUser%2F1&from=2026-11-16T00%3A00%3A00Z&to=2026-11-15T00%3A00%3A00Z",
	} {
		t.Run(query, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/?"+query, nil)
			(&SystemHandlers{}).UserUsageDetails(c)
			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

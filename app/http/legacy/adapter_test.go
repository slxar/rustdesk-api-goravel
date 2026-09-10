package legacy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	goravelgin "github.com/goravel/gin"
)

func TestHandlerDispatchesToLegacyGinRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/legacy", func(c *gin.Context) { c.String(http.StatusTeapot, "legacy") })

	writer := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(writer)
	context.Request = httptest.NewRequest(http.MethodGet, "/legacy", nil)
	Handler(router)(goravelgin.NewContext(context))

	if writer.Code != http.StatusTeapot || writer.Body.String() != "legacy" {
		t.Fatalf("legacy response = %d %q", writer.Code, writer.Body.String())
	}
}

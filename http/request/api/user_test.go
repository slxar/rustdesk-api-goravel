package api

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGroupQueriesAcceptRustDeskCurrentParameter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("GET", "/api/users?current=2&pageSize=100", nil)

	query := &UserListQuery{}
	if err := ctx.ShouldBindQuery(query); err != nil {
		t.Fatal(err)
	}
	if query.Current != 2 || query.PageSize != 100 {
		t.Fatalf("query = %#v", query)
	}
	if query.PageNumber() != 2 {
		t.Fatalf("current was not normalized: %d", query.PageNumber())
	}
	query.Page = 3
	if query.PageNumber() != 3 {
		t.Fatalf("explicit page did not take precedence: %d", query.PageNumber())
	}
}

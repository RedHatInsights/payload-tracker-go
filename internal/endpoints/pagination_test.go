package endpoints_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"gorm.io/gorm"

	"github.com/redhatinsights/payload-tracker-go/internal/endpoints"
	"github.com/redhatinsights/payload-tracker-go/internal/models"
	"github.com/redhatinsights/payload-tracker-go/internal/structs"
	"github.com/redhatinsights/payload-tracker-go/internal/utils/test"
)

var _ = Describe("Pagination parameter validation", func() {
	var (
		gotPage     int
		gotPageSize int
		called      bool
	)

	BeforeEach(func() {
		gotPage, gotPageSize, called = -99, -99, false
		endpoints.RetrievePayloads = func(_ *gorm.DB, page int, pageSize int, _ structs.Query) (int64, []models.Payloads) {
			called = true
			gotPage, gotPageSize = page, pageSize
			return 0, nil
		}
		endpoints.RetrieveStatuses = func(_ *gorm.DB, q structs.Query) (int64, []structs.StatusRetrieve) {
			called = true
			gotPage, gotPageSize = q.Page, q.PageSize
			return 0, nil
		}
	})

	serve := func(path string, h http.HandlerFunc, query map[string]interface{}) *httptest.ResponseRecorder {
		req, err := test.MakeTestRequest(path, query)
		Expect(err).To(BeNil())
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		return rr
	}

	for _, ep := range []struct {
		path    string
		handler http.HandlerFunc
	}{
		{"/api/v1/payloads", endpoints.Payloads},
		{"/api/v1/statuses", endpoints.Statuses},
	} {
		ep := ep

		Describe(ep.path, func() {
			It("uses the default page size when page_size is omitted", func() {
				rr := serve(ep.path, ep.handler, map[string]interface{}{})
				Expect(rr.Code).To(Equal(http.StatusOK))
				Expect(called).To(BeTrue())
				Expect(gotPage).To(Equal(0))
				Expect(gotPageSize).To(Equal(structs.DefaultPageSize))
			})

			It("accepts page_size at the maximum", func() {
				rr := serve(ep.path, ep.handler, map[string]interface{}{"page_size": structs.MaxPageSize, "page": 2})
				Expect(rr.Code).To(Equal(http.StatusOK))
				Expect(gotPage).To(Equal(2))
				Expect(gotPageSize).To(Equal(structs.MaxPageSize))
			})

			DescribeTable("rejects invalid pagination parameters with 400",
				func(query map[string]interface{}) {
					rr := serve(ep.path, ep.handler, query)
					Expect(rr.Code).To(Equal(http.StatusBadRequest))
					Expect(called).To(BeFalse())
				},
				Entry("negative page_size", map[string]interface{}{"page_size": -1}),
				Entry("zero page_size", map[string]interface{}{"page_size": 0}),
				Entry("page_size above maximum", map[string]interface{}{"page_size": structs.MaxPageSize + 1}),
				Entry("non-integer page_size", map[string]interface{}{"page_size": "abc"}),
				Entry("negative page", map[string]interface{}{"page": -1}),
				Entry("non-integer page", map[string]interface{}{"page": "abc"}),
				Entry("invalid page with valid page_size", map[string]interface{}{"page": "abc", "page_size": 5}),
			)
		})
	}
})

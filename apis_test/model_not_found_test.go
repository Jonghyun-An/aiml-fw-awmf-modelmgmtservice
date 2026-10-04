// SPDX-License-Identifier: Apache-2.0
package apis_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gerrit.o-ran-sc.org/r/aiml-fw/awmf/modelmgmtservice/apis"
	"gerrit.o-ran-sc.org/r/aiml-fw/awmf/modelmgmtservice/apis_test/mme_mocks"
	"gerrit.o-ran-sc.org/r/aiml-fw/awmf/modelmgmtservice/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func TestModelNameVersionNotFound(t *testing.T) {
	lookups := []struct {
		name   string
		method string
		path   string
		handle func(*apis.MmeApiHandler) gin.HandlerFunc
	}{
		{"discovery", http.MethodGet, "/model?model-name=missing&model-version=1", func(h *apis.MmeApiHandler) gin.HandlerFunc { return h.GetModelInfo }},
		{"upload", http.MethodPost, "/model/missing/1", func(h *apis.MmeApiHandler) gin.HandlerFunc { return h.UploadModel }},
		{"artifact", http.MethodPut, "/artifact/missing/1/new", func(h *apis.MmeApiHandler) gin.HandlerFunc { return h.UpdateArtifact }},
	}
	for _, lookup := range lookups {
		for _, test := range []struct {
			name string
			err  error
			want int
		}{
			{"not found", gorm.ErrRecordNotFound, http.StatusNotFound},
			{"wrapped not found", fmt.Errorf("lookup: %w", gorm.ErrRecordNotFound), http.StatusNotFound},
			{"database failure", fmt.Errorf("database unavailable"), http.StatusInternalServerError},
		} {
			t.Run(lookup.name+"/"+test.name, func(t *testing.T) {
				t.Setenv("LOG_FILE_NAME", "testing")
				store := new(mme_mocks.IDBMock)
				store.On("GetModelInfoByNameAndVer").Return((*models.ModelRelatedInformation)(nil), test.err).Once()
				handler := apis.NewMmeApiHandler(nil, store)
				router := gin.New()
				router.GET("/model", lookup.handle(handler))
				router.POST("/model/:modelName/:modelVersion", lookup.handle(handler))
				router.PUT("/artifact/:modelname/:modelversion/:artifactversion", lookup.handle(handler))
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, httptest.NewRequest(lookup.method, lookup.path, nil))
				assert.Equal(t, test.want, recorder.Code, recorder.Body.String())
				store.AssertExpectations(t)
				store.AssertNotCalled(t, "Update", mock.Anything)
			})
		}
	}
}

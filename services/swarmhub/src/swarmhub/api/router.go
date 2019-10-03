package api

import (
	"net/http"
	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/ec2"
	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/jwt"

	"github.com/julienschmidt/httprouter"
)

func SetRouterPaths(router *httprouter.Router) {
	router.GET("/api/test/:id/attachment/:attachmentid", TokenApiAuth(GetTestAttachment))
	router.GET("/api/grafana/info", TokenApiAuth(GrafanaConfigs))
	router.GET("/api/test/:id/attachments", TokenApiAuth(TestAttachments))
	router.POST("/api/grid/:id/start", PowerTokenAPIAuth(StartGrid))
	router.POST("/api/test/:id/edit", PowerTokenAPIAuth(EditTest))
	router.POST("/api/test/:id/label/:label", PowerTokenAPIAuth(LabelToTest))
	router.DELETE("/api/test/:id/label/:label", PowerTokenAPIAuth(LabelToTest))
	router.POST("/api/grid/:id/stop", PowerTokenAPIAuth(StopGrid))
	router.GET("/api/test/:id/deploylogs", TokenApiAuth(deployerLogs))
	router.GET("/api/paginate/test/info", TokenApiAuth(PaginateTestInfo))
	router.GET("/api/paginate/grid/info", TokenApiAuth(PaginateGridInfo))
	router.GET("/api/paginate/test/key/:id", TokenApiAuth(GetTestPaginateKey))
	router.GET("/api/paginate/grid/key/:id", TokenApiAuth(GetGridPaginateKey))
	router.GET("/api/test/:id/ip", TokenApiAuth(ec2.GetMasterIP))
	router.GET("/api/status/test", TokenApiAuth(GetTestStatus))
	router.GET("/api/status/test/refresh", TokenApiAuth(RefreshTestStatus))
	router.GET("/api/test/:id/files", TokenApiAuth(TestFiles))
	router.GET("/api/test/:id/files/download", TokenApiAuth(DownloadScriptFiles))
	router.GET("/api/status/grid", TokenApiAuth(GetGridStatus))
	router.GET("/api/grid/:id/deploylogs", TokenApiAuth(deployerLogs))
	router.GET("/api/grids/providers", TokenApiAuth(GetGridProviderTypes))
	router.GET("/api/grids/regions", TokenApiAuth(GetGridRegionTypes))
	router.GET("/api/grids/instances", TokenApiAuth(GetGridInstanceTypes))
	router.GET("/api/grid/:id", TokenApiAuth(Grid))
	router.POST("/api/grid/:id/delete", PowerTokenAPIAuth(DeleteGrid))
	router.POST("/api/test/:id/delete", PowerTokenAPIAuth(DeleteTest))
	router.POST("/api/test/:id/stop", PowerTokenAPIAuth(StopTest))
	router.POST("/api/test/:id/cancel", PowerTokenAPIAuth(CancelTestDeployment))
	router.GET("/api/grids", TokenApiAuth(Grids))
	router.POST("/api/grid", TokenApiAuth(CreateGrid))
	router.GET("/api/tests", TokenApiAuth(Tests))
	router.GET("/api/tests/:id", TokenApiAuth(TestsPaginate))
	router.POST("/api/test/:id/start", PowerTokenAPIAuth(StartTest))
	router.POST("/api/test/:id/duplicate", PowerTokenAPIAuth(DuplicateTest))
	router.POST("/api/test/:id/attachment", PowerTokenAPIAuth(UploadTestAttachment))
	router.POST("/api/test/:id/attachment/:attachmentid/delete", PowerTokenAPIAuth(DeleteTestAttachment))
	router.GET("/api/grids/list/:id", TokenApiAuth(GridsPaginate))
	router.GET("/api/test", TokenApiAuth(Test))
	router.POST("/api/test", TokenApiAuth(CreateTest))
}

func TokenApiAuth(handler httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		cookie, err := r.Cookie("Authorization")
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		ok, err := jwt.ValidateToken(cookie.Value)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		if ok {
			handler(w, r, ps)
			return
		}

		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized."))
		return

	}
}

func PowerTokenAPIAuth(handler httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		cookie, err := r.Cookie("Authorization")
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		role, err := jwt.TokenRole(cookie.Value)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		if role >= jwt.RolePowerUser {
			handler(w, r, ps)
			return
		}

		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Not a valid poweruser, you can't do that!"))
		return
	}
}

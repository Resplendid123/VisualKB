package document

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, doc *DocumentHandler, tree *TreeHandler) {
	docs := rg.Group("/documents")
	{
		docs.POST("", doc.Create)
		docs.POST("/upload", doc.UploadFile)
		docs.GET("", doc.List)
		docs.GET("/:id", doc.Get)
		docs.GET("/:id/file", doc.ServeFile)
		docs.PATCH("/:id", doc.Patch)
		docs.POST("/:id/versions", doc.AddVersion)
		docs.POST("/:id/archive", doc.Archive)
		docs.POST("/:id/move", doc.Move)
		docs.POST("/:id/ingest", doc.IngestOne)
		docs.POST("/ingest-all", doc.IngestAll)
	}

	treeGroup := rg.Group("/tree")
	{
		treeGroup.GET("", tree.List)
		treeGroup.POST("/folder", tree.CreateFolder)
		treeGroup.POST("/folder/:id/rename", tree.RenameNode)
		treeGroup.POST("/folder/:id/move", tree.MoveNode)
		treeGroup.POST("/folder/:id/delete", tree.DeleteFolder)
	}
}

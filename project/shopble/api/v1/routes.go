package v1

import "github.com/gin-gonic/gin"

// Init — wire toàn bộ v1 routes vào group `/api/v1`.
//
// Không có auth middleware: ở scope này ví đã connect chính là danh tính của buyer,
// không có tài khoản user riêng.
func Init(group *gin.RouterGroup) {
	group.POST("/orders", CreateOrder)
	group.GET("/orders", ListOrders)
	group.GET("/orders/:id", GetOrder)
	group.GET("/orders/:id/evidence", GetOrderEvidence)
	group.GET("/evidence", ListEvidence)
}

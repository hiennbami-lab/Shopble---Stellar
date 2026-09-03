package v1

import (
	"net/http"
	"strconv"

	"shopble/database"
	"shopble/project/shopble/api/apierr"
	"shopble/project/shopble/lib/libstellar"
	"shopble/project/shopble/models"

	"github.com/gin-gonic/gin"
	"github.com/segmentio/ksuid"
	"github.com/shopspring/decimal"
)

// stellarAccountLen — public key Stellar dạng strkey luôn đúng 56 ký tự và bắt đầu bằng G.
const stellarAccountLen = 56

func validAccount(addr string) bool {
	return len(addr) == stellarAccountLen && addr[0] == 'G'
}

// CreateOrder godoc
// @Summary      Tạo order intent và sinh payment instruction testnet
// @Tags         orders
// @Accept       json
// @Produce      json
// @Param        body  body      CreateOrderRequest  true  "Order intent"
// @Success      200   {object}  CreateOrderEnvelope
// @Failure      400   {object}  ErrorResponse
// @Router       /orders [post]
func CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, apierr.CodeDataInvalid, err.Error())
		return
	}
	if !validAccount(req.BuyerWallet) {
		apierr.BadRequest(c, apierr.CodeDataInvalid,
			"buyer_wallet phải là public key Stellar hợp lệ (G..., 56 ký tự)")
		return
	}
	amount, err := decimal.NewFromString(req.ExpectedAmount)
	if err != nil {
		apierr.BadRequest(c, apierr.CodeDataInvalid, "expected_amount không phải số thập phân hợp lệ")
		return
	}

	cfg := libstellar.GetStellarConfig()

	// Memo CHÍNH LÀ order id. ksuid là 27 byte, vừa đúng dưới trần 28 byte của MEMO_TEXT,
	// nên không cần bảng ánh xạ memo→order và cũng không có chỗ nào để hai thứ lệch nhau.
	id := ksuid.New().String()

	instruction, err := cfg.BuildInstruction(amount, id)
	if err != nil {
		apierr.BadRequest(c, apierr.CodeDataInvalid, err.Error())
		return
	}

	row := models.OrderIntent{
		Id:                 id,
		ProductRef:         req.ProductRef,
		ExpectedAmount:     amount,
		AssetCode:          cfg.AssetCode,
		AssetIssuer:        cfg.AssetIssuer,
		DestinationAccount: cfg.DestinationAccount,
		Memo:               id,
		BuyerWallet:        req.BuyerWallet,
		Status:             models.OrderStatusAwaitingPayment,
	}
	if err := database.GetDb().WithContext(c).Create(&row).Error; err != nil {
		apierr.Internal(c, "create order failed: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"data": CreateOrderData{
			Order:       toOrderDto(&row),
			Instruction: instruction,
		},
	})
}

// GetOrder godoc
// @Summary      Đọc một order intent kèm trạng thái hiện tại
// @Tags         orders
// @Produce      json
// @Param        id   path      string  true  "Order id"
// @Success      200  {object}  OrderEnvelope
// @Failure      404  {object}  ErrorResponse
// @Router       /orders/{id} [get]
func GetOrder(c *gin.Context) {
	var row models.OrderIntent
	err := database.GetDb().WithContext(c).Where("id = ?", c.Param("id")).Take(&row).Error
	if err != nil {
		apierr.NotFound(c, "order not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "data": toOrderDto(&row)})
}

// ListOrders godoc
// @Summary      Liệt kê order intent của một ví buyer
// @Tags         orders
// @Produce      json
// @Param        buyer_wallet  query     string  true  "Địa chỉ ví buyer"
// @Param        limit         query     int     false "Mặc định 50, tối đa 200"
// @Success      200  {object}  OrderListEnvelope
// @Failure      400  {object}  ErrorResponse
// @Router       /orders [get]
func ListOrders(c *gin.Context) {
	wallet := c.Query("buyer_wallet")
	if !validAccount(wallet) {
		apierr.BadRequest(c, apierr.CodeDataInvalid, "buyer_wallet bắt buộc và phải là public key Stellar hợp lệ")
		return
	}
	limit := 50
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
		limit = min(v, 200)
	}

	var rows []models.OrderIntent
	err := database.GetDb().WithContext(c).
		Where("buyer_wallet = ?", wallet).
		Order("created_at DESC").Limit(limit).Find(&rows).Error
	if err != nil {
		apierr.Internal(c, "lookup failed: "+err.Error())
		return
	}

	items := make([]OrderDto, 0, len(rows))
	for i := range rows {
		items = append(items, toOrderDto(&rows[i]))
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "data": items})
}

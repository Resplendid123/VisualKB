package response

import (
	"learn/internal/domain"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CodeSuccess  = 0
	CodeInternal = 9999
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// Fail maps errors to standard response.
func Fail(c *gin.Context, err error) {
	de, ok := domain.AsDomainError(err)
	if ok {
		c.AbortWithStatusJSON(httpStatus(de.Type), Response{
			Code:    de.Code,
			Message: de.Message,
		})
		return
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
		Code:    CodeInternal,
		Message: err.Error(),
	})
}

func AbortUnauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, Response{
		Code:    CodeInternal,
		Message: message,
	})
}

func AbortBadRequest(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, Response{
		Code:    CodeInternal,
		Message: message,
	})
}

func httpStatus(typ string) int {
	switch typ {
	case domain.TypeInvalidArg:
		return http.StatusBadRequest
	case domain.TypeNotFound:
		return http.StatusNotFound
	case domain.TypeConflict:
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}

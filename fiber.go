package apitoolkit

import (
	"context"
	"time"

	fiber "github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (c *Client) FiberMiddleware(ctx *fiber.Ctx) error {
	msgID := uuid.Must(uuid.NewRandom())
	ctx.Locals(string(CurrentRequestMessageID), msgID)
	errorList := []ATError{}
	ctx.Locals(string(ErrorListCtxKey), &errorList)
	newCtx := context.WithValue(ctx.Context(), ErrorListCtxKey, &errorList)
	newCtx = context.WithValue(newCtx, CurrentRequestMessageID, msgID)
	ctx.SetUserContext(newCtx)

	start := time.Now()

	if err := ctx.Next(); err != nil {
		return err
	}

	respHeaders := map[string][]string{}
	for k, v := range ctx.GetRespHeaders() {
		respHeaders[k] = []string{v}
	}

	payload := c.buildFastHTTPPayload(GoFiberSDKType, start,
		ctx.Context(), ctx.Response().StatusCode(),
		ctx.Request().Body(), ctx.Response().Body(), respHeaders,
		ctx.AllParams(), ctx.Route().Path,
		c.config.RedactHeaders, c.config.RedactRequestBody, c.config.RedactResponseBody,
		errorList,
		msgID,
		nil,
		string(ctx.Context().Referer()),
	)

	c.PublishMessage(ctx.Context(), payload)
	return nil
}

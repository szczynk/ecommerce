package handler

import (
	"api-gateway-go/helper/response"
	"api-gateway-go/helper/timeout"
	"api-gateway-go/model"
	"api-gateway-go/service"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ShortenHandler struct {
	svc service.ShortenServiceI
}

func NewShortenHandler(svc service.ShortenServiceI) ShortenHandlerI {
	h := new(ShortenHandler)
	h.svc = svc
	return h
}

func (h *ShortenHandler) Get(ctx *gin.Context) {
	urlCtx, _ := ctx.Get("url")
	url, _ := urlCtx.(string)

	userIDCtx, _ := ctx.Get("userID")
	userID, _ := userIDCtx.(string)

	needBypassCtx, _ := ctx.Get("need_bypass")
	needBypass, _ := needBypassCtx.(bool)

	timeoutCtx, cancel := timeout.NewCtxTimeout()
	defer cancel()

	reqBody := ctx.Request.Body
	req, errReq := http.NewRequestWithContext(timeoutCtx, ctx.Request.Method, url, reqBody)
	if errReq != nil {
		_ = ctx.Error(errReq)
		response.NewJSONResErr(ctx, http.StatusInternalServerError, "", errReq.Error())
		return
	}
	if !needBypass {
		req.Header.Add("user-id", userID)
	}

	resp, errResp := http.DefaultClient.Do(req)
	if errResp != nil {
		_ = ctx.Error(errResp)
		response.NewJSONResErr(ctx, http.StatusInternalServerError, "", errResp.Error())
		return
	}
	defer resp.Body.Close()

	// Copy the response headers to the gateway response
	h.copyHeaders(ctx.Writer.Header(), resp.Header)

	respBody, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		_ = ctx.Error(errRead)
		response.NewJSONResErr(ctx, http.StatusInternalServerError, "", errRead.Error())
		return
	}

	var jsonGRes response.JSONGatewayRes
	if errJSONUn := json.Unmarshal(respBody, &jsonGRes); errJSONUn != nil {
		_ = ctx.Error(errJSONUn)
		response.NewJSONResErr(ctx, http.StatusInternalServerError, "", errJSONUn.Error())
		return
	}

	ctx.JSON(resp.StatusCode, jsonGRes)
}

func (h *ShortenHandler) copyHeaders(dst http.Header, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func (h *ShortenHandler) Shorten(ctx *gin.Context) {
	shortenReq := new(model.ShortenReq)
	if errJSON := ctx.ShouldBindJSON(&shortenReq); errJSON != nil {
		response.NewJSONResErr(ctx, http.StatusBadRequest, "", errJSON.Error())
		return
	}

	apiManagement, errSvc := h.svc.Create(shortenReq)
	if errSvc != nil {
		_ = ctx.Error(errSvc)
		response.NewJSONResErr(ctx, http.StatusInternalServerError, "", errSvc.Error())
		return
	}

	response.NewJSONRes(ctx, http.StatusOK, "", apiManagement)
}

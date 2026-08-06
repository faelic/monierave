package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	db "github.com/faelic/monierave/db/sqlc"
	"github.com/faelic/monierave/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type beneficiaryResponse struct {
	ID                       string    `json:"id"`
	Nickname                 string    `json:"nickname"`
	DestinationAccountID     string    `json:"destination_account_id"`
	Currency                 string    `json:"currency"`
	DestinationAccountStatus string    `json:"destination_account_status"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

type createBeneficiaryRequest struct {
	DestinationAccountID string `json:"destination_account_id" binding:"required"`
	Nickname             string `json:"nickname" binding:"required,max=50"`
}

type updateBeneficiaryRequest struct {
	Nickname string `json:"nickname" binding:"required,max=50"`
}

type listBeneficiaryRequest struct {
	PageID   int32 `form:"page_id" binding:"omitempty,min=1"`
	PageSize int32 `form:"page_size" binding:"omitempty,min=1,max=100"`
}

func (server *Server) createBeneficiary(ctx *gin.Context) {
	var req createBeneficiaryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	nickname, ok := normalizedBeneficiaryNickname(ctx, req.Nickname)
	if !ok {
		return
	}
	destinationPublicID, err := parsePublicID(req.DestinationAccountID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(ErrInvalidAccountID))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	result, err := server.store.CreateBeneficiaryTx(
		ctx,
		db.CreateBeneficiaryTxParams{
			Owner:                      authPayload.Username,
			DestinationAccountPublicID: destinationPublicID,
			Nickname:                   nickname,
		},
	)
	switch {
	case errors.Is(err, db.ErrAccountNotFound):
		ctx.JSON(http.StatusNotFound, errorResponse(ErrAccountNotFound))
		return
	case errors.Is(err, db.ErrAccountClosed):
		ctx.JSON(
			http.StatusConflict,
			codedErrorResponse("account_closed", ErrAccountClosed),
		)
		return
	case errors.Is(err, db.ErrBeneficiaryAlreadyExists):
		ctx.JSON(
			http.StatusConflict,
			codedErrorResponse("beneficiary_already_exists", ErrBeneficiaryAlreadyExists),
		)
		return
	case err != nil:
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	ctx.JSON(http.StatusCreated, beneficiaryResponse{
		ID:                       publicUUIDString(result.Beneficiary.ID),
		Nickname:                 result.Beneficiary.Nickname,
		DestinationAccountID:     publicUUIDString(result.DestinationAccount.PublicID),
		Currency:                 result.DestinationAccount.Currency,
		DestinationAccountStatus: result.DestinationAccount.Status,
		CreatedAt:                result.Beneficiary.CreatedAt.Time,
		UpdatedAt:                result.Beneficiary.UpdatedAt.Time,
	})
}

func (server *Server) listBeneficiaries(ctx *gin.Context) {
	var req listBeneficiaryRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	if req.PageID == 0 {
		req.PageID = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	rows, err := server.store.ListOwnedBeneficiaries(
		ctx,
		db.ListOwnedBeneficiariesParams{
			Owner:  authPayload.Username,
			Limit:  req.PageSize,
			Offset: (req.PageID - 1) * req.PageSize,
		},
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	response := make([]beneficiaryResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, beneficiaryResponseFromList(row))
	}
	ctx.JSON(http.StatusOK, response)
}

func (server *Server) updateBeneficiary(ctx *gin.Context) {
	beneficiaryID, ok := bindBeneficiaryID(ctx)
	if !ok {
		return
	}
	var req updateBeneficiaryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	nickname, ok := normalizedBeneficiaryNickname(ctx, req.Nickname)
	if !ok {
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	row, err := server.store.UpdateOwnedBeneficiaryNickname(
		ctx,
		db.UpdateOwnedBeneficiaryNicknameParams{
			Nickname:      nickname,
			BeneficiaryID: beneficiaryID,
			Owner:         authPayload.Username,
		},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		ctx.JSON(http.StatusNotFound, errorResponse(ErrBeneficiaryNotFound))
		return
	}
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	ctx.JSON(http.StatusOK, beneficiaryResponseFromUpdate(row))
}

func (server *Server) deleteBeneficiary(ctx *gin.Context) {
	beneficiaryID, ok := bindBeneficiaryID(ctx)
	if !ok {
		return
	}
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	deleted, err := server.store.DeleteOwnedBeneficiary(
		ctx,
		db.DeleteOwnedBeneficiaryParams{
			BeneficiaryID: beneficiaryID,
			Owner:         authPayload.Username,
		},
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}
	if deleted == 0 {
		ctx.JSON(http.StatusNotFound, errorResponse(ErrBeneficiaryNotFound))
		return
	}
	ctx.Status(http.StatusNoContent)
}

func beneficiaryResponseFromList(
	row db.ListOwnedBeneficiariesRow,
) beneficiaryResponse {
	return newBeneficiaryResponse(
		row.ID,
		row.Nickname,
		row.DestinationAccountPublicID,
		row.Currency,
		row.DestinationAccountStatus,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func beneficiaryResponseFromUpdate(
	row db.UpdateOwnedBeneficiaryNicknameRow,
) beneficiaryResponse {
	return newBeneficiaryResponse(
		row.ID,
		row.Nickname,
		row.DestinationAccountPublicID,
		row.Currency,
		row.DestinationAccountStatus,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func newBeneficiaryResponse(
	id pgtype.UUID,
	nickname string,
	destinationAccountID pgtype.UUID,
	currency string,
	status string,
	createdAt pgtype.Timestamptz,
	updatedAt pgtype.Timestamptz,
) beneficiaryResponse {
	return beneficiaryResponse{
		ID:                       publicUUIDString(id),
		Nickname:                 nickname,
		DestinationAccountID:     publicUUIDString(destinationAccountID),
		Currency:                 currency,
		DestinationAccountStatus: status,
		CreatedAt:                createdAt.Time,
		UpdatedAt:                updatedAt.Time,
	}
}

func bindBeneficiaryID(ctx *gin.Context) (pgtype.UUID, bool) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(ErrInvalidBeneficiaryID))
		return pgtype.UUID{}, false
	}
	return pgtype.UUID{Bytes: id, Valid: true}, true
}

func normalizedBeneficiaryNickname(
	ctx *gin.Context,
	value string,
) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" || len([]rune(value)) > 50 {
		ctx.JSON(http.StatusBadRequest, errorResponse(ErrInvalidBeneficiaryNickname))
		return "", false
	}
	return value, true
}

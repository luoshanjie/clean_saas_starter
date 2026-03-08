package resp

// 统一错误码（与 api_phase1.md 保持一致）
const (
	CodeOK                             = 0
	CodeUnauthorized                   = 40101
	CodeForbidden                      = 40301
	CodeIDORForbidden                  = 40302
	CodeValidation                     = 42201
	CodeVersionConflict                = 409001
	CodeVideoFileUnavailable           = 422101
	CodeInteractionTimestampOutOfRange = 422102
	CodeInteractionQuestionNotFound    = 422103
	CodeInteractionPointsExceedLimit   = 422104
	CodeInteractionTimestampDuplicated = 422105
	CodeRateLimited                    = 42901
	CodeServerError                    = 50000
)

package gconsts

import "shopble/common/comerr"

var (
	ErrorServerUnkown      comerr.OurErrorCode = comerr.NewOurErrorCode("error_server_unkown")
	ErrorIPBanned          comerr.OurErrorCode = comerr.NewOurErrorCode("error_ip_banned")
	ErrorDataInvalid       comerr.OurErrorCode = comerr.NewOurErrorCode("error_data_invalid")
	ErrorDataNotFound      comerr.OurErrorCode = comerr.NewOurErrorCode("error_data_not_found")
	ErrorTokenExpired      comerr.OurErrorCode = comerr.NewOurErrorCode("error_token_expried")
	ErrorExistsData        comerr.OurErrorCode = comerr.NewOurErrorCode("error_exists_data")
	ErrorFeatureNotSupport comerr.OurErrorCode = comerr.NewOurErrorCode("error_feature_not_support")
	ErrorDataDuplicated    comerr.OurErrorCode = comerr.NewOurErrorCode("error_data_duplicated")
	ErrorTimeOut           comerr.OurErrorCode = comerr.NewOurErrorCode("error_time_out")
)

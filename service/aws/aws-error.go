package aws

import (
	"errors"
	"fmt"

	smithy "github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// WrapError returns an error for an AWS API call, adding actionable hints for
// common failure modes.

// IMPORTANT: use this when calling aws sdk functions
func WrapError(operation string, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case IsAuthError(err):
		return fmt.Errorf("%s: %w\nhint: invalid AWS credentials — run: devbox setup", operation, err)
	case IsQuotaError(err):
		return fmt.Errorf("%s: %w\nhint: AWS account limit reached — request a quota increase in the AWS console", operation, err)
	case IsThrottlingError(err):
		return fmt.Errorf("%s: %w\nhint: AWS rate limit hit — wait a moment and retry", operation, err)
	case IsRegionError(err):
		return fmt.Errorf("%s: %w\nhint: resource not available in this region — check your region in: devbox setup", operation, err)
	case IsPermissionError(err):
		return fmt.Errorf("%s: %w\nhint: credentials are valid but lack permission for this operation — check your IAM policy", operation, err)
	default:
		return fmt.Errorf("%s: %w", operation, err)
	}
}

// IsAuthError reports whether err is an AWS authentication failure.
func IsAuthError(err error) bool {
	if hasErrorCode(err,
		"AuthFailure", "InvalidClientTokenId", "SignatureDoesNotMatch",
		"IncompleteSignature",
		"UnrecognizedClientException", "ExpiredToken", "ExpiredTokenException",
	) {
		return true
	}

	var respErr *smithyhttp.ResponseError
	if errors.As(err, &respErr) && respErr.HTTPStatusCode() == 401 {
		return true
	}

	return false
}

// IsPermissionError reports whether err is an IAM authorization failure.
func IsPermissionError(err error) bool {
	if hasErrorCode(err, "AccessDenied", "AccessDeniedException", "UnauthorizedOperation") {
		return true
	}

	var respErr *smithyhttp.ResponseError
	if errors.As(err, &respErr) && respErr.HTTPStatusCode() == 403 {
		return true
	}

	return false
}

// IsQuotaError reports whether err is an AWS account quota or limit failure.
func IsQuotaError(err error) bool {
	return hasErrorCode(err,
		"InstanceLimitExceeded",
		"VpcLimitExceeded",
		"VolumeLimitExceeded",
		"AddressLimitExceeded",
	)
}

// IsThrottlingError reports whether err is an AWS rate-limit failure.
func IsThrottlingError(err error) bool {
	return hasErrorCode(err,
		"RequestLimitExceeded",
		"Throttling",
		"ThrottlingException",
		"TooManyRequestsException",
	)
}

// IsRegionError reports whether err indicates a region, opt-in, or resource
// availability problem.
func IsRegionError(err error) bool {
	return hasErrorCode(err,
		"OptInRequired",
		"InvalidAMIID.NotFound",
		"InvalidSubnetID.NotFound",
		"InvalidVpcID.NotFound",
	)
}

func hasErrorCode(err error, codes ...string) bool {
	var apiErr smithy.APIError    // aws uses smithy.APIError for errors
	if !errors.As(err, &apiErr) { // extract the error from the response
		return false // if not of type smithy.APIError, return false - as we can't check the error code
	}
	code := apiErr.ErrorCode()
	for _, c := range codes {
		if code == c {
			return true
		}
	}
	return false
}

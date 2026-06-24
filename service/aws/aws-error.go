package aws

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"

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
	case IsNetworkError(err):
		return fmt.Errorf("%s: %w\nhint: network error — check your connection and retry when your network is back", operation, err)
	case IsServerError(err):
		return fmt.Errorf("%s: %w\nhint: AWS service temporarily unavailable — retry in a moment", operation, err)
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

	return hasHTTPStatus(err, 401)
}

// IsPermissionError reports whether err is an IAM authorization failure.
func IsPermissionError(err error) bool {
	if hasErrorCode(err, "AccessDenied", "AccessDeniedException", "UnauthorizedOperation") {
		return true
	}

	return hasHTTPStatus(err, 403)
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

// IsNetworkError reports whether err is a transport-level network failure.
func IsNetworkError(err error) bool {
	for err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return true
		}

		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return true
		}

		var opErr *net.OpError
		if errors.As(err, &opErr) {
			if opErr.Timeout() {
				return true
			}
			var dnsErr *net.DNSError
			if errors.As(opErr.Err, &dnsErr) {
				return true
			}
			if errors.Is(opErr.Err, syscall.ECONNRESET) || errors.Is(opErr.Err, syscall.ECONNREFUSED) {
				return true
			}
		}

		err = errors.Unwrap(err)
	}
	return false
}

// IsServerError reports whether err is a transient AWS server-side failure.
func IsServerError(err error) bool {
	if hasErrorCode(err,
		"ServiceUnavailable",
		"InternalError",
		"InternalFailure",
		"InternalServerError",
	) {
		return true
	}

	return hasHTTPStatusAtLeast(err, 500)
}

// IsRetryableError reports whether err is likely transient and worth retrying.
func IsRetryableError(err error) bool {
	return IsNetworkError(err) || IsThrottlingError(err) || IsServerError(err)
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

// extract the error code, sometimes the error is wrapped
// for each level of the error, check if the error code is in the list of codes
func hasErrorCode(err error, codes ...string) bool {
	for err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			code := apiErr.ErrorCode()
			for _, c := range codes {
				if code == c {
					return true
				}
			}
		}
		err = errors.Unwrap(err)
	}
	return false
}

// for each level of the error, check if the http status code is the same as the status code passed in
func hasHTTPStatus(err error, status int) bool {
	for err != nil {
		var respErr *smithyhttp.ResponseError
		if errors.As(err, &respErr) && respErr.HTTPStatusCode() == status {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}

// for each level of the error, check if the http status code is greater than or equal to the status code passed in
func hasHTTPStatusAtLeast(err error, status int) bool {
	for err != nil {
		var respErr *smithyhttp.ResponseError
		if errors.As(err, &respErr) && respErr.HTTPStatusCode() >= status {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}

// ShortMessage returns a concise, user-facing explanation for common AWS errors.
func ShortMessage(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case IsAuthError(err):
		return "invalid or expired credentials — run: devbox setup"
	case IsPermissionError(err):
		return "credentials valid but lack permission — check IAM policy"
	case IsThrottlingError(err):
		return "rate limited — retry in a moment"
	case IsNetworkError(err):
		return "network error — check your connection and retry when your network is back"
	case IsServerError(err):
		return "AWS service temporarily unavailable — retry in a moment"
	case IsQuotaError(err):
		return "AWS account limit reached"
	case IsRegionError(err):
		return "resource not available in this region — check: devbox setup"
	default:
		for e := err; e != nil; e = errors.Unwrap(e) {
			var apiErr smithy.APIError
			if errors.As(e, &apiErr) && apiErr.ErrorMessage() != "" {
				return apiErr.ErrorMessage()
			}
		}
		return err.Error()
	}
}

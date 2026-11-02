package workflow

func ApproveRequest(request Request, supervisor string) Request {
	if supervisor != "" {
		request.Status = "APPROVED"
	}
	return request
}

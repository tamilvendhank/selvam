package openai

type FileObject struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Filename  string `json:"filename"`
	Purpose   string `json:"purpose"`
	Status    string `json:"status"`
	Bytes     int64  `json:"bytes"`
	CreatedAt int64  `json:"created_at"`
}

type Batch struct {
	ID            string         `json:"id"`
	Object        string         `json:"object"`
	Status        string         `json:"status"`
	Endpoint      string         `json:"endpoint"`
	InputFileID   string         `json:"input_file_id"`
	OutputFileID  string         `json:"output_file_id"`
	ErrorFileID   string         `json:"error_file_id"`
	RequestCounts map[string]any `json:"request_counts"`
	Errors        any            `json:"errors"`
	Metadata      map[string]any `json:"metadata"`
	CreatedAt     int64          `json:"created_at"`
	CompletedAt   int64          `json:"completed_at"`
	FailedAt      int64          `json:"failed_at"`
	CancelledAt   int64          `json:"cancelled_at"`
	ExpiredAt     int64          `json:"expired_at"`
}

type apiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

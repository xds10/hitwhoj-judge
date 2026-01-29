package v1

type TaskReq struct {
	CPULimit            int64        `json:"cpu_limit" binding:"required"`
	MemLimit            int64        `json:"mem_limit" binding:"required"`
	StackLimit          int64        `json:"stack_limit"`
	ProcLimit           int64        `json:"proc_limit"`
	CodeFile            string       `json:"code_file" binding:"required"`
	CodeLanguage        string       `json:"code_language" binding:"required"`
	IsSpecial           *bool        `json:"is_special" binding:"required"`
	SpecialCodeFile     string       `json:"special_code_file" `
	SpecialCodeFileName string       `json:"special_code_file_name" `
	Bucket              string       `json:"bucket" binding:"required"`
	CheckPoints         []checkPoint `json:"check_points" binding:"required"`
}

type checkPoint struct {
	InputFile  string `json:"input"`
	OutputFile string `json:"output"`
}

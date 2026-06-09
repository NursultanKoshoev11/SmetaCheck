package domain

type Severity string

const(
 Critical Severity="critical"
 Warning Severity="warning"
 Info Severity="info"
)

type Issue struct{
 Row int `json:"row"`
 Severity Severity `json:"severity"`
 Type string `json:"type"`
 Message string `json:"message"`
 Recommendation string `json:"recommendation"`
}

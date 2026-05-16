package vue

import (
	"github.com/tamnguyendinh/avmatrix-go/internal/providers/sfc"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type Request = sfc.Request

func Extract(request Request) (scopeir.ScopeIR, error) {
	return sfc.Extract(request, sfc.Options{
		Name:            "vue",
		Language:        scanner.Vue,
		ScriptExtractor: sfc.ExtractHTMLScript,
	})
}

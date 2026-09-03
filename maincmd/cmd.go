package maincmd

import (
	"fmt"
	"os"
)

// executor được set trong init() của file có build tag tương ứng (xem shopble.go).
// Default này chạy khi binary build THIẾU tag: không command nào được đăng ký, cobra
// không bao giờ chạy, và binary thoát 0 im lặng — kể cả với --help. Im lặng ở đây tốn
// của người ta hàng giờ đoán mò, nên nó phải nói ra mình bị build sai.
var executor func() = func() {
	fmt.Fprintln(os.Stderr, "binary built without a project build tag — no commands registered.")
	fmt.Fprintln(os.Stderr, "rebuild with: go build --tags shopble -o shopble ./main.go")
	os.Exit(2)
}

func Execute() {
	executor()
}

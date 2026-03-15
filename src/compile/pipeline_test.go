package compile

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const snapshotDir = ".testdata"

func RunPipeline(t *testing.T, source string) *CompilationResult {
	t.Helper()

	var buf bytes.Buffer

	opts := DefaultPipelineOptions()
	opts.Source = source
	opts.Verbose = true
	opts.OutputInstr = true
	opts.Output = &buf

	result, err := Pipeline(opts)

	if err != nil {
		fmt.Fprintf(&buf, "Compilation failed: %s\n", err)
	}
	for _, perr := range result.Diagnostics {
		fmt.Fprintf(&buf, "  ParseErr: %s\n", perr.Error())
	}
	for _, serr := range result.SemanticErrors {
		fmt.Fprintf(&buf, "  SemErr: %s\n", serr.Error())
	}

	assertSnapshot(t, buf.Bytes())

	return result
}

// assertSnapshot compares got against the snapshot file for the current test.
// If the file does not exist it is created and the test passes.
// If the content differs, the first differing line is reported and the test fails.
func assertSnapshot(t *testing.T, got []byte) {
	t.Helper()

	snapshotFile := filepath.Join(snapshotDir, t.Name()+".txt")

	existing, err := os.ReadFile(snapshotFile)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(snapshotDir, 0755); err != nil {
			t.Fatalf("snapshot: cannot create directory %s: %v", snapshotDir, err)
		}
		if err := os.WriteFile(snapshotFile, got, 0644); err != nil {
			t.Fatalf("snapshot: cannot write %s: %v", snapshotFile, err)
		}
		t.Logf("snapshot: created %s", snapshotFile)
		return
	}
	if err != nil {
		t.Fatalf("snapshot: cannot read %s: %v", snapshotFile, err)
	}

	if bytes.Equal(existing, got) {
		return
	}

	// Report the first differing line.
	wantLines := splitLines(existing)
	gotLines := splitLines(got)
	for i := 0; i < len(wantLines) || i < len(gotLines); i++ {
		var w, g string
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w != g {
			t.Errorf("snapshot %s differs at line %d:\n  want: %q\n   got: %q",
				snapshotFile, i+1, w, g)
			return
		}
	}
	// Lengths differ but no line mismatch found (trailing newline difference).
	t.Errorf("snapshot %s: content length differs (want %d bytes, got %d bytes)",
		snapshotFile, len(existing), len(got))
}

func splitLines(b []byte) []string {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}

func Test_Pipeline_Factorial(t *testing.T) {
	sourceCode := `
		factorial: (n: u8) u8 {
			if n <= 1 {
				ret 1
			}
			ret n * factorial(n - 1)
		}
	`

	RunPipeline(t, sourceCode)
}

func Test_Pipeline_Max(t *testing.T) {
	sourceCode := `
		max: (a: u8, b: u8) u8 {
			if a > b {
				ret a
			} else {
				ret b
			}
		}
	`

	RunPipeline(t, sourceCode)
}

func Test_Pipeline_ArrMax(t *testing.T) {
	sourceCode := `
		arrMax: (arr: u8[]) u8 {
			if arr[0] > arr[1] {
				ret arr[0]
			} else {
				ret arr[1]
			}
		}
	`

	RunPipeline(t, sourceCode)
}

func Test_Pipeline_ForLoop(t *testing.T) {
	sourceCode := `
		forLoop: (n: u8) u8 {
			for i := 0; i < n; i++ {
				n = n + i
			}
			ret n
		}
	`

	RunPipeline(t, sourceCode)
}

func Test_Pipeline_Variables(t *testing.T) {
	sourceCode := `variables: (p: u8) u8 {
		x := p + 42
		y := x + 42
		ret x + y + p
	}`

	RunPipeline(t, sourceCode)
}

func Test_Pipeline_LocalArray(t *testing.T) {
	sourceCode := `localArr: () u16 {
		x: u8[] = [1, 2, 3]
		y: u16[2] = [1234, 5678]
		ret x[0] + y[0]
	}`

	RunPipeline(t, sourceCode)
}
func Test_Pipeline_LocalArray8(t *testing.T) {
	sourceCode := `localArr8: () u8 {
		x: u8[] = [1, 2, 3]
		ret x[1]
	}`

	RunPipeline(t, sourceCode)
}
func Test_Pipeline_LocalArray16(t *testing.T) {
	sourceCode := `localArr16: () u16 {
		y: u16[2] = [1234, 5678]
		ret y[1]
	}`

	RunPipeline(t, sourceCode)
}

func Test_Pipeline_ArraySum(t *testing.T) {
	sourceCode := `arrSum: (arr: u8[]) u8 {
		l := @len(arr)
		sum := 0
		for i := 0; i < l; i++ {
			sum = sum + arr[i]
		}
		ret sum
	}`

	RunPipeline(t, sourceCode)
}

func Test_Pipeline_Reverse(t *testing.T) {
	sourceCode := `reverse: (arr: u8[]) {
		l := @len(arr)
		for i := 0; i < l / 2 ; i++ {
			j := l - 1 - i
			tmp := arr[i]
			arr[i] = arr[j]
			arr[j] = tmp
		}
	}`

	RunPipeline(t, sourceCode)
}

func Test_Pipeline_Struct(t *testing.T) {
	sourceCode := `struct Point {
		x: u8,
		y: u8
	}
	pt: (pt: Point*) u8 {
		ret pt.x + pt.y
	}`

	RunPipeline(t, sourceCode)
}

package engine

import "testing"

func TestTopLevelReturnReturnsError(t *testing.T) {
	if err := Run(`return 1;`, "return-test.vs"); err == nil {
		t.Fatal("return di luar fungsi seharusnya error")
	}
}

func TestLoopControlCannotEscapeFunction(t *testing.T) {
	if err := Run(`function bad() { break; } bad();`, "break-test.vs"); err == nil {
		t.Fatal("break dari fungsi seharusnya error")
	}
	if err := Run(`function bad() { continue; } bad();`, "continue-test.vs"); err == nil {
		t.Fatal("continue dari fungsi seharusnya error")
	}
}

func TestMultipleVideoBlocksRejected(t *testing.T) {
	source := `video 320 240 10 1s { background("#000000"); }
video 320 240 10 1s { background("#000000"); }
export video "out.mp4";`
	if err := Run(source, "multi-video-test.vs"); err == nil {
		t.Fatal("multiple video blocks seharusnya ditolak")
	}
}

func TestNestedVideoAndExportRejected(t *testing.T) {
	if err := Run(`if true { video 320 240 10 1s { background("#000"); } }`, "nested-video-test.vs"); err == nil {
		t.Fatal("video nested seharusnya error")
	}
	if err := Run(`if true { export video "out.mp4"; }`, "nested-export-test.vs"); err == nil {
		t.Fatal("export nested seharusnya error")
	}
}

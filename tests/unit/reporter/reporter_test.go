package reporter_test

import (
	"clip/errors"
	"clip/processors/reporter"
	"os"
	"testing"
	"time"
)

func TestReportBuilder(t *testing.T) {
	r := reporter.NewReport()
	r.NewReporter(".pdf")
	errCh := make(chan error, 2)
	defer close(errCh)
	t.Run("singular", func(t *testing.T) {
		r.Content = append(r.Content, &reporter.ReportContent{Mname: "test", Body: "some words\nextra"})
		done := make(chan struct{})
		defer close(done)
		go func() {
			select {
			case err := <-errCh:
				t.Fatalf("Unexpected error: %v", err)
			case <-time.After(30 * time.Second):
				t.Fatal("Report generation took too long")
			case <-done:
				return
			}
		}()

		r.Reporter.CreateReport("C:\\Users\\w\\Desktop\\1.pdf", r.Content, errCh)
		done <- struct{}{}
	})
	t.Run("few modules", func(t *testing.T) {
		r.Content = append(r.Content, &reporter.ReportContent{Mname: "test", Body: "some words\nextra"})
		r.Content = append(r.Content, &reporter.ReportContent{Mname: "test2", Body: "MOOOOOORRRRRRRRRRRRRRRRRR words\nextra\n\n\n\n\n\thiiiiiii"})
		r.Content = append(r.Content, &reporter.ReportContent{Mname: "test2", Body: "link: \n\tgoogle.com \n1233"})
		done := make(chan struct{})
		defer close(done)
		go func() {
			select {
			case err := <-errCh:
				t.Fatalf("Unexpected error: %v", err)
			case <-time.After(30 * time.Second):
				t.Fatal("Report generation took too long")
			case <-done:
				return
			}
		}()

		r.Reporter.CreateReport("C:\\Users\\w\\Desktop\\1.pdf", r.Content, errCh)
		done <- struct{}{}
	})
	t.Run("catch error", func(t *testing.T) {
		r.Content = append(r.Content, &reporter.ReportContent{Mname: "test", Body: "some words\nextra"})

		done := make(chan struct{})
		defer close(done)
		go func() {
			select {
			case err := <-errCh:
				if err != nil {
					if customErr, ok := err.(*errors.Error); ok {
						if customErr.Code == errors.Code("report_file_writing_error") {
							t.Logf("Expected error occurred: %s", customErr.Code)
							done <- struct{}{}
						} else {
							t.Fatalf("Unexpected error code: %s", customErr.Code)
						}
					} else {
						t.Fatalf("Unexpected error type: %v", err)
					}
				}
			case <-time.After(30 * time.Second):
				t.Fatal("Report generation took too long")
			case <-done:
				return
			}
		}()
		f, err := os.OpenFile("C:\\Users\\w\\Desktop\\1.pdf", os.O_RDWR|os.O_CREATE, 0777)
		if err != nil {
			errCh <- err
		}
		defer f.Close()
		r.Reporter.CreateReport("C:\\Users\\w\\Desktop\\1.pdf", r.Content, errCh)
		done <- struct{}{}
	})
}

package output

// rawOutputDecorator sets up the writer
// most commonly this will be a bytes.Buffer which is not concurrency safe
// mu property locks it from multiple writes
type rawOutputDecorator struct {
	w *SafeWriter
}

func newRawOutputWriter(w *SafeWriter) *rawOutputDecorator {
	return &rawOutputDecorator{w: w}
}

func (d *rawOutputDecorator) WriteHeader() error {
	return nil
}

func (d *rawOutputDecorator) Write(b []byte) (int, error) {
	return d.w.Write(b)
}

func (d *rawOutputDecorator) WriteFooter() error {
	return nil
}

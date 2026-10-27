package metric

type NoOp struct{}

func (NoOp) Inc(string, int) {}

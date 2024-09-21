package HastenServer

type ComputeS1 struct {
}

func NewComputeS1() *ComputeS1 {
	return &ComputeS1{}
}

func (c *ComputeS1) Abc(a int, b int) int {
	return a + b
}

func (c *ComputeS1) Abs(a int, b int) int {
	return a - b
}

type TwoOperands struct {
	A int
	B int
}

func (c *ComputeS1) Add(twoOperands TwoOperands, output *int) error {
	*output = twoOperands.A + twoOperands.B
	return nil
}

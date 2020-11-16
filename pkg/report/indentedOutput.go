package report

import "strings"

const (
	colorReset = "\033[0m"
	colorRed   = "\033[31m"
	colorGreen = "\033[32m"
)

func NewIndentedOutput(indentationStep int, color bool) IndentedOutput {
	return IndentedOutput{
		color:            color,
		output:           "",
		indentationLevel: 0,
		indentationSize:  indentationStep,
	}
}

type IndentedOutput struct {
	color            bool
	output           string
	indentationLevel int
	indentationSize  int
}

func (o *IndentedOutput) SetIndentation(indentationLevel int) {
	o.indentationLevel = indentationLevel
}

func (o *IndentedOutput) IncreaseIndentation() {
	o.indentationLevel++
}

func (o *IndentedOutput) DecreaseIndentation() {
	o.indentationLevel--
}

func (o *IndentedOutput) ResetIndentation() {
	o.indentationLevel = 0
}

func (o *IndentedOutput) currentIndentation() string {
	return strings.Repeat(" ", o.indentationLevel*o.indentationSize)
}

func (o *IndentedOutput) AddLine(line string) {
	o.output += o.currentIndentation() + line + "\n"
}

func (o *IndentedOutput) AddErrorLine(line string) {
	if o.color {
		o.addColoredLine(line, colorRed)
	} else {
		o.AddLine(line)
	}
}

func (o *IndentedOutput) AddSuccessLine(line string) {
	if o.color {
		o.addColoredLine(line, colorGreen)
	} else {
		o.AddLine(line)
	}
}

func (o *IndentedOutput) addColoredLine(line string, color string) {
	o.output += color + o.currentIndentation() + line + colorReset + "\n"
}

func (o *IndentedOutput) AddTooPreviousLine(line string) {
	o.output = o.output[:len(o.output)-1] + line + "\n"
}

func (o *IndentedOutput) WriteErrors(errors []error) {
	for _, err := range errors {
		o.AddErrorLine("- " + err.Error())
	}
}

func (o *IndentedOutput) Text() string {
	return o.output
}

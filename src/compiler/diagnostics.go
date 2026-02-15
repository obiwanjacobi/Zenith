package compiler

import (
	"fmt"
)

type Location struct {
	Index  int // file stream index
	Line   int // code line
	Column int // column on line
}

var locationZero = Location{0, 0, 0}

type PipelinePhase uint8

const (
	PipelineInternal PipelinePhase = iota
	PipelineTokenizer
	PipelineParser
	PipelineSemanticAnalysis
	PipelineControlFlowAnalysis
	PipelineInstructionSelection
)

type DiagnosticSeverity uint8

const (
	SeverityCritical DiagnosticSeverity = iota
	SeverityError
	SeverityWarning
	SeverityInfo
	SeverityVerbose
)

type Diagnostic struct {
	Source   string
	Message  string
	Location Location
	Phase    PipelinePhase
	Severity DiagnosticSeverity
}

func NewDiagnostic(source, message string, location Location, phase PipelinePhase, severity DiagnosticSeverity) *Diagnostic {
	return &Diagnostic{
		Source:   source,
		Message:  message,
		Location: location,
		Phase:    phase,
		Severity: severity,
	}
}

func (d *Diagnostic) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", d.Source, d.Location.Line, d.Location.Column, d.Message)
}

func (d *Diagnostic) String() string {
	return fmt.Sprintf("%T", d)
}

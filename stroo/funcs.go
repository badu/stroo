package stroo

type FunctionInfo struct {
	Name         string
	ReturnType   string
	ReceiverName string
	ReceiverType string
	Package      string
	PackagePath  string
	IsExported   bool
}

type Methods []*FunctionInfo

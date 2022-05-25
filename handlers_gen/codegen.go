package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

const (
	filePatchIn  = "api.go"
	filePatchOut = "api_handlers.go"
)
const (
	validatorLabelRequired  = "required"
	validatorLabelParamName = "paramname"
	validatorLabelEnum      = "enum"
	validatorLabelDefault   = "default"
	validatorLabelMin       = "min"
	validatorLabelMax       = "max"
)

const (
	responseErrorFuncRaw = `func responseError(rw http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	type CR map[string]interface{}

	apiErr, ok := err.(ApiError)
	if ok {
		rw.WriteHeader(apiErr.HTTPStatus)
	}
	responseMap := CR{"error": err.Error()}
	response, _ := json.Marshal(responseMap)
	rw.Write(response)
}`
	responseResultFuncRaw = `func responseResult(rw http.ResponseWriter, err error, result interface{}) {
	type CR map[string]interface{}
	textErr := ""

	if err != nil {
		textErr = err.Error()
	}

	responseMap := CR{
		"error":    textErr,
		"response": result,
	}

	response, err := json.Marshal(responseMap)
	if err != nil {
		fmt.Println(err)
	}
	rw.Write(response)
}`
)

func main() {
	hc, err := NewHandlersCodegen(filePatchIn, filePatchOut)
	if err != nil {
		return
	}
	hc.GenerateAndWrite()
}

type handlersCodegen struct {
	filePatchIn            string
	filePatchOut           string
	sourceFileBuffer       []byte
	baseNode               *ast.File
	out                    *os.File
	needsMethods           needsMethods
	needsValidateStructMap needsValidateStructMap
}

func NewHandlersCodegen(filePatchIn, filePatchOut string) (*handlersCodegen, error) {
	baseNode, err := parser.ParseFile(token.NewFileSet(), filePatchIn, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	out, err := os.Create(filePatchOut)
	if err != nil {
		return nil, err
	}

	sourceFileBuffer, err := os.ReadFile(filePatchIn)
	if err != nil {
		return nil, err
	}

	return &handlersCodegen{
		filePatchIn:            filePatchIn,
		filePatchOut:           filePatchOut,
		sourceFileBuffer:       sourceFileBuffer,
		baseNode:               baseNode,
		out:                    out,
		needsMethods:           needsMethods{},
		needsValidateStructMap: needsValidateStructMap{},
	}, err
}

func (hc *handlersCodegen) GenerateAndWrite() error {
	if err := hc.packageAndImportWrite(); err != nil {
		return err
	}

	for _, decl := range hc.baseNode.Decls {
		if ok := hc.needsMethods.AddDecl(decl, hc.sourceFileBuffer); !ok {
			hc.needsValidateStructMap.AddDecl(decl)
		}
	}

	if len(hc.needsMethods) > 0 {
		hc.needsMethods.MethodsWrapperWrite(hc.out, hc.sourceFileBuffer)
	}
	if len(hc.needsValidateStructMap) > 0 {
		hc.needsValidateStructMap.StructValidationWrite(hc.out, hc.sourceFileBuffer)
	}
	return nil
}

func (hc *handlersCodegen) packageAndImportWrite() (err error) {
	_, err = fmt.Fprintln(hc.out, "package "+hc.baseNode.Name.Name)
	_, err = fmt.Fprintln(hc.out)
	_, err = fmt.Fprintln(hc.out, "import (")
	_, err = fmt.Fprintln(hc.out, "\t\"encoding/json\"")
	_, err = fmt.Fprintln(hc.out, "\t\"errors\"")
	_, err = fmt.Fprintln(hc.out, "\t\"fmt\"")
	_, err = fmt.Fprintln(hc.out, "\t\"net/http\"")
	_, err = fmt.Fprintln(hc.out, "\t\"strconv\"")
	_, err = fmt.Fprintln(hc.out, ")")
	_, err = fmt.Fprintln(hc.out)
	return
}

type needsValidateStructMap map[string]*ast.StructType

func (nvs needsValidateStructMap) AddDecl(decl interface{}) bool {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok {
		return false
	}
	for _, spec := range genDecl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}
		for _, spec := range structType.Fields.List {
			if spec.Tag == nil {
				continue
			}
			if strings.Contains(spec.Tag.Value, "apivalidator") {
				nvs[typeSpec.Name.Name] = structType
				return true
			}
		}
	}
	return false
}

func (nvs needsValidateStructMap) StructValidationWrite(out *os.File, src []byte) {
	for name, structDecl := range nvs {
		if structDecl == nil {
			return
		}
		firstSymReceiverName := getFirstSymFromString(name)

		fmt.Fprintf(out, "func (%s *%s) FilingAndValidate(r *http.Request) error {\n", firstSymReceiverName, name)
		fmt.Fprintln(out, "\tvar err error")
		fmt.Fprintln(out, "\tfmt.Println(err)")

		for _, field := range structDecl.Fields.List {
			fieldType := astFieldToString(src, field)

			//заполнение полей
			switch fieldType {
			case "int":
				fmt.Fprintf(out, "%s.%s,err = strconv.Atoi(r.FormValue(\"%s\"))\n", firstSymReceiverName, field.Names[0], strings.ToLower(field.Names[0].Name))
				fmt.Fprintln(out, "if err != nil{")
				fmt.Fprintln(out, "return errors.New(\"age must be int\")")
				fmt.Fprintln(out, "}")

			case "string":
				fmt.Fprintf(out, "%s.%s = r.FormValue(\"%s\")\n", firstSymReceiverName, field.Names[0], strings.ToLower(field.Names[0].Name))
			}

			validatorLabels := getValitatorParams(field.Tag.Value)
			if validatorLabels == nil {
				continue
			}

			//работа с параметрами валидатора
			for _, keyValue := range validatorLabels {

				switch keyValue.key {
				case validatorLabelDefault:
					switch fieldType {
					case "string":
						fmt.Fprintf(out, "if %s.%s == \"\"{\n", firstSymReceiverName, field.Names[0])
						fmt.Fprintf(out, "%s.%s = \"%s\"\n", firstSymReceiverName, field.Names[0], keyValue.value)
						fmt.Fprintln(out, "}")
					case "int":
						fmt.Fprintf(out, "if %s.%s == 0{\n", firstSymReceiverName, field.Names[0])
						fmt.Fprintf(out, "%s.%s,err = strconv.Atoi(r.FormValue(%s)\n", firstSymReceiverName, field.Names[0], keyValue.value)
						fmt.Fprintln(out, "if err != nil{")
						fmt.Fprintln(out, "return errors.New(\"age must be int\")")
						fmt.Fprintln(out, "}")
					}
				case validatorLabelParamName:
					fmt.Fprintf(out, "%s.%s = r.FormValue(\"%s\")\n", firstSymReceiverName, field.Names[0], keyValue.value)
				case validatorLabelEnum:
					paramForErr := strings.ReplaceAll(keyValue.value, "|", ", ")
					params := strings.Split(keyValue.value, "|")

					fmt.Fprintln(out, "isTrue:=false")
					for _, paramname := range params {
						switch fieldType {
						case "string":
							fmt.Fprintf(out, "if %s.%s == \"%s\"{\n", firstSymReceiverName, field.Names[0], paramname)
							fmt.Fprintln(out, "isTrue=true")
							fmt.Fprintln(out, "}")
						case "int":
							fmt.Fprintf(out, "if %s.%s == strconv.Atoi(\"%s\"){\n", firstSymReceiverName, field.Names[0], paramname)
							fmt.Fprintln(out, "isTrue=true")
							fmt.Fprintln(out, "}")
						}
					}
					fmt.Fprintln(out, "if !isTrue{")
					fmt.Fprintf(out, "return errors.New(\"%s must be one of [%s]\")\n", strings.ToLower(field.Names[0].Name), paramForErr)
					fmt.Fprintln(out, "}")
				case validatorLabelMin:
					switch fieldType {
					case "string":
						fmt.Fprintf(out, "if len(%s.%s) < %s{\n", firstSymReceiverName, field.Names[0], keyValue.value)
						fmt.Fprintf(out, "return errors.New(\"%s len must be >= %s\")\n", strings.ToLower(field.Names[0].Name), keyValue.value)
						fmt.Fprintln(out, "}")
					case "int":
						fmt.Fprintf(out, "if %s.%s < %s{\n", firstSymReceiverName, field.Names[0], keyValue.value)
						fmt.Fprintf(out, "return errors.New(\"%s must be >= %s\")\n", strings.ToLower(field.Names[0].Name), keyValue.value)
						fmt.Fprintln(out, "}")
					}
				case validatorLabelMax:
					switch fieldType {
					case "string":
						fmt.Fprintf(out, "if len(%s.%s) > %s{\n", firstSymReceiverName, field.Names[0], keyValue.value)
						fmt.Fprintf(out, "return errors.New(\"%s len must be <= %s\")\n", strings.ToLower(field.Names[0].Name), keyValue.value)
						fmt.Fprintln(out, "}")
					case "int":
						fmt.Fprintf(out, "if %s.%s > %s{\n", firstSymReceiverName, field.Names[0], keyValue.value)
						fmt.Fprintf(out, "return errors.New(\"%s must be <= %s\")\n", strings.ToLower(field.Names[0].Name), keyValue.value)
						fmt.Fprintln(out, "}")
					}
				case validatorLabelRequired:
					switch fieldType {
					case "string":
						fmt.Fprintf(out, "if %s.%s == \"\"{\n", firstSymReceiverName, field.Names[0])
					case "int":
						fmt.Fprintf(out, "if %s.%s == 0{\n", firstSymReceiverName, field.Names[0])
					}

					fmt.Fprintf(out, "return errors.New(\"%s must me not empty\")\n", strings.ToLower(field.Names[0].Name))
					fmt.Fprintln(out, "}")
				}
			}
		}

		fmt.Fprintln(out, "\treturn nil")
		fmt.Fprintln(out, "}")
		fmt.Fprintln(out)
	}
}

func getValitatorParams(tagValue string) (result []struct{ key, value string }) {
	validatorText := strings.ReplaceAll(tagValue, "`", "")
	validatorText = strings.ReplaceAll(validatorText, "apivalidator:", "")
	validatorText = strings.ReplaceAll(validatorText, "\"", "")
	if validatorText == "" {
		return nil
	}

	params := strings.Split(validatorText, ",")
	for _, param := range params {
		kV := strings.Split(param, "=")
		switch len(kV) {
		case 0:
			continue
		case 1:
			result = append(result, struct{ key, value string }{key: kV[0], value: ""})
		case 2:
			result = append(result, struct{ key, value string }{key: kV[0], value: kV[1]})
		}
	}

	return result
}

type needsMethods map[string][]needsMethod

type needsMethod struct {
	method       *ast.FuncDecl
	methodParams paramCodegenMethod
}

type paramCodegenMethod struct {
	Url        string `json:"url"`
	Auth       bool   `json:"auth"`
	Method     string `json:"method"`
	PapaStruct string
}

func (nm needsMethods) AddDecl(decl interface{}, src []byte) bool {
	g, ok := decl.(*ast.FuncDecl)
	if !ok {
		return false
	}
	if g.Doc == nil {
		return false
	}
	for _, dock := range g.Doc.List {
		commentText := dock.Text
		if !strings.HasPrefix(commentText, "// apigen:api") {
			return false
		}
		commentText = strings.ReplaceAll(commentText, "// apigen:api ", "")

		paramCodegenMethod := paramCodegenMethod{}
		json.Unmarshal([]byte(commentText), &paramCodegenMethod)
		paramCodegenMethod.PapaStruct = astFieldToString(src, g.Recv.List[0])
		nm[paramCodegenMethod.PapaStruct] = append(nm[paramCodegenMethod.PapaStruct], needsMethod{method: g, methodParams: paramCodegenMethod})
	}
	return true
}

func (nm needsMethods) MethodsWrapperWrite(out *os.File, src []byte) {
	nm.ServeHttpGenerate(out)

	for receiver, method := range nm {
		firstSymReceiverName := getFirstSymFromString(receiver)

		for _, m := range method {

			fmt.Fprintf(out, "func (%s %s) %s(rw http.ResponseWriter, r *http.Request) {\n", firstSymReceiverName, m.methodParams.PapaStruct, strings.ToLower(m.method.Name.Name))

			if m.methodParams.Auth {
				fmt.Fprintln(out, "\tif r.Header.Get(\"X-Auth\") != \"100500\" {")
				fmt.Fprintln(out, "\t\tresponseError(rw, ApiError{HTTPStatus: http.StatusForbidden, Err: errors.New(\"unauthorized\")})")
				fmt.Fprintln(out, "\t\treturn")
				fmt.Fprintln(out, "\t}")
			}

			if m.methodParams.Method != "" {
				fmt.Fprintf(out, "\tif r.Method != \"%s\" {\n", m.methodParams.Method)
				fmt.Fprintln(out, "\t\tresponseError(rw, ApiError{HTTPStatus: http.StatusNotAcceptable, Err: errors.New(\"bad method\")})")
				fmt.Fprintln(out, "\t\treturn")
				fmt.Fprintln(out, "\t}")
			}

			methodParamsSlice := make([]string, 0, 2)
			for _, params := range m.method.Type.Params.List {
				variableName := strings.ToLower(strings.ReplaceAll(astFieldToString(src, params), ".", ""))
				if variableName == "contextcontext" {
					methodParamsSlice = append(methodParamsSlice, "nil")
					continue
				}

				methodParamsSlice = append(methodParamsSlice, variableName)

				fmt.Fprintf(out, "\t%s := %s{}\n", variableName, astFieldToString(src, params))
				fmt.Fprintf(out, "\tif err := %s.FilingAndValidate(r); err != nil {\n", variableName)
				fmt.Fprintln(out, "\t\tresponseError(rw, ApiError{HTTPStatus: http.StatusBadRequest,Err:err})")
				fmt.Fprintln(out, "\t\treturn")
				fmt.Fprintln(out, "\t}")
			}

			methodParamsString := paramStringSliceToFormatString(methodParamsSlice)

			fmt.Fprintf(out, "\tresponse, err := %s.%s(%s)\n", firstSymReceiverName, m.method.Name.Name, methodParamsString)
			fmt.Fprintln(out, "\tif err != nil {")
			fmt.Fprintln(out, "\t\tapiErr, ok := err.(ApiError)")
			fmt.Fprintln(out, "\t\tif ok {")
			fmt.Fprintln(out, "\t\t\tresponseError(rw, apiErr)")
			fmt.Fprintln(out, "\t\t\treturn")
			fmt.Fprintln(out, "\t\t}")
			fmt.Fprintln(out, "\t\tresponseError(rw, ApiError{HTTPStatus: http.StatusInternalServerError, Err:err})")
			fmt.Fprintln(out, "\t\treturn")
			fmt.Fprintln(out, "\t}")
			fmt.Fprintln(out, "\tresponseResult(rw, err, response)")
			fmt.Fprintln(out, "}")
			fmt.Fprintln(out)
		}
	}

	fmt.Fprintln(out, responseErrorFuncRaw)
	fmt.Fprintln(out)
	fmt.Fprintln(out, responseResultFuncRaw)
	fmt.Fprintln(out)

}

func (nm needsMethods) ServeHttpGenerate(out *os.File) {
	for receiver, method := range nm {
		firstSymReceiverName := getFirstSymFromString(receiver)

		fmt.Fprintf(out, "func (%s %s) ServeHTTP(rw http.ResponseWriter, r *http.Request) {\n", firstSymReceiverName, receiver)
		fmt.Fprintln(out, "\tswitch r.URL.Path {")

		for _, m := range method {
			fmt.Fprintf(out, "\tcase \"%s\":\n", m.methodParams.Url)
			fmt.Fprintf(out, "\t\t%s.%s(rw, r)\n", firstSymReceiverName, strings.ToLower(m.method.Name.Name))
		}

		fmt.Fprintln(out, "\tdefault:")
		fmt.Fprintln(out, "\t\tresponseError(rw, ApiError{HTTPStatus:http.StatusNotFound,Err: errors.New(\"unknown method\")})")
		fmt.Fprintln(out, "\t}")
		fmt.Fprintln(out, "}")
		fmt.Fprintln(out)
	}
}

func astFieldToString(src []byte, field *ast.Field) string {
	typeExpr := field.Type
	start := typeExpr.Pos() - 1
	end := typeExpr.End() - 1
	return string(src)[start:end]
}

func getFirstSymFromString(str string) string {
	for _, ch := range str {
		if ch != '*' {
			return strings.ToLower(string(ch))
		}
	}
	return ""
}

func paramStringSliceToFormatString(params []string) string {
	paramsString := ""
	for _, param := range params {
		if paramsString != "" {
			paramsString += ", "
		}
		paramsString += param
	}
	return paramsString
}

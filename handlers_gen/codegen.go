package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
)

const (
	filePatchIn  = "api.go"
	filePatchOut = "api_handlers.go"
)
const (
	validatorLabelRequired  = "required"
	validatorLabelParamname = "paramname="
	validatorLabelEnum      = "enum="
	validatorLabelDefault   = "default="
	validatorLabelMin       = "min="
	validatorLabelMax       = "max="
)

var (
	responseErrorFuncRaw = `func responseError(rw http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	type CR map[string]interface{}

	apiErr, ok := err.(ApiError)
	if ok {
		rw.WriteHeader(apiErr.HTTPStatus)
	}
	rw.WriteHeader(http.StatusInternalServerError)
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

type NeedsMethods map[string][]NeedsMethod

type NeedsMethod struct {
	method       *ast.FuncDecl
	methodParams ParamCodegenMethod
}

type ParamCodegenMethod struct {
	Url        string `json:"url"`
	Auth       bool   `json:"auth"`
	Method     string `json:"method"`
	PapaStruct string
}

type NeedsValidateStructMap map[string]*ast.StructType
type OtherStructMap map[string]*ast.StructType

func main() {
	fset := token.NewFileSet()
	needsMethods := NeedsMethods{}
	needsValidateStructs := NeedsValidateStructMap{}
	otherStructMap := OtherStructMap{}

	node, err := parser.ParseFile(fset, filePatchIn, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := os.Create(filePatchOut)
	packageAndImportWrite(out, node)

	for _, decl := range node.Decls {
		switch {
		case needsMethods.addDecl(decl):
		case needsValidateStructs.addDecl(decl):
		default:
			otherStructMap.addDecl(decl)
		}
	}
	needsValidateStructs.structValidationWrite(out)
	needsMethods.MethodsWrapperWrite(out)
}

func packageAndImportWrite(out *os.File, node *ast.File) {
	fmt.Fprintln(out, `package `+node.Name.Name)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "import (")
	fmt.Fprintln(out, "\"encoding/json\"")
	fmt.Fprintln(out, "\"errors\"")
	fmt.Fprintln(out, "\"fmt\"")
	fmt.Fprintln(out, "\"net/http\"")
	fmt.Fprintln(out, "\"strconv\"")
	fmt.Fprintln(out, ")")
	fmt.Fprintln(out)
}

func (nm NeedsMethods) addDecl(decl interface{}) bool {
	src, _ := os.ReadFile(filePatchIn)
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

		paramCodegenMethod := ParamCodegenMethod{}
		json.Unmarshal([]byte(commentText), &paramCodegenMethod)
		paramCodegenMethod.PapaStruct = astFieldToString(src, g.Recv.List[0])
		nm[paramCodegenMethod.PapaStruct] = append(nm[paramCodegenMethod.PapaStruct], NeedsMethod{method: g, methodParams: paramCodegenMethod})
	}
	return true
}

func (nvs NeedsValidateStructMap) addDecl(decl interface{}) bool {
	g, ok := decl.(*ast.GenDecl)
	if !ok {
		return false
	}
	for _, spec := range g.Specs {
		currType, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		currStruct, ok := currType.Type.(*ast.StructType)
		if !ok {
			continue
		}
		for _, spec := range currStruct.Fields.List {
			if spec.Tag == nil {
				continue
			}
			if strings.Contains(spec.Tag.Value, "apivalidator") {
				nvs[currType.Name.Name] = currStruct
				return true
			}
		}
	}
	return false
}

func (nvs OtherStructMap) addDecl(decl interface{}) bool {
	g, ok := decl.(*ast.GenDecl)
	if !ok {
		return false
	}
	for _, spec := range g.Specs {
		currType, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		currStruct, ok := currType.Type.(*ast.StructType)
		if !ok {
			continue
		}
		nvs[currType.Name.Name] = currStruct
	}
	return false
}

func (nvs NeedsValidateStructMap) structValidationWrite(out *os.File) {
	for name, structDecl := range nvs {
		firstSymName := ""
		src, _ := os.ReadFile(filePatchIn)

		//узнаём первую букву структуры
		for i, ch := range name {
			if i == 0 {
				firstSymName = strings.ToLower(string(ch))
				break
			}
		}

		fmt.Fprintf(out, "func (%s *%s) FilingAndValidate(r *http.Request) error {\n", firstSymName, name)

		if structDecl == nil {
			return
		}

		for _, field := range structDecl.Fields.List {
			validatorText := strings.ReplaceAll(field.Tag.Value, "`", "")
			validatorText = strings.ReplaceAll(validatorText, "apivalidator:", "")
			validatorText = strings.ReplaceAll(validatorText, "\"", "")
			validatorLabls := strings.Split(validatorText, ",")

			for _, validatorLabl := range validatorLabls {
				fieldType := astFieldToString(src, field)

				//заполнение полей
				switch fieldType {
				case "int":
					fmt.Fprintf(out, "%s.%s,_ = strconv.Atoi(r.FormValue(\"%s\"))\n", firstSymName, field.Names[0], strings.ToLower(field.Names[0].Name))
				case "string":
					fmt.Fprintf(out, "%s.%s = r.FormValue(\"%s\")\n", firstSymName, field.Names[0], strings.ToLower(field.Names[0].Name))
				}

				//разбор меток валидатора
				if strings.Contains(validatorLabl, validatorLabelParamname) {
					paramName := strings.ReplaceAll(validatorLabl, validatorLabelParamname, "")

					switch fieldType {
					case "int":
						fmt.Fprintf(out, "%s.%s,_ = strconv.Atoi(r.FormValue(\"%s\"))\n", firstSymName, field.Names[0], paramName)
					case "string":
						fmt.Fprintf(out, "%s.%s = r.FormValue(\"%s\")\n", firstSymName, field.Names[0], paramName)
					}
				}

				if strings.Contains(validatorLabl, validatorLabelEnum) {
					params := strings.ReplaceAll(validatorLabl, validatorLabelEnum, "")
					paramNames := strings.Split(params, "|")

					for _, paramname := range paramNames {
						switch fieldType {
						case "string":
							fmt.Fprintf(out, "if %s.%s == \"\"{\n", firstSymName, field.Names[0])
							fmt.Fprintf(out, "%s.%s = r.FormValue(\"%s\")\n", firstSymName, field.Names[0], paramname)
							fmt.Fprintln(out, "}")
						case "int":
							fmt.Fprintf(out, "if %s.%s == \"\"{\n", firstSymName, field.Names[0])
							fmt.Fprintf(out, "%s.%s = strconv.Atoi(r.FormValue(\"%s\"))\n", firstSymName, field.Names[0], paramname)
							fmt.Fprintln(out, "}")
						}
					}
				}

				if strings.Contains(validatorLabl, validatorLabelDefault) {
					defaultParam := strings.ReplaceAll(validatorLabl, validatorLabelDefault, "")
					switch fieldType {
					case "string":
						fmt.Fprintf(out, "if %s.%s == \"\"{\n", firstSymName, field.Names[0])
						fmt.Fprintf(out, "%s.%s = \"%s\"\n", firstSymName, field.Names[0], defaultParam)
						fmt.Fprintln(out, "}")
					case "int":
						fmt.Fprintf(out, "if %s.%s == 0{\n", firstSymName, field.Names[0])
						fmt.Fprintf(out, "%s.%s = %s\n", firstSymName, field.Names[0], defaultParam)
						fmt.Fprintln(out, "}")
					}
				}

				if strings.Contains(validatorLabl, validatorLabelMin) {
					param := strings.ReplaceAll(validatorLabl, validatorLabelMin, "")
					switch fieldType {
					case "string":
						fmt.Fprintf(out, "if len(%s.%s) < %s{\n", firstSymName, field.Names[0], param)
						fmt.Fprintf(out, "return errors.New(\"validate error\")\n")
						fmt.Fprintln(out, "}")
					case "int":
						fmt.Fprintf(out, "if %s.%s < %s{\n", firstSymName, field.Names[0], param)
						fmt.Fprintf(out, "return errors.New(\"validate error\")\n")
						fmt.Fprintln(out, "}")
					}
				}

				if strings.Contains(validatorLabl, validatorLabelMax) {
					param := strings.ReplaceAll(validatorLabl, validatorLabelMax, "")
					switch fieldType {
					case "string":
						fmt.Fprintf(out, "if len(%s.%s) > %s{\n", firstSymName, field.Names[0], param)
						fmt.Fprintf(out, "return errors.New(\"validate error\")\n")
						fmt.Fprintln(out, "}")
					case "int":
						fmt.Fprintf(out, "if %s.%s > %s{\n", firstSymName, field.Names[0], param)
						fmt.Fprintf(out, "return errors.New(\"validate error\")\n")
						fmt.Fprintln(out, "}")
					}
				}

				if strings.Contains(validatorLabl, validatorLabelRequired) {
					switch fieldType {
					case "string":
						fmt.Fprintf(out, "if %s.%s == \"\"{\n", firstSymName, field.Names[0])
						fmt.Fprintf(out, "return errors.New(\"validate error\")\n")
						fmt.Fprintln(out, "}")
					case "int":
						fmt.Fprintf(out, "if %s.%s == 0{\n", firstSymName, field.Names[0])
						fmt.Fprintf(out, "return errors.New(\"validate error\")\n")
						fmt.Fprintln(out, "}")
					}
				}
			}
		}

		fmt.Fprintln(out, "return nil")
		fmt.Fprintln(out, "}")
		fmt.Fprintln(out)
	}
}

func (nm NeedsMethods) MethodsWrapperWrite(out *os.File) {
	src, _ := os.ReadFile(filePatchIn)
	//http serve generate
	for papaStruct, method := range nm {
		firstSymName := ""
		for _, ch := range papaStruct {
			if ch != '*' {
				firstSymName = strings.ToLower(string(ch))
				break
			}
		}

		fmt.Fprintf(out, "func (%s %s) ServeHTTP(rw http.ResponseWriter, r *http.Request) {\n", firstSymName, papaStruct)
		fmt.Fprintln(out, "\tswitch r.URL.Path {")

		for _, m := range method {
			fmt.Fprintf(out, "\tcase \"%s\":\n", m.methodParams.Url)
			fmt.Fprintf(out, "\t\t%s.%s(rw, r)\n", firstSymName, strings.ToLower(m.method.Name.Name))
		}

		fmt.Fprintln(out, "\tdefault:")
		fmt.Fprintln(out, "\t\trw.WriteHeader(http.StatusNotFound)")
		fmt.Fprintln(out, "\t\tresponseError(rw, errors.New(\"unknown method\"))")
		fmt.Fprintln(out, "\t}")
		fmt.Fprintln(out, "}")
		fmt.Fprintln(out)
	}
	//methods generate
	for papaStruct, method := range nm {
		firstSymName := ""

		for _, ch := range papaStruct {
			if ch != '*' {
				firstSymName = strings.ToLower(string(ch))
				break
			}
		}

		for _, m := range method {
			fmt.Fprintf(out, "func(%s %s) %s(rw http.ResponseWriter, r * http.Request) {\n", firstSymName, m.methodParams.PapaStruct, strings.ToLower(m.method.Name.Name))
			methodParamsSlice := make([]string, 0, 2)
			methodParamsString := ""

			for _, params := range m.method.Type.Params.List {
				variableName := strings.ToLower(strings.ReplaceAll(astFieldToString(src, params), ".", ""))

				if variableName == "contextcontext" {
					methodParamsSlice = append(methodParamsSlice, "nil")
					continue
				}
				methodParamsSlice = append(methodParamsSlice, variableName)

				fmt.Fprintf(out, "%s := %s{}\n", variableName, astFieldToString(src, params))
				fmt.Fprintf(out, "if err := %s.FilingAndValidate(r); err != nil {\n", variableName)
				fmt.Fprintln(out, "responseError(rw,err)")
				fmt.Fprintln(out, "return")
				fmt.Fprintln(out, "}")
			}

			for _, param := range methodParamsSlice {
				if methodParamsString != "" {
					methodParamsString += ","
				}
				methodParamsString += param
			}

			fmt.Fprintf(out, "response, err := %s.%s(%s)\n", firstSymName, m.method.Name.Name, methodParamsString)
			fmt.Fprintln(out, "if err != nil {")
			fmt.Fprintln(out, "responseError(rw, err)")
			fmt.Fprintln(out, "return")
			fmt.Fprintln(out, "}")
			fmt.Fprintln(out, "responseResult(rw, err, response)")
			fmt.Fprintln(out, "}")

			//responseResult(rw, err, user)
		}
	}

	fmt.Fprintf(out, responseErrorFuncRaw)
	fmt.Fprintln(out)
	fmt.Fprintln(out)
	fmt.Fprintf(out, responseResultFuncRaw)

}

func astFieldToString(src []byte, field *ast.Field) string {
	typeExpr := field.Type
	start := typeExpr.Pos() - 1
	end := typeExpr.End() - 1
	return string(src)[start:end]
}
